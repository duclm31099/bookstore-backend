package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bookstore-backend/pkg/container"
)

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
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
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
