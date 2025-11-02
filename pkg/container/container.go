package container

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"bookstore-backend/internal/config"
	infraCache "bookstore-backend/internal/infrastructure/cache"
	"bookstore-backend/internal/infrastructure/database"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/jwt"

	// User domain imports
	"bookstore-backend/internal/domains/user"
	userHandler "bookstore-backend/internal/domains/user/handler"
	userRepo "bookstore-backend/internal/domains/user/repository"
	userService "bookstore-backend/internal/domains/user/service"

	// Category domain imports ◄── NEW
	category "bookstore-backend/internal/domains/category"
	categoryHandler "bookstore-backend/internal/domains/category/handler"
	categoryRepo "bookstore-backend/internal/domains/category/repository"
	categoryService "bookstore-backend/internal/domains/category/service"
	// TODO: Import other domains khi implement
	// "bookstore/internal/domains/book"
	// bookHandler "bookstore/internal/domains/book/handler"
)

// ========================================
// CONTAINER STRUCT
// ========================================

// Container chứa TẤT CẢ dependencies của application
// Struct này là "root" của dependency graph
// Pattern: Service Locator + Dependency Injection
type Container struct {
	// ========================================
	// INFRASTRUCTURE LAYER
	// ========================================
	// Infrastructure components - shared across all domains
	// Lifecycle: Singleton (1 instance duy nhất trong app lifetime)

	Config     *config.Config       // Application config
	DB         *database.PostgresDB // Database connection pool
	Cache      cache.Cache          // Redis cache (interface)
	JWTManager *jwt.Manager

	// ========================================
	// REPOSITORY LAYER (DATA ACCESS)
	// ========================================
	UserRepo     user.Repository
	CategoryRepo category.CategoryRepository

	// ========================================
	// SERVICE LAYER (BUSINESS LOGIC)
	// ========================================

	UserService     user.Service
	CategoryService category.CategoryService

	// ========================================
	// HANDLER LAYER (HTTP)
	// ========================================
	UserHandler     *userHandler.UserHandler
	CategoryHandler *categoryHandler.CategoryHandler
}

// ========================================
// CONSTRUCTOR: BUILD CONTAINER
// ========================================
// NewContainer tạo và initialize toàn bộ dependency graph
// Đây là entry point của DI container
// QUAN TRỌNG: Thứ tự initialization:
// 1. Config (không phụ thuộc gì)
// 2. Infrastructure (DB, Cache) - phụ thuộc Config
// 3. Repositories - phụ thuộc Infrastructure
// 4. Services - phụ thuộc Repositories
// 5. Handlers - phụ thuộc Services
//
// Nếu thứ tự sai → panic (nil pointer dereference)
func NewContainer() (*Container, error) {
	log.Println("🔧 Initializing DI Container...")

	// Tạo empty container
	// Các fields sẽ được populate dần theo thứ tự
	c := &Container{}

	// ========================================
	// STEP 1: LOAD CONFIGURATION
	// ========================================
	// Config không phụ thuộc vào ai - tạo đầu tiên
	log.Println("📋 Loading configuration...")

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	c.Config = cfg
	log.Printf("✅ Config loaded (Environment: %s)", cfg.App.Environment)

	// ========================================
	// STEP 2: INITIALIZE DATABASE
	// ========================================
	// Database phụ thuộc Config
	log.Println("🗄️  Connecting to PostgreSQL...")

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	db := database.NewPostgresDB(dbConfig)

	// Connect với timeout 30s
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Health check
	if err := db.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	c.DB = db
	log.Println("✅ Database connected")

	// ========================================
	// STEP 3: INITIALIZE CACHE
	// ========================================
	// Cache phụ thuộc Config
	log.Println("🔴 Connecting to Redis...")

	redisCache := infraCache.NewRedisCache(
		cfg.Redis.Host,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)

	// Connect Redis
	// Type assertion để gọi Connect method (không có trong interface)
	if rc, ok := redisCache.(*infraCache.RedisCache); ok {
		if err := rc.Connect(context.Background()); err != nil {
			// Redis failure không critical - log warning và continue
			log.Printf("⚠️  Redis connection failed (non-critical): %v", err)
		} else {
			log.Println("✅ Redis connected")
		}
	}

	c.Cache = redisCache

	jwtSecret := cfg.JWT.Secret // Use from config
	c.JWTManager = jwt.NewManager(jwtSecret)

	// ========================================
	// STEP 4: INITIALIZE REPOSITORIES
	// ========================================
	// Repositories phụ thuộc DB và Cache
	log.Println("📦 Initializing repositories...")

	if err := c.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to init repositories: %w", err)
	}
	log.Println("✅ Repositories initialized")

	// ========================================
	// STEP 5: INITIALIZE SERVICES
	// ========================================
	// Services phụ thuộc Repositories và Config
	log.Println("⚙️  Initializing services...")

	if err := c.initServices(); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}
	log.Println("✅ Services initialized")

	// ========================================
	// STEP 6: INITIALIZE HANDLERS
	// ========================================
	// Handlers phụ thuộc Services
	log.Println("🎯 Initializing handlers...")

	if err := c.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to init handlers: %w", err)
	}
	log.Println("✅ Handlers initialized")

	log.Println("🎉 DI Container initialized successfully")
	return c, nil
}

// ========================================
// PRIVATE INITIALIZATION METHODS
// ========================================
func (c *Container) initRepositories() error {
	// Chuẩn bị sql.DB từ pgxpool
	// userRepo.NewPostgresRepository cần *sql.DB, không phải *pgxpool.Pool
	pool := c.DB.Pool

	c.UserRepo = userRepo.NewPostgresRepository(pool, c.Cache)
	c.CategoryRepo = categoryRepo.NewPostgresRepository(pool, c.Cache)
	return nil
}

// initServices khởi tạo tất cả services
func (c *Container) initServices() error {
	c.UserService = userService.NewUserService(
		c.UserRepo,   // Inject repository
		c.JWTManager, // Inject JWT secret từ config
	)

	c.CategoryService = categoryService.NewCategoryService(c.CategoryRepo)
	return nil
}

// initHandlers khởi tạo tất cả HTTP handlers
func (c *Container) initHandlers() error {
	c.UserHandler = userHandler.NewUserHandler(c.UserService)
	c.CategoryHandler = categoryHandler.NewCategoryHandler(c.CategoryService)

	return nil
}

// ========================================
// HELPER METHODS
// ========================================
// getSQLDB convert pgxpool.Pool sang *sql.DB
// Một số libraries cần *sql.DB thay vì *pgxpool.Pool
func (c *Container) getSQLDB() *sql.DB {
	// Note: Đây là workaround
	// Nếu repository accept *pgxpool.Pool thì không cần method này
	// TODO: Refactor repository để dùng pgxpool.Pool directly

	// Tạm thời return nil, sẽ implement sau
	// Hoặc dùng stdlib/sql wrapper
	return nil // FIXME
}

// Cleanup dọn dẹp resources khi shutdown
// Gọi trong graceful shutdown của server
func (c *Container) Cleanup() {
	log.Println("🧹 Cleaning up container resources...")

	// Close database connections
	if c.DB != nil && c.DB.Pool != nil {
		c.DB.Pool.Close()
		log.Println("✅ Database connections closed")
	}

	// Close Redis connections
	if c.Cache != nil {
		if rc, ok := c.Cache.(*infraCache.RedisCache); ok {
			if err := rc.Close(); err != nil {
				log.Printf("⚠️  Failed to close Redis: %v", err)
			} else {
				log.Println("✅ Redis connections closed")
			}
		}
	}

	log.Println("✅ Container cleanup completed")
}
