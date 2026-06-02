package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3/log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/handler"
	"github.com/meteormin/wuwa-tracker/internal/server"
	"github.com/meteormin/wuwa-tracker/webui"
)

const banner = `

wuwa-tracker %s

`

var (
	appName  = "wuwa-tracker"
	buildTag = "dev"
)

func main() {
	if err := run(); err != nil {
		log.Errorf("Server error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// 설정 로드 (서버 기동 시 최초 1회만 로드하여 메모리에 적재)
	cfg := config.Default()

	// CLI 플래그 파싱 정의
	hostFlag := flag.String("host", cfg.ServerHost, "Host address to listen on")
	portFlag := flag.String("port", cfg.ServerPort, "Port to listen on")
	dbPathFlag := flag.String("dbpath", cfg.DBPath, "BadgerDB storage directory")
	dbGCEnabledFlag := flag.Bool("db-gc", cfg.DBGCEnabled, "Enable periodic Badger value log GC")
	dbGCIntervalFlag := flag.Duration("db-gc-interval", cfg.DBGCInterval, "Badger value log GC interval")
	dbGCDiscardRatioFlag := flag.Float64("db-gc-discard-ratio", cfg.DBGCDiscardRatio, "Badger value log GC discard ratio")
	flag.Parse()

	cfg.ServerHost = *hostFlag
	cfg.ServerPort = *portFlag
	cfg.DBPath = *dbPathFlag
	cfg.DBGCEnabled = *dbGCEnabledFlag
	cfg.DBGCInterval = *dbGCIntervalFlag
	cfg.DBGCDiscardRatio = *dbGCDiscardRatioFlag

	log.Infof("Build tag: %s\n", buildTag)
	log.Infof("Configured BadgerDB directory: %s\n", cfg.DBPath)

	// Fiber v3 애플리케이션 생성
	runtime := server.NewRuntimeService(cfg)
	app := fiber.New(fiber.Config{
		AppName:  appName,
		Services: []fiber.Service{runtime},
	})
	defer func() {
		_ = runtime.Terminate(context.Background())
	}()

	// 스타트업 훅 시스템을 사용한 소문자 아스키 배너 적용
	app.Hooks().OnPreStartupMessage(func(sm *fiber.PreStartupMessageData) error {
		sm.BannerHeader = fmt.Sprintf(banner, buildTag)
		return nil
	})

	// 공통 미들웨어 등록
	app.Use(recover.New())
	app.Use(logger.New())

	// Svelte 로컬 개발 환경과의 통신을 위해 CORS 활성화
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
	}))

	// 핸들러 인스턴스 생성
	h := handler.NewHandler()

	// API 라우팅 설정
	api := app.Group("/api")
	api.Post("/scan", h.Scan)
	api.Post("/track", h.Track)
	api.Get("/stats/:playerId", h.GetStats)
	api.Get("/players", h.ListPlayers)
	api.Post("/upload", h.Upload)
	api.Get("/config", h.GetConfig)
	api.Get("/i18n", h.GetI18n)
	api.Get("/export/:playerId", h.ExportReport)

	// 빌드된 WebUI 정적 자원들을 Go embed 파일 시스템으로 내장 호스팅
	app.Use("/", static.New("", static.Config{
		FS:     webui.FS,
		Browse: false,
	}))

	// Graceful shutdown 지원을 위한 시그널 기반 컨텍스트 생성
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := net.JoinHostPort(cfg.ServerHost, cfg.ServerPort)
	// 서버 수신 리스너를 동기식으로 실행하여 GracefulContext 주입
	if err := app.Listen(addr, fiber.ListenConfig{
		GracefulContext: ctx,
	}); err != nil {
		return err
	}

	log.Info("Server stopped.")

	return nil
}
