package main

import (
	"log"

	"bookstore-backend/internal/config"
	"bookstore-backend/internal/infrastructure/queue"
)

// asynqScheduler wraps queue.Scheduler with additional functionality
type asynqScheduler struct {
	*queue.Scheduler
}

// setupScheduler creates and configures the scheduler
func setupScheduler(cfg *Config, jobConfig config.JobConfig) *asynqScheduler {
	scheduler := queue.NewScheduler(cfg.RedisAddr, jobConfig)

	// Register cron jobs
	if err := scheduler.RegisterCleanupJobs(); err != nil {
		log.Fatalf("[Scheduler] Failed to register: %v", err)
	}

	// Start scheduler in goroutine
	go func() {
		log.Println("[Scheduler] Starting...")
		if err := scheduler.Start(); err != nil {
			log.Fatalf("[Scheduler] Failed: %v", err)
		}
	}()

	return &asynqScheduler{Scheduler: scheduler}
}

// Shutdown gracefully shuts down the scheduler
func (s *asynqScheduler) Shutdown() {
	log.Println("[Scheduler] Shutting down...")
	s.Scheduler.Shutdown()
	log.Println("[Scheduler] ✓ Stopped")
}
