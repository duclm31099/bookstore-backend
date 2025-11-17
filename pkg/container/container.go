package container

import (
	"bookstore-backend/internal/config"
	infraCache "bookstore-backend/internal/infrastructure/cache"
	"bookstore-backend/internal/infrastructure/database"
	"bookstore-backend/internal/infrastructure/storage"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/jwt"
	"context"
	"fmt"
	"log"
	"time"

	// User domain imports

	"bookstore-backend/internal/domains/category"
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

	// INVENTORY
	inventoryHandler "bookstore-backend/internal/domains/inventory/handler"
	inventoryRepo "bookstore-backend/internal/domains/inventory/repository"
	inventoryService "bookstore-backend/internal/domains/inventory/service"

	// CART
	cartHandler "bookstore-backend/internal/domains/cart/handler"
	cartRepo "bookstore-backend/internal/domains/cart/repository"
	cartService "bookstore-backend/internal/domains/cart/service"

	// PROMOTION
	promotionHandler "bookstore-backend/internal/domains/promotion/handler"
	promotionRepo "bookstore-backend/internal/domains/promotion/repository"
	promotionService "bookstore-backend/internal/domains/promotion/service"

	// ORDER
	orderHandler "bookstore-backend/internal/domains/order/handler"
	orderRepo "bookstore-backend/internal/domains/order/repository"
	orderService "bookstore-backend/internal/domains/order/service"

	// PAYMENT
	"bookstore-backend/internal/domains/payment/gateway"
	paymentHandler "bookstore-backend/internal/domains/payment/handler"
	paymentRepo "bookstore-backend/internal/domains/payment/repository"
	paymentService "bookstore-backend/internal/domains/payment/service"

	// REVIEW
	reviewHandler "bookstore-backend/internal/domains/review/handler"
	reviewRepo "bookstore-backend/internal/domains/review/repository"
	reviewService "bookstore-backend/internal/domains/review/service"

	warehouseHandler "bookstore-backend/internal/domains/warehouse/handler"
	warehouseRepo "bookstore-backend/internal/domains/warehouse/repository"
	warehouseService "bookstore-backend/internal/domains/warehouse/service"

	"github.com/hibiken/asynq"
)

