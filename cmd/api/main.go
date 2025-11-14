package main

import (
	"bookstore-backend/pkg/container"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func Serve() {
	// ========================================
	// 1. BUILD DI CONTAINER
	// ========================================
	// Container tự động initialize toàn bộ dependencies
	// Nếu có lỗi → application không start
	appContainer, err := container.NewContainer()
	if err != nil {
		log.Fatalf("❌ Failed to initialize container: %v", err)
	}

	// Ensure cleanup on shutdown
	defer appContainer.Cleanup()

	// ========================================
	// 2. SETUP ROUTER
	// ========================================
	// Router nhận container để access handlers
	router := SetupRouter(appContainer)

	// ========================================
	// 3. CONFIGURE HTTP SERVER
	// ========================================
	port := appContainer.Config.App.Port
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// ========================================
	// 4. START SERVER (NON-BLOCKING)
	// ========================================
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%s", port)
		log.Printf("📚 Environment: %s", appContainer.Config.App.Environment)
		log.Printf("💚 Health Check: http://localhost:%s/api/v1/health", port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// ========================================
	// 5. GRACEFUL SHUTDOWN
	// ========================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}
