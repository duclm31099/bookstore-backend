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

	"bookstore-backend/internal/config"
	"bookstore-backend/internal/infrastructure/cache"
	"bookstore-backend/internal/infrastructure/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Global variables để dễ access trong các handlers
var (
	db          *database.PostgresDB
	redisClient *cache.RedisClient // ← ADD THIS
)

func main() {
	// Load environment variables từ .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Set Gin mode theo environment
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// ========================================
	// INITIALIZE DATABASE
	// ========================================
	log.Println("📦 Initializing dependencies...")

	// Load database config từ environment
	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load database config: %v", err)
	}

	// Tạo PostgresDB instance
	db = database.NewPostgresDB(dbConfig)

	// Connect to database với context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	// Verify connection với health check
	if err := db.HealthCheck(context.Background()); err != nil {
		log.Fatalf("❌ Database health check failed: %v", err)
	}

	log.Println("✅ Database connected successfully")
	redisClient = cache.NewRedisClient(
		getEnv("REDIS_HOST", "localhost:6379"),
		getEnv("REDIS_PASSWORD", ""),
		0, // DB number
	)

	if err := redisClient.Connect(context.Background()); err != nil {
		log.Printf("⚠️  Redis connection failed (non-critical): %v", err)
	} else {
		log.Println("✅ Redis connected successfully")
	}

	// TODO: Initialize Redis, other services...

	// ========================================
	// SETUP ROUTER
	// ========================================
	router := setupRouter()

	// ========================================
	// CONFIGURE SERVER
	// ========================================
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%s", getEnv("APP_PORT", "8080")),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// ========================================
	// START SERVER
	// ========================================
	go func() {
		log.Printf("🚀 Server starting on port %s", getEnv("APP_PORT", "8080"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// ========================================
	// GRACEFUL SHUTDOWN
	// ========================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Shutdown server với timeout
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	}

	// Close database connections
	if db != nil && db.Pool != nil {
		log.Println("🔌 Closing database connections...")
		db.Pool.Close()
	}

	log.Println("✅ Server exited gracefully")
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Bookstore API",
			"version": getEnv("APP_VERSION", "1.0.0"),
			"status":  "running",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint với REAL database check
		v1.GET("/health", healthCheckHandler)

		// Database test endpoint
		v1.GET("/db-test", databaseTestHandler)

		// TODO: Add more route groups
		// auth := v1.Group("/auth")
		// books := v1.Group("/books")
	}

	return router
}

// healthCheckHandler - Enhanced health check với real database ping
func healthCheckHandler(c *gin.Context) {
	health := gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   getEnv("APP_VERSION", "1.0.0"),
		"services":  gin.H{},
	}

	// Check database
	dbStatus := "ok"
	if db == nil || db.Pool == nil {
		dbStatus = "disconnected"
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.HealthCheck(ctx); err != nil {
			dbStatus = fmt.Sprintf("error: %v", err)
			health["status"] = "degraded"
		}
	}

	redisStatus := "ok"
	if redisClient == nil {
		redisStatus = "disconnected"
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := redisClient.HealthCheck(ctx); err != nil {
			redisStatus = fmt.Sprintf("error: %v", err)
			health["status"] = "degraded"
		}
	}

	health["services"] = gin.H{
		"database": dbStatus,
		"redis":    redisStatus, // ← Updated
	}

	// Return 503 if database is down
	statusCode := http.StatusOK
	if dbStatus != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// databaseTestHandler - Test endpoint để verify database queries
func databaseTestHandler(c *gin.Context) {
	if db == nil || db.Pool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Database not connected",
		})
		return
	}

	// Test query: Get PostgreSQL version
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var version string
	err := db.Pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Query failed: %v", err),
		})
		return
	}

	// Get pool statistics
	stats := db.Pool.Stat()

	c.JSON(http.StatusOK, gin.H{
		"message":          "Database test successful",
		"postgres_version": version,
		"pool_stats": gin.H{
			"total_connections":    stats.TotalConns(),
			"idle_connections":     stats.IdleConns(),
			"acquired_connections": stats.AcquiredConns(),
		},
	})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
