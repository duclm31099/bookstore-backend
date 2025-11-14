package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"bookstore-backend/internal/infrastructure/email"
	"bookstore-backend/internal/infrastructure/queue/handlers"
	"bookstore-backend/internal/shared/utils"

	"github.com/hibiken/asynq"
)

func main() {
	// Đọc Redis config từ environment
	redisAddr := utils.GetEnvVariable("REDIS_HOST", "localhost:6379")

	// Đọc SMTP config từ environment
	// ✅ Default phải là "mailhog" (service name trong docker-compose)
	smtpHost := utils.GetEnvVariable("SMTP_HOST", "localhost")
	smtpPort := utils.GetEnvVariable("SMTP_PORT", "1025")

	log.Printf("[Asynq] Starting worker with Redis: %s, SMTP: %s:%s",
		redisAddr, smtpHost, smtpPort)

	// Khởi tạo email service
	emailSvc := email.NewDevEmailService(smtpHost, smtpPort)

	// Đăng ký task handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc("email:verification", handlers.EmailVerificationHandler(emailSvc))
	mux.HandleFunc("email:reset_password", handlers.EmailResetPasswordHandler(emailSvc))

	// Khởi tạo Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Queues:      map[string]int{"high": 20, "default": 10, "low": 5},
			Concurrency: 20,
		},
	)

	// Graceful shutdown
	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("[Asynq] Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[Asynq] Shutting down worker...")
	srv.Shutdown()
	log.Println("[Asynq] Worker stopped")
}
