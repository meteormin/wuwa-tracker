package main

import (
	"flag"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3/log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/server/db"
	"github.com/meteormin/wuwa-tracker/internal/server/handlers"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/webui"
)

const banner = `

wuwa-tracker

`

var (
	appName       = "wuwa-tracker"
	defaultPort   = "3000"
	defaultDBPath = "data/wuwa_badger"
)

func main() {
	// 기본값 정의 및 환경변수(PORT, DB_PATH) 폴백 설정
	if envPort := os.Getenv("PORT"); envPort != "" {
		defaultPort = envPort
	}

	if envDBPath := os.Getenv("DB_PATH"); envDBPath != "" {
		defaultDBPath = envDBPath
	}

	// CLI 플래그 파싱 정의
	portFlag := flag.String("port", defaultPort, "Port to listen on")
	dbPathFlag := flag.String("dbpath", defaultDBPath, "BadgerDB storage directory")
	flag.Parse()

	port := *portFlag
	dbPath := *dbPathFlag

	// 설정 로드 (서버 기동 시 최초 1회만 로드하여 메모리에 적재)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 다국어 배너 이름 사전 매핑 (최초 1회 한국어로 캐싱 매핑 수행)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	client := tracker.NewClient(httpClient)
	localeData, err := client.FetchGachaLocale("ko")
	if err != nil {
		log.Warnf("Failed to fetch remote 'ko' banner locale on startup: %v. Using defaults.\n", err)
		localeData = types.LocaleData{SelectList: map[string]string{}}
	}
	cfg.GachaTypes.MapFromSelectList(localeData.SelectList)

	// BadgerDB 네이티브 KV 엔진 초기화
	badgerDB, err := db.NewBadgerDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := badgerDB.Close(); err != nil {
			log.Errorf("Failed to close database: %v\n", err)
		} else {
			log.Info("Database connection closed cleanly.")
		}
	}()
	log.Infof("Successfully started BadgerDB engine under directory: %s\n", dbPath)

	// Fiber v3 애플리케이션 생성
	app := fiber.New(fiber.Config{
		AppName: appName,
	})

	// 스타트업 훅 시스템을 사용한 소문자 아스키 배너 적용
	app.Hooks().OnPreStartupMessage(func(sm *fiber.PreStartupMessageData) error {
		sm.BannerHeader = banner
		return nil
	})

	// 공통 미들웨어 등록
	app.Use(recover.New())
	app.Use(logger.New())

	// Svelte 로컬 개발 환경과의 통신을 위해 CORS 활성화
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
	}))

	// 핸들러 인스턴스 생성 및 의존성(DB, Config) 주입
	h := handlers.NewHandler(badgerDB, cfg)

	// API 라우팅 설정
	api := app.Group("/api")
	api.Post("/track", h.Track)
	api.Get("/stats/:playerId", h.GetStats)
	api.Get("/players", h.ListPlayers)
	api.Post("/upload", h.Upload)
	api.Get("/config", h.GetConfig)

	// 1. 빌드된 WebUI 정적 자원들을 Go embed 파일 시스템으로 내장 호스팅 (Fiber v3 static 미들웨어 적용)
	subFS, err := fs.Sub(webui.DistFS, "dist")
	if err != nil {
		log.Fatalf("Failed to create sub-FS for embedded WebUI: %v", err)
	}

	app.Use("/", static.New("", static.Config{
		FS:     subFS,
		Browse: false,
	}))

	// 2. SPA 클라이언트 라우팅 대응 (API가 아닌 잘못된 모든 경로는 index.html로 폴백)
	app.Get("/*", func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "API endpoint not found",
			})
		}

		indexFile, err := webui.DistFS.ReadFile("dist/index.html")
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Embedded frontend index.html not found")
		}

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(indexFile)
	})

	// Graceful shutdown 지원을 위한 시그널 리스너 채널 생성
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// 서버 수신 리스너를 고루틴으로 실행하여 차단 회피
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Errorf("Server listen ended: %v\n", err)
		}
	}()

	// 종료 시그널 수신 대기
	<-stop
	log.Info("Shutting down server gracefully...")

	// 1. Fiber v3 웹 서버 종료 (신규 연결 요청 차단 및 처리 중인 기존 커넥션 대기)
	if err := app.Shutdown(); err != nil {
		log.Errorf("Error shutting down Fiber server: %v\n", err)
	}
}
