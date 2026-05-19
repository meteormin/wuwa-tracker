package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/meteormin/wuwa-tracker/internal/server/handlers"
	"github.com/meteormin/wuwa-tracker/internal/server/models"
)

func main() {
	// 데이터베이스 저장 디렉터리 경로 지정 (기본값: data/wuwa_badger)
	dbDir := "data/wuwa_badger"
	if dirVal := os.Getenv("DATABASE_DIR"); dirVal != "" {
		dbDir = dirVal
	}

	// BadgerDB 네이티브 KV 엔진 초기화
	db, err := models.InitDB(dbDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
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

	// API 라우팅 설정
	api := app.Group("/api")
	api.Post("/track", handlers.TrackHandler(db))
	api.Get("/stats/:playerId", handlers.GetStatsHandler(db))
	api.Get("/players", handlers.ListPlayersHandler(db))

	// 프론트엔드 Svelte 정적 빌드본 호스팅 (production 빌드 대응)
	if _, err := os.Stat("./webui/dist"); err == nil {
		app.Static("/", "./webui/dist")
		// SPA 라우팅을 지원하기 위해 매치되지 않는 요청은 index.html로 폴백
		app.Get("/*", func(c *fiber.Ctx) error {
			return c.SendFile("./webui/dist/index.html")
		})
		log.Println("Serving production frontend from ./webui/dist")
	} else {
		log.Println("Frontend build directory './webui/dist' not found. Backend API mode active.")
	}

	// 서버 포트 설정 및 구동
	port := "8080"
	if portVal := os.Getenv("PORT"); portVal != "" {
		port = portVal
	}

	log.Printf("Starting backend server on port %s...\n", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}
