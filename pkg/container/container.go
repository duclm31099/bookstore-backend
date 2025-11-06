package container

import (
	"context"
	"fmt"
	"log"
	"time"

	"bookstore-backend/internal/config"
	infraCache "bookstore-backend/internal/infrastructure/cache"
	"bookstore-backend/internal/infrastructure/database"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/jwt"

	// User domain imports
	"bookstore-backend/internal/domains/address"
	"bookstore-backend/internal/domains/author"
	"bookstore-backend/internal/domains/book"
	"bookstore-backend/internal/domains/category"
	"bookstore-backend/internal/domains/publisher"
	"bookstore-backend/internal/domains/user"

	userHandler "bookstore-backend/internal/domains/user/handler"
	userRepo "bookstore-backend/internal/domains/user/repository"
	userService "bookstore-backend/internal/domains/user/service"

	// AUTHOR
	authorHandler "bookstore-backend/internal/domains/author/handler"
	authorRepository "bookstore-backend/internal/domains/author/repository"
	authorService "bookstore-backend/internal/domains/author/service"

	// CATEGORY

	categoryHandler "bookstore-backend/internal/domains/category/handler"
	categoryRepo "bookstore-backend/internal/domains/category/repository"
	categoryService "bookstore-backend/internal/domains/category/service"

	// PUBLISHER
	publisherHandler "bookstore-backend/internal/domains/publisher/handler"
	publisherRepo "bookstore-backend/internal/domains/publisher/repository"
	publisherService "bookstore-backend/internal/domains/publisher/service"

	// ADDRESS
	addressHandler "bookstore-backend/internal/domains/address/handler"
	addressRepo "bookstore-backend/internal/domains/address/repository"
	addressService "bookstore-backend/internal/domains/address/service"

	// BOOK
	bookHandler "bookstore-backend/internal/domains/book/handler"
	bookRepo "bookstore-backend/internal/domains/book/repository"
	bookService "bookstore-backend/internal/domains/book/service"
)

type Container struct {
	Config     *config.Config
	DB         *database.PostgresDB
	Cache      cache.Cache
	JWTManager *jwt.Manager

	// ========================================
	// REPOSITORY LAYER (DATA ACCESS)
	// ========================================
	UserRepo      user.Repository
	CategoryRepo  category.CategoryRepository
	AuthorRepo    author.Repository
	PublisherRepo publisher.Repository
	AddressRepo   address.Repository
	BookRepo      book.RepositoryInterface

	// ========================================
	// SERVICE LAYER (BUSINESS LOGIC)
	// ========================================

	UserService      user.Service
	CategoryService  category.CategoryService
	AuthorService    author.Service
	PublisherService publisher.Service
	AddressService   address.Service
	BookService      book.ServiceInterface
	// ========================================
	// HANDLER LAYER (HTTP)
	// ========================================
	UserHandler      *userHandler.UserHandler
	CategoryHandler  *categoryHandler.CategoryHandler
	AuthorHandler    *authorHandler.AuthorHandler
	PublisherHandler *publisherHandler.PublisherHandler
	AddressHandler   *addressHandler.AddressHandler
	BookHandler      *bookHandler.Handler
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

	c := &Container{}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	c.Config = cfg

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

	redisCache := infraCache.NewRedisCache(
		cfg.Redis.Host,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)

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

	if err := c.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to init repositories: %w", err)
	}

	if err := c.initServices(); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := c.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to init handlers: %w", err)
	}

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
	c.AuthorRepo = authorRepository.NewPostgresRepository(pool, c.Cache)
	c.PublisherRepo = publisherRepo.NewPostgresRepository(pool, c.Cache)
	c.AddressRepo = addressRepo.NewPostgresRepository(pool)
	c.BookRepo = bookRepo.NewPostgresRepository(pool, c.Cache)
	return nil
}

// initServices khởi tạo tất cả services
func (c *Container) initServices() error {
	c.UserService = userService.NewUserService(
		c.UserRepo,   // Inject repository
		c.JWTManager, // Inject JWT secret từ config
	)

	c.CategoryService = categoryService.NewCategoryService(c.CategoryRepo)
	c.AuthorService = authorService.NewAuthorService(c.AuthorRepo)
	c.PublisherService = publisherService.NewPublisherService(c.PublisherRepo)
	c.AddressService = addressService.NewAddressService(c.AddressRepo)
	c.BookService = bookService.NewService(c.BookRepo, c.Cache)
	return nil
}

// initHandlers khởi tạo tất cả HTTP handlers
func (c *Container) initHandlers() error {
	c.UserHandler = userHandler.NewUserHandler(c.UserService)
	c.CategoryHandler = categoryHandler.NewCategoryHandler(c.CategoryService)
	c.AuthorHandler = authorHandler.NewAuthorHandler(c.AuthorService)
	c.PublisherHandler = publisherHandler.NewPublisherHandler(c.PublisherService)
	c.AddressHandler = addressHandler.NewAddressHandler(c.AddressService)
	c.BookHandler = bookHandler.NewHandler(c.BookService, c.Cache)
	return nil
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
