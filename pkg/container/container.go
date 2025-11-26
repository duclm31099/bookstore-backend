package container

import (
	"bookstore-backend/internal/config"
	infraCache "bookstore-backend/internal/infrastructure/cache"
	"bookstore-backend/internal/infrastructure/database"
	"bookstore-backend/internal/infrastructure/storage"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/jwt"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"
	"log"
	"time"

	// Domain imports
	"bookstore-backend/internal/domains/category"
	"bookstore-backend/internal/domains/user"

	userHandler "bookstore-backend/internal/domains/user/handler"
	userRepo "bookstore-backend/internal/domains/user/repository"
	userService "bookstore-backend/internal/domains/user/service"

	authorHandler "bookstore-backend/internal/domains/author/handler"
	authorRepository "bookstore-backend/internal/domains/author/repository"
	authorService "bookstore-backend/internal/domains/author/service"

	categoryHandler "bookstore-backend/internal/domains/category/handler"
	categoryRepo "bookstore-backend/internal/domains/category/repository"
	categoryService "bookstore-backend/internal/domains/category/service"

	publisherHandler "bookstore-backend/internal/domains/publisher/handler"
	publisherRepo "bookstore-backend/internal/domains/publisher/repository"
	publisherService "bookstore-backend/internal/domains/publisher/service"

	addressHandler "bookstore-backend/internal/domains/address/handler"
	addressRepo "bookstore-backend/internal/domains/address/repository"
	addressService "bookstore-backend/internal/domains/address/service"

	bookHandler "bookstore-backend/internal/domains/book/handler"
	bookRepo "bookstore-backend/internal/domains/book/repository"
	bookService "bookstore-backend/internal/domains/book/service"

	inventoryHandler "bookstore-backend/internal/domains/inventory/handler"
	inventoryRepo "bookstore-backend/internal/domains/inventory/repository"
	inventoryService "bookstore-backend/internal/domains/inventory/service"

	cartHandler "bookstore-backend/internal/domains/cart/handler"
	cartRepo "bookstore-backend/internal/domains/cart/repository"
	cartService "bookstore-backend/internal/domains/cart/service"

	promotionHandler "bookstore-backend/internal/domains/promotion/handler"
	promotionRepo "bookstore-backend/internal/domains/promotion/repository"
	promotionService "bookstore-backend/internal/domains/promotion/service"

	orderHandler "bookstore-backend/internal/domains/order/handler"
	orderRepo "bookstore-backend/internal/domains/order/repository"
	orderService "bookstore-backend/internal/domains/order/service"

	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/gateway/vnpay"
	paymentHandler "bookstore-backend/internal/domains/payment/handler"
	paymentRepo "bookstore-backend/internal/domains/payment/repository"
	paymentService "bookstore-backend/internal/domains/payment/service"

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

	// Repositories
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

	// Services
	UserService       user.Service
	CategoryService   category.CategoryService
	AuthorService     authorService.ServiceInterface
	PublisherService  publisherService.ServiceInterface
	AddressService    addressService.ServiceInterface
	BookService       bookService.ServiceInterface
	InventoryService  inventoryService.ServiceInterface
	CartService       cartService.ServiceInterface
	PromotionService  promotionService.ServiceInterface
	OrderService      orderService.OrderService
	PaymentService    paymentService.PaymentService
	RefundService     paymentService.RefundInterface
	ReviewService     reviewService.ServiceInterface
	ImageBookService  bookService.BookImageService
	BulkImportService bookService.BulkImportServiceInterface
	WarehouseService  warehouseService.Service

	// Handlers
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
// CONSTRUCTOR
// ========================================
func NewContainer() (*Container, error) {
	c := &Container{}

	// Step 1: Infrastructure
	if err := c.initInfrastructure(); err != nil {
		return nil, fmt.Errorf("failed to init infrastructure: %w", err)
	}

	// Step 2: Gateways
	if err := c.initGateways(); err != nil {
		return nil, fmt.Errorf("failed to init gateways: %w", err)
	}

	// Step 3: Repositories
	if err := c.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to init repositories: %w", err)
	}

	// Step 4: Services (3 phases)
	if err := c.initServices(); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// Step 5: Handlers
	if err := c.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to init handlers: %w", err)
	}

	log.Println("✅ Container initialized successfully")
	return c, nil
}

