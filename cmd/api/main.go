package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// ========================================
	// LOAD ENVIRONMENT VARIABLES
	// ========================================
	// Load từ .env file (development/local)
	// Production sẽ dùng system environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// ========================================
	// SET GIN MODE
	// ========================================
	// Tùy theo APP_ENV: development (debug logs) hoặc production (optimize)
	env := getEnv("APP_ENV", "development")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Printf("🌍 Environment: %s", env)

	// ========================================
	// START SERVER
	// ========================================
	// Delegate toàn bộ logic sang Serve()
	// Giữ main() gọn gàng, chỉ làm entry point
	Serve()
}

// getEnv lấy environment variable với fallback default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
