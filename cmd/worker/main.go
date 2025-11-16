// cmd/worker/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"bookstore-backend/pkg/container"
)

func main() {
	// Initialize container
	c, err := container.NewContainer()
	if err != nil {
		log.Fatalf("[Container] Failed to initialize: %v", err)
	}
	defer c.Cleanup()

	// Load configuration
	cfg := loadConfig()

	// Initialize handlers
	handlers := initializeHandlers(c, cfg)

	// Setup Asynq server
	srv := setupAsynqServer(cfg, handlers)

	// Setup scheduler
	scheduler := setupScheduler(cfg)

	// ✅ Perform health checks and log startup
	if err := startServices(srv, scheduler, cfg); err != nil {
		log.Fatalf("[Startup] Health check failed: %v", err)
	}

	// Wait for shutdown signal
	waitForShutdown(srv, scheduler)
}

func waitForShutdown(srv *asynqServer, scheduler *asynqScheduler) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[Shutdown] Gracefully stopping...")
	scheduler.Shutdown()
	srv.Shutdown()
	log.Println("[Shutdown] ✓ Stopped")
}