type Container struct {
	Config         *config.Config
	DB             *database.PostgresDB
	Cache          cache.Cache
	JWTManager     *jwt.Manager
	VNPayGateway   gateway.VNPayGateway
	MomoGateway    gateway.MomoGateway
	AsynqClient    *asynq.Client
	MinIOStorage   *storage.MinIOStorage
	ImageProcessor *storage.ImageProcessor
	// ========================================
	// REPOSITORY LAYER (DATA ACCESS)
	// ========================================
	UserRepo       user.Repository
	CategoryRepo   category.CategoryRepository
	AuthorRepo     authorRepository.RepositoryInterface
	PublisherRepo  publisherRepo.RepositoryInterface
	AddressRepo    addressRepo.RepositoryInterface
	BookRepo       bookRepo.RepositoryInterface
	InventoryRepo  inventoryRepo.RepositoryInterface
	CartRepo       cartRepo.RepositoryInterface
	PromotionRepo  promotionRepo.PromotionRepository
	OrderRepo      orderRepo.OrderRepository
	PaymentRepo    paymentRepo.PaymentRepoInteface
	RefundRepo     paymentRepo.RefundRepoInterface
	WebHookRepo    paymentRepo.WebhookRepoInterface
	TxManager      paymentRepo.TransactionManager
	ReviewRepo     reviewRepo.ReviewRepository
	ImageBookRepo  bookRepo.BookImageRepository
	BulkImportRepo bookRepo.BulkImportRepoI
	WarehouseRepo  warehouseRepo.Repository
	// ========================================
	// SERVICE LAYER (BUSINESS LOGIC)
	// ========================================

	UserService       user.Service
	CategoryService   category.CategoryService
	AuthorService     authorService.ServiceInterface
	PublisherService  publisherService.ServiceInterface
	AddressService    addressService.ServiceInterface
	BookService       bookService.ServiceInterface
	InventoryService  inventoryService.ServiceInterface
	CartService       cartService.ServiceInterface
	PromotionService  promotionService.ServiceInterface
	OrderSerivce      orderService.OrderService
	PaymentService    paymentService.PaymentService
	RefundService     paymentService.RefundInterface
	ReviewService     reviewService.ServiceInterface
	ImageBookService  bookService.BookImageService
	BulkImportService bookService.BulkImportServiceInterface
	WarehouseService  warehouseService.Service
	// ========================================
	// HANDLER LAYER (HTTP)
	// ========================================
	UserHandler       *userHandler.UserHandler
	CategoryHandler   *categoryHandler.CategoryHandler
	AuthorHandler     *authorHandler.AuthorHandler
	PublisherHandler  *publisherHandler.PublisherHandler
	AddressHandler    *addressHandler.AddressHandler
	BookHandler       *bookHandler.Handler
	InventoryHandler  *inventoryHandler.Handler
	CartHandler       *cartHandler.Handler
	PublicProHandler  *promotionHandler.PublicHandler
	AdminProHandler   *promotionHandler.AdminHandler
	OrderHandler      *orderHandler.OrderHandler
	PaymentHandler    *paymentHandler.PaymentHandler
	ReviewHandler     *reviewHandler.ReviewHandler
	BulkImportHandler *bookHandler.BulkImportHandler
	WarehouseHandler  *warehouseHandler.Handler
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
	redisOpt := asynq.RedisClientOpt{
		Addr:     "localhost:6379", // có thể đọc từ config/env
		Password: "redispassword",  // nếu có password redis
		DB:       0,                // db index
	}
	c := &Container{}
	c.AsynqClient = asynq.NewClient(redisOpt)

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

	// MinIO Configuration
	minioConfig := config.MinIOConfig{
		Endpoint:  utils.GetEnvVariable("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: utils.GetEnvVariable("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: utils.GetEnvVariable("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    utils.GetEnvVariable("MINIO_BUCKET", "bookstore"),
		UseSSL:    utils.GetEnvVariable("MINIO_USE_SSL", "false") == "true",
	}

	minioStorage, err := storage.NewMinIOStorage(minioConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init MinIO storage: %w", err)
	}
	c.MinIOStorage = minioStorage
	log.Println("✅ MinIO storage initialized")

	// Image Processor
	c.ImageProcessor = storage.NewImageProcessor()
	log.Println("✅ Image processor initialized")

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
	c.InventoryRepo = inventoryRepo.NewRepository(pool)
	c.CartRepo = cartRepo.NewPostgresRepository(pool, c.Cache)
	c.PromotionRepo = promotionRepo.NewPostgresRepository(pool)
	c.OrderRepo = orderRepo.NewPostgresOrderRepository(pool)
	c.PaymentRepo = paymentRepo.NewppRepository(pool)
	c.RefundRepo = paymentRepo.NewRefundRepository(pool)
	c.TxManager = paymentRepo.NewPostgresTransactionManager(pool)
	c.ReviewRepo = reviewRepo.NewPostgresReviewRepository(pool)
	c.ImageBookRepo = bookRepo.NewBookImageRepository(pool)
	c.BulkImportRepo = bookRepo.NewBulkImportRepository(pool)
	c.WarehouseRepo = warehouseRepo.NewRepository(pool)
	return nil
}

// initServices khởi tạo tất cả services
func (c *Container) initServices() error {
	c.UserService = userService.NewUserService(
		c.UserRepo,   // Inject repository
		c.JWTManager, // Inject JWT secret từ config
		c.AsynqClient,
		c.Cache,
	)
	c.BulkImportService = bookService.NewBulkImportService(
		c.BookRepo, c.ImageBookRepo, c.AuthorRepo,
		c.CategoryRepo, c.PublisherRepo, c.ImageBookRepo, c.DB.Pool, c.MinIOStorage,
		c.ImageProcessor, c.AsynqClient,
	)
	c.WarehouseService = warehouseService.NewService(c.WarehouseRepo)
	c.CategoryService = categoryService.NewCategoryService(c.CategoryRepo)
	c.AuthorService = authorService.NewAuthorService(c.AuthorRepo)
	c.PublisherService = publisherService.NewPublisherService(c.PublisherRepo)
	c.AddressService = addressService.NewAddressService(c.AddressRepo)
	c.BookService = bookService.NewService(
		c.BookRepo, c.Cache, c.ImageProcessor, c.MinIOStorage, c.ImageBookRepo, c.AsynqClient,
	)
	c.InventoryService = inventoryService.NewService(c.InventoryRepo, c.AsynqClient)
	c.CartService = cartService.NewCartService(
		c.CartRepo, c.InventoryService,
		c.AddressService, c.InventoryRepo,
		c.BookService, c.OrderRepo, c.AsynqClient,
	)
	c.PromotionService = promotionService.NewPromotionService(c.PromotionRepo, c.DB.Pool, c.CartService)
	c.OrderSerivce = orderService.NewOrderService(
		c.OrderRepo, c.WarehouseService,
		c.InventoryRepo,
		c.AddressRepo,
		c.CartRepo,
		c.PromotionRepo,
		c.InventoryService,
		c.AsynqClient,
	)
	c.PaymentService = paymentService.NewPaymentService(
		c.PaymentRepo, c.WebHookRepo, c.RefundRepo, c.TxManager,
		c.VNPayGateway, c.MomoGateway, c.OrderSerivce,
	)
	c.RefundService = paymentService.NewRefundService(
		c.PaymentRepo, c.RefundRepo, c.OrderRepo,
		c.VNPayGateway, c.MomoGateway, c.OrderSerivce,
	)
	c.ReviewService = reviewService.NewReviewService(c.ReviewRepo)
	c.ImageBookService = bookService.NewBookImageService(c.ImageBookRepo, c.MinIOStorage, c.ImageProcessor)
	return nil
}

// initHandlers khởi tạo tất cả HTTP handlers
func (c *Container) initHandlers() error {
	c.UserHandler = userHandler.NewUserHandler(c.UserService, c.CartService, c.JWTManager)
	c.CategoryHandler = categoryHandler.NewCategoryHandler(c.CategoryService)
	c.AuthorHandler = authorHandler.NewAuthorHandler(c.AuthorService)
	c.PublisherHandler = publisherHandler.NewPublisherHandler(c.PublisherService)
	c.AddressHandler = addressHandler.NewAddressHandler(c.AddressService)
	c.BookHandler = bookHandler.NewHandler(c.BookService, c.Cache, c.ImageProcessor)
	c.InventoryHandler = inventoryHandler.NewHandler(c.InventoryService)
	c.ReviewHandler = reviewHandler.NewReviewHandler(c.ReviewService)
	c.CartHandler = cartHandler.NewHandler(c.CartService)
	c.WarehouseHandler = warehouseHandler.NewHandler(c.WarehouseService)
	c.BulkImportHandler = bookHandler.NewBulkImportHandler(c.BulkImportService)
	c.AdminProHandler = promotionHandler.NewAdminHandler(c.PromotionService)
	c.PublicProHandler = promotionHandler.NewPublicHandler(c.PromotionService, c.CartService)
	c.PaymentHandler = paymentHandler.NewPaymentHandler(c.PaymentService, c.RefundService)
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
	if c.AsynqClient != nil {
		if err := c.AsynqClient.Close(); err != nil {
			log.Printf("⚠️  AsynqClient close failed: %v", err)
		} else {
			log.Println("✅ Asynq client closed")
		}
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
