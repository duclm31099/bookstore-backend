package main

import (
	"context"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// asynqServer wraps asynq.Server with additional functionality
type asynqServer struct {
	*asynq.Server
}

// setupAsynqServer creates and configures the Asynq server
func setupAsynqServer(cfg *Config, handlers *HandlerRegistry) *asynqServer {
	// Create ServeMux
	mux := asynq.NewServeMux()

	// Register all handlers
	handlers.RegisterHandlers(mux)
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Queues: map[string]int{
				"high":         20, // 40% - Orders, payments
				"default":      8,  // 16% - General tasks
				"low":          5,  // 10% - Analytics
				"notification": 17, // 34% - ✅ TĂNG LÊN vì notification quan trọng
			},
			Concurrency: 30, // ✅ TĂNG LÊN nếu có nhiều CPU cores
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				// ✅ THÊM logging chi tiết hơn
				log.Printf("[Asynq] ❌ Task failed - Type: %s, TaskID: %s, Error: %v",
					task.Type(),
					task.ResultWriter().TaskID(),
					err,
				)
			}),
			// ✅ THÊM retry delay
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				// Exponential backoff: 1m, 2m, 4m, 8m, ...
				return time.Duration(1<<uint(n)) * time.Minute
			},
		},
	)

	// Start server in goroutine
	go func() {
		log.Println("[Worker] Starting...")
		if err := srv.Run(mux); err != nil {
			log.Fatalf("[Worker] Failed: %v", err)
		}
	}()

	return &asynqServer{Server: srv}
}

// Shutdown gracefully shuts down the server with timeout
func (s *asynqServer) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("[Worker] Shutting down (waiting max 30s)...")
	s.Server.Shutdown()

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		log.Println("[Worker] ⚠️ Shutdown timeout exceeded")
	} else {
		log.Println("[Worker] ✓ Gracefully stopped")
	}
}