// ========================================
// STEP 1: INFRASTRUCTURE
// ========================================
func (c *Container) initInfrastructure() error {
	// Config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	c.Config = cfg
	log.Println("✅ Config loaded")

	// Database
	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		return fmt.Errorf("failed to load database config: %w", err)
	}

	db := database.NewPostgresDB(dbConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.HealthCheck(context.Background()); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	c.DB = db
	log.Println("✅ Database connected")

	// Redis Cache
	redisCache := infraCache.NewRedisCache(
		cfg.Redis.Host,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)

	if rc, ok := redisCache.(*infraCache.RedisCache); ok {
		if err := rc.Connect(context.Background()); err != nil {
			log.Printf("⚠️  Redis connection failed (non-critical): %v", err)
		} else {
			log.Println("✅ Redis connected")
		}
	}
	c.Cache = redisCache

	// JWT Manager
	c.JWTManager = jwt.NewManager(cfg.JWT.Secret)
	log.Println("✅ JWT Manager initialized")

	// Asynq Client
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Host,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	c.AsynqClient = asynq.NewClient(redisOpt)
	log.Println("✅ Asynq Client initialized")

	// MinIO Storage
	minioConfig := config.MinIOConfig{
		Endpoint:  utils.GetEnvVariable("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: utils.GetEnvVariable("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: utils.GetEnvVariable("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    utils.GetEnvVariable("MINIO_BUCKET", "bookstore"),
		UseSSL:    utils.GetEnvVariable("MINIO_USE_SSL", "false") == "true",
	}

	minioStorage, err := storage.NewMinIOStorage(minioConfig)
	if err != nil {
		return fmt.Errorf("failed to init MinIO storage: %w", err)
	}
	c.MinIOStorage = minioStorage
	log.Println("✅ MinIO storage initialized")

	// Image Processor
	c.ImageProcessor = storage.NewImageProcessor()
	log.Println("✅ Image processor initialized")

	return nil
}

// ========================================
// STEP 2: GATEWAYS
// ========================================
func (c *Container) initGateways() error {
	// VNPay Gateway
	vnpCfg := vnpay.NewConfig(
		c.Config.VNPay.TmnCode,
		c.Config.VNPay.HashSecret,
		c.Config.VNPay.APIURL,
		c.Config.VNPay.ReturnURL,
		c.Config.VNPay.IPNURL,
	)

	vnpClient, err := vnpay.NewClient(vnpCfg)
	if err != nil {
		return fmt.Errorf("failed to init VNPay client: %w", err)
	}
	c.VNPayGateway = vnpClient
	log.Println("✅ VNPay Gateway initialized")
	logger.Info("Init gateway:", map[string]interface{}{
		"vnpCfg": vnpCfg,
	})
	// TODO: Momo Gateway (phase 2)
	// c.MomoGateway = momo.NewClient(momoConfig)

	return nil
}

// ========================================
// STEP 3: REPOSITORIES
// ========================================
func (c *Container) initRepositories() error {
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

	log.Println("✅ All repositories initialized")
	return nil
}

// ========================================
// STEP 4: SERVICES (3 PHASES)
// ========================================
func (c *Container) initServices() error {
	// PHASE 1: Independent Services (no cross-dependencies)
	if err := c.initIndependentServices(); err != nil {
		return fmt.Errorf("phase 1 failed: %w", err)
	}

	// PHASE 2: Dependent Services (depend on Phase 1)
	if err := c.initDependentServices(); err != nil {
		return fmt.Errorf("phase 2 failed: %w", err)
	}

	// PHASE 3: Cross-dependent Services (circular dependencies)
	if err := c.initCrossDependentServices(); err != nil {
		return fmt.Errorf("phase 3 failed: %w", err)
	}

	// PHASE 4: Validate all services
	if err := c.validateServices(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	log.Println("✅ All services initialized and validated")
	return nil
}

// ========================================
// PHASE 1: INDEPENDENT SERVICES
// ========================================
func (c *Container) initIndependentServices() error {
	// Services with no service dependencies (only repos, cache, etc.)

	c.UserService = userService.NewUserService(
		c.UserRepo,
		c.JWTManager,
		c.AsynqClient,
		c.Cache,
	)
	log.Println("  ✓ UserService")

	c.CategoryService = categoryService.NewCategoryService(c.CategoryRepo)
	log.Println("  ✓ CategoryService")

	c.AuthorService = authorService.NewAuthorService(c.AuthorRepo)
	log.Println("  ✓ AuthorService")

	c.PublisherService = publisherService.NewPublisherService(c.PublisherRepo)
	log.Println("  ✓ PublisherService")

	c.AddressService = addressService.NewAddressService(c.AddressRepo)
	log.Println("  ✓ AddressService")

	c.WarehouseService = warehouseService.NewService(c.WarehouseRepo)
	log.Println("  ✓ WarehouseService")

	c.ReviewService = reviewService.NewReviewService(c.ReviewRepo)
	log.Println("  ✓ ReviewService")

	c.ImageBookService = bookService.NewBookImageService(
		c.ImageBookRepo,
		c.MinIOStorage,
		c.ImageProcessor,
	)
	log.Println("  ✓ ImageBookService")

	c.InventoryService = inventoryService.NewService(
		c.InventoryRepo,
		c.AsynqClient,
	)
	log.Println("  ✓ InventoryService")

	return nil
}

// ========================================
// PHASE 2: DEPENDENT SERVICES
// ========================================
func (c *Container) initDependentServices() error {
	// Services that depend on Phase 1 services

	c.BookService = bookService.NewService(
		c.BookRepo,
		c.Cache,
		c.ImageProcessor,
		c.MinIOStorage,
		c.ImageBookRepo,
		c.AsynqClient,
	)
	log.Println("  ✓ BookService")

	c.BulkImportService = bookService.NewBulkImportService(
		c.BookRepo,
		c.ImageBookRepo,
		c.AuthorRepo,
		c.CategoryRepo,
		c.PublisherRepo,
		c.ImageBookRepo,
		c.DB.Pool,
		c.MinIOStorage,
		c.ImageProcessor,
		c.AsynqClient,
	)
	log.Println("  ✓ BulkImportService")

	// OrderService - Initialize WITHOUT CartService (will be wired later)
	c.OrderService = orderService.NewOrderService(
		c.OrderRepo,
		c.WarehouseService,
		c.InventoryRepo,
		c.AddressRepo,
		c.CartRepo,
		c.PromotionRepo,
		c.InventoryService,
		c.AsynqClient,
	)
	log.Println("  ✓ OrderService (without CartService)")

	return nil
}

// ========================================
// PHASE 3: CROSS-DEPENDENT SERVICES
// ========================================
func (c *Container) initCrossDependentServices() error {
	// Services with circular dependencies

	// CartService needs OrderService (which is already created)
	c.CartService = cartService.NewCartService(
		c.CartRepo,
		c.InventoryService,
		c.AddressService,
		c.InventoryRepo,
		c.BookService,
		c.OrderService, // ✅ OrderService already exists
		c.AsynqClient,
	)
	log.Println("  ✓ CartService")

	// PromotionService needs CartService
	c.PromotionService = promotionService.NewPromotionService(
		c.PromotionRepo,
		c.DB.Pool,
		c.CartService, // ✅ CartService now exists
	)
	log.Println("  ✓ PromotionService")

	// PaymentService needs OrderService
	c.PaymentService = paymentService.NewPaymentService(
		c.PaymentRepo,
		c.WebHookRepo,
		c.RefundRepo,
		c.TxManager,
		c.VNPayGateway,
		c.MomoGateway,
		c.OrderService, // ✅ OrderService exists
	)
	log.Println("  ✓ PaymentService")

	// RefundService needs OrderService
	c.RefundService = paymentService.NewRefundService(
		c.PaymentRepo,
		c.RefundRepo,
		c.OrderRepo,
		c.VNPayGateway,
		c.MomoGateway,
		c.OrderService, // ✅ OrderService exists
	)
	log.Println("  ✓ RefundService")

	return nil
}

// ========================================
// PHASE 4: VALIDATION
// ========================================
func (c *Container) validateServices() error {
	services := map[string]interface{}{
		"UserService":       c.UserService,
		"CategoryService":   c.CategoryService,
		"AuthorService":     c.AuthorService,
		"PublisherService":  c.PublisherService,
		"AddressService":    c.AddressService,
		"BookService":       c.BookService,
		"InventoryService":  c.InventoryService,
		"CartService":       c.CartService,
		"PromotionService":  c.PromotionService,
		"OrderService":      c.OrderService,
		"PaymentService":    c.PaymentService,
		"RefundService":     c.RefundService,
		"ReviewService":     c.ReviewService,
		"ImageBookService":  c.ImageBookService,
		"BulkImportService": c.BulkImportService,
		"WarehouseService":  c.WarehouseService,
	}

	var nilServices []string
	for name, svc := range services {
		if svc == nil {
			nilServices = append(nilServices, name)
		}
	}

	if len(nilServices) > 0 {
		return fmt.Errorf("the following services are nil: %v", nilServices)
	}

	log.Println("  ✓ All services validated (none are nil)")
	return nil
}

// ========================================
// STEP 5: HANDLERS
// ========================================
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
	c.OrderHandler = orderHandler.NewOrderHandler(c.OrderService)
	c.PaymentHandler = paymentHandler.NewPaymentHandler(c.PaymentService, c.RefundService)

	log.Println("✅ All handlers initialized")
	return nil
}

// ========================================
// CLEANUP
// ========================================
func (c *Container) Cleanup() {
	log.Println("🧹 Cleaning up container resources...")

	if c.DB != nil && c.DB.Pool != nil {
		c.DB.Pool.Close()
		log.Println("  ✓ Database connections closed")
	}

	if c.AsynqClient != nil {
		if err := c.AsynqClient.Close(); err != nil {
			log.Printf("  ⚠️  AsynqClient close failed: %v", err)
		} else {
			log.Println("  ✓ Asynq client closed")
		}
	}

	if c.Cache != nil {
		if rc, ok := c.Cache.(*infraCache.RedisCache); ok {
			if err := rc.Close(); err != nil {
				log.Printf("  ⚠️  Failed to close Redis: %v", err)
			} else {
				log.Println("  ✓ Redis connections closed")
			}
		}
	}

	log.Println("✅ Container cleanup completed")
}
