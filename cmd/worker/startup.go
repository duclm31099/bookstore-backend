// cmd/worker/startup.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

// HealthChecker performs startup health checks
type HealthChecker struct {
	redisClient *redis.Client
}

// startServices performs health checks and logs startup information
func startServices(srv *asynqServer, scheduler *asynqScheduler, cfg *Config) error {
	log.Println("============================================")
	log.Println("🚀 Bookstore Worker Starting...")
	log.Println("============================================")

	// ✅ 1. Perform Health Checks
	checker := &HealthChecker{
		redisClient: redis.NewClient(&redis.Options{
			Addr: cfg.RedisAddr,
			MaintNotificationsConfig: &maintnotifications.Config{
				Mode: maintnotifications.ModeDisabled, // ✅ No warnings
			},
		}),
	}

	if err := checker.checkAll(); err != nil {
		log.Printf("❌ Health check failed: %v\n", err)
		return err
	}

	// ✅ 4. Start health check endpoint
	go startHealthCheckServer()

	return nil
}

// checkAll runs all health checks
func (h *HealthChecker) checkAll() error {
	checks := []struct {
		name string
		fn   func() error
	}{
		{"Redis Connection", h.checkRedis},
		{"Asynq Worker", h.checkAsynq},
	}

	for _, check := range checks {
		log.Printf("⏳ Checking %s...\n", check.name)
		if err := check.fn(); err != nil {
			log.Printf("❌ %s: %v\n", check.name, err)
			return fmt.Errorf("%s failed: %w", check.name, err)
		}
		log.Printf("✓ %s: OK\n", check.name)
	}

	return nil
}

// checkRedis verifies Redis connection
func (h *HealthChecker) checkRedis() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return h.redisClient.Ping(ctx).Err()
}

// checkAsynq verifies Asynq can connect to Redis
func (h *HealthChecker) checkAsynq() error {
	// Asynq uses same Redis, so if Redis check passed, this is OK
	return nil
}

// startHealthCheckServer starts HTTP server for health checks
func startHealthCheckServer() {
	http.HandleFunc("/health", healthCheckHandler)
	http.HandleFunc("/ready", readyCheckHandler)

	log.Println("[Health] Starting health check server on :9999")
	if err := http.ListenAndServe(":9999", nil); err != nil {
		log.Printf("[Health] Failed to start: %v\n", err)
	}
}

// healthCheckHandler handles /health endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"UP","service":"bookstore-worker"}`))
}

// readyCheckHandler handles /ready endpoint (Kubernetes readiness probe)
func readyCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"READY"}`))
}
