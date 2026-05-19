package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/meteormin/wuwa-tracker/internal/server/db"
	"github.com/meteormin/wuwa-tracker/internal/server/handlers"
	"github.com/meteormin/wuwa-tracker/webui"
)

func main() {
	// 기본값 정의 및 환경변수(PORT, DB_PATH) 폴백 설정
	defaultPort := "3000"
	if envPort := os.Getenv("PORT"); envPort != "" {
		defaultPort = envPort
	}

	defaultDBDir := "data/wuwa_badger"
	if envDBDir := os.Getenv("DB_PATH"); envDBDir != "" {
		defaultDBDir = envDBDir
	}

	// CLI 플래그 파싱 정의
	portFlag := flag.String("port", defaultPort, "Port to listen on")
	dbDirFlag := flag.String("dbpath", defaultDBDir, "BadgerDB storage directory")
	flag.Parse()

	port := *portFlag
	dbDir := *dbDirFlag

	// BadgerDB 네이티브 KV 엔진 초기화
	badgerDB, err := db.NewBadgerDB(dbDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := badgerDB.Close(); err != nil {
			log.Printf("Error closing database: %v\n", err)
		} else {
			log.Println("Database connection closed cleanly.")
		}
	}()
	log.Printf("Successfully started BadgerDB engine under directory: %s\n", dbDir)

	// Fiber 애플리케이션 생성
	app := fiber.New(fiber.Config{
		AppName: "Wuwa Tracker Server",
	})

	// 공통 미들웨어 등록
	app.Use(recover.New())
	app.Use(logger.New())

	// Svelte 로컬 개발 환경(포트 5173 등)과의 통신을 위해 CORS 활성화
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	// 핸들러 인스턴스 생성 및 의존성 주입
	h := handlers.NewHandler(badgerDB)

	// API 라우팅 설정
	api := app.Group("/api")
	api.Post("/track", h.Track)
	api.Get("/stats/:playerId", h.GetStats)
	api.Get("/players", h.ListPlayers)

	// 1. 빌드된 WebUI 정적 자원들을 Go embed 파일 시스템으로 내장 호스팅
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(webui.DistFS),
		PathPrefix: "dist",
		Browse:     false,
	}))

	// 2. SPA 클라이언트 라우팅 대응 (API가 아닌 잘못된 모든 경로는 index.html로 폴백)
	app.Get("/*", func(c *fiber.Ctx) error {
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

	log.Printf("Starting embedded single-binary backend server on port %s...\n", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}
