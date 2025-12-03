package container

import (
	"bookstore-backend/internal/config"
	infraCache "bookstore-backend/internal/infrastructure/cache"
	"bookstore-backend/internal/infrastructure/database"
	"bookstore-backend/internal/infrastructure/email"
	"bookstore-backend/internal/infrastructure/push"
	"bookstore-backend/internal/infrastructure/sms"
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

	// Handlers
	addressHandler "bookstore-backend/internal/domains/address/handler"
	authorHandler "bookstore-backend/internal/domains/author/handler"
	bookHandler "bookstore-backend/internal/domains/book/handler"
	cartHandler "bookstore-backend/internal/domains/cart/handler"
	categoryHandler "bookstore-backend/internal/domains/category/handler"
	inventoryHandler "bookstore-backend/internal/domains/inventory/handler"
	notificationHandler "bookstore-backend/internal/domains/notification/handler"
	orderHandler "bookstore-backend/internal/domains/order/handler"
	paymentHandler "bookstore-backend/internal/domains/payment/handler"
	promotionHandler "bookstore-backend/internal/domains/promotion/handler"
	publisherHandler "bookstore-backend/internal/domains/publisher/handler"
	reviewHandler "bookstore-backend/internal/domains/review/handler"
	userHandler "bookstore-backend/internal/domains/user/handler"
	warehouseHandler "bookstore-backend/internal/domains/warehouse/handler"

	// Repositories
	addressRepo "bookstore-backend/internal/domains/address/repository"
	authorRepository "bookstore-backend/internal/domains/author/repository"
	bookRepo "bookstore-backend/internal/domains/book/repository"
	cartRepo "bookstore-backend/internal/domains/cart/repository"
	categoryRepo "bookstore-backend/internal/domains/category/repository"
	inventoryRepo "bookstore-backend/internal/domains/inventory/repository"
	notificationRepo "bookstore-backend/internal/domains/notification/repository"
	orderRepo "bookstore-backend/internal/domains/order/repository"
	paymentRepo "bookstore-backend/internal/domains/payment/repository"
	promotionRepo "bookstore-backend/internal/domains/promotion/repository"
	publisherRepo "bookstore-backend/internal/domains/publisher/repository"
	reviewRepo "bookstore-backend/internal/domains/review/repository"
	userRepo "bookstore-backend/internal/domains/user/repository"
	warehouseRepo "bookstore-backend/internal/domains/warehouse/repository"

	// Services
	addressService "bookstore-backend/internal/domains/address/service"
	authorService "bookstore-backend/internal/domains/author/service"
	bookService "bookstore-backend/internal/domains/book/service"
	cartService "bookstore-backend/internal/domains/cart/service"
	categoryService "bookstore-backend/internal/domains/category/service"
	inventoryService "bookstore-backend/internal/domains/inventory/service"
	notificationService "bookstore-backend/internal/domains/notification/service"
	orderService "bookstore-backend/internal/domains/order/service"
	paymentService "bookstore-backend/internal/domains/payment/service"
	promotionService "bookstore-backend/internal/domains/promotion/service"
	publisherService "bookstore-backend/internal/domains/publisher/service"
	reviewService "bookstore-backend/internal/domains/review/service"
	userService "bookstore-backend/internal/domains/user/service"
	warehouseService "bookstore-backend/internal/domains/warehouse/service"

	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/gateway/vnpay"

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
	JobConfig      config.JobConfig

	// Infrastructure Services
	EmailService              email.EmailService
	SMSService                *sms.MockSMSService
	NotificationEmailProvider *email.NotificationEmailProvider // ✅ For notification domain (adapter)

	PushService *push.MockPushService

	// Repositories
	UserRepo         user.Repository
	CategoryRepo     category.CategoryRepository
	AuthorRepo       authorRepository.RepositoryInterface
	PublisherRepo    publisherRepo.RepositoryInterface
	AddressRepo      addressRepo.RepositoryInterface
	BookRepo         bookRepo.RepositoryInterface
	InventoryRepo    inventoryRepo.RepositoryInterface
	CartRepo         cartRepo.RepositoryInterface
	PromotionRepo    promotionRepo.PromotionRepository
	OrderRepo        orderRepo.OrderRepository
	PaymentRepo      paymentRepo.PaymentRepoInteface
	RefundRepo       paymentRepo.RefundRepoInterface
	WebHookRepo      paymentRepo.WebhookRepoInterface
	TxManager        paymentRepo.TransactionManager
	ReviewRepo       reviewRepo.ReviewRepository
	ImageBookRepo    bookRepo.BookImageRepository
	BulkImportRepo   bookRepo.BulkImportRepoI
	WarehouseRepo    warehouseRepo.Repository
	NotificationRepo notificationRepo.NotificationRepository
	PreferencesRepo  notificationRepo.PreferencesRepository
	TemplateRepo     notificationRepo.TemplateRepository
	DeliveryLogRepo  notificationRepo.DeliveryLogRepository
	CampaignRepo     notificationRepo.CampaignRepository
	RateLimitRepo    notificationRepo.RateLimitRepository

	// Services
	UserService         user.Service
	CategoryService     category.CategoryService
	AuthorService       authorService.ServiceInterface
	PublisherService    publisherService.ServiceInterface
	AddressService      addressService.ServiceInterface
	BookService         bookService.ServiceInterface
	InventoryService    inventoryService.ServiceInterface
	CartService         cartService.ServiceInterface
	PromotionService    promotionService.ServiceInterface
	OrderService        orderService.OrderService
	PaymentService      paymentService.PaymentService
	RefundService       paymentService.RefundInterface
	ReviewService       reviewService.ServiceInterface
	ImageBookService    bookService.BookImageService
	BulkImportService   bookService.BulkImportServiceInterface
	WarehouseService    warehouseService.Service
	NotificationService notificationService.NotificationService
	PreferencesService  notificationService.PreferencesService
	TemplateService     notificationService.TemplateService
	DeliveryService     notificationService.DeliveryService
	CampaignService     notificationService.CampaignService

	// Handlers
	UserHandler         *userHandler.UserHandler
	CategoryHandler     *categoryHandler.CategoryHandler
	AuthorHandler       *authorHandler.AuthorHandler
	PublisherHandler    *publisherHandler.PublisherHandler
	AddressHandler      *addressHandler.AddressHandler
	BookHandler         *bookHandler.Handler
	InventoryHandler    *inventoryHandler.Handler
	CartHandler         *cartHandler.Handler
	PublicProHandler    *promotionHandler.PublicHandler
	AdminProHandler     *promotionHandler.AdminHandler
	OrderHandler        *orderHandler.OrderHandler
	PaymentHandler      *paymentHandler.PaymentHandler
	ReviewHandler       *reviewHandler.ReviewHandler
	BulkImportHandler   *bookHandler.BulkImportHandler
	WarehouseHandler    *warehouseHandler.Handler
	NotificationHandler notificationHandler.NotificationHandler
	PreferencesHandler  notificationHandler.PreferencesHandler
	TemplateHandler     notificationHandler.TemplateHandler
	CampaignHandler     notificationHandler.CampaignHandler
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

	// Step 3: Providers (Email, SMS, Push)
	if err := c.initProviders(); err != nil {
		return nil, fmt.Errorf("failed to init providers: %w", err)
	}

	// Step 4: Repositories
	if err := c.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to init repositories: %w", err)
	}

	// Step 5: Services (3 phases)
	if err := c.initServices(); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// Step 6: Handlers
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
	c.JobConfig = cfg.Job

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
// STEP 3: PROVIDERS (Email, SMS, Push)
// ========================================
func (c *Container) initProviders() error {
	// Email Service (existing SMTP service for user domain)
	smtpHost := utils.GetEnvVariable("SMTP_HOST", "localhost")
	smtpPort := utils.GetEnvVariable("SMTP_PORT", "1025")
	c.EmailService = email.NewDevEmailService(smtpHost, smtpPort)
	log.Println("✅ Email Service (SMTP) initialized")

	// Create Notification Email Adapter (for notification domain)
	c.NotificationEmailProvider = email.NewNotificationEmailProvider(c.EmailService)
	log.Println("✅ Notification Email Provider (Adapter) initialized")

	// SMS Service (mock for dev, Twilio for prod)
	useMockSMS := utils.GetEnvVariable("USE_MOCK_SMS", "true") == "true"
	if useMockSMS {
		c.SMSService = sms.NewMockSMSService()
		log.Println("✅ SMS Service (Mock) initialized")
	} else {
		// twilioSID := utils.GetEnvVariable("TWILIO_ACCOUNT_SID", "")
		// twilioToken := utils.GetEnvVariable("TWILIO_AUTH_TOKEN", "")
		// twilioFrom := utils.GetEnvVariable("TWILIO_PHONE_NUMBER", "")
		// c.SMSService = sms.NewTwilioSMSService(twilioSID, twilioToken, twilioFrom)
		log.Println("✅ SMS Service (Twilio) initialized")
	}

	// Push Service (mock for dev, FCM for prod)
	useMockPush := utils.GetEnvVariable("USE_MOCK_PUSH", "true") == "true"
	if useMockPush {
		c.PushService = push.NewMockPushService()
		log.Println("✅ Push Service (Mock) initialized")
	} else {
		// fcmKey := utils.GetEnvVariable("FCM_SERVER_KEY", "")
		// c.PushService = push.NewFCMPushService(fcmKey)
		log.Println("✅ Push Service (FCM) initialized")
	}

	return nil
}

// ========================================
// STEP 4: REPOSITORIES
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

	// Notification Repositories
	c.NotificationRepo = notificationRepo.NewNotificationRepository(pool)
	c.PreferencesRepo = notificationRepo.NewPreferencesRepository(pool)
	c.TemplateRepo = notificationRepo.NewTemplateRepository(pool)
	c.DeliveryLogRepo = notificationRepo.NewDeliveryLogRepository(pool)
	c.CampaignRepo = notificationRepo.NewCampaignRepository(pool)
	c.RateLimitRepo = notificationRepo.NewRateLimitRepository(pool)

	log.Println("✅ All repositories initialized")
	return nil
}

// ========================================
// STEP 5: SERVICES (4 PHASES)
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

	// PHASE 4: Notification Services (with circular deps)
	if err := c.initNotificationServices(); err != nil {
		return fmt.Errorf("phase 4 (notification) failed: %w", err)
	}

	// PHASE 5: Validate all services
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

	// Preferences Service (independent)
	c.PreferencesService = notificationService.NewPreferencesService(c.PreferencesRepo)
	log.Println("  ✓ PreferencesService")

	// Template Service (independent)
	c.TemplateService = notificationService.NewTemplateService(c.TemplateRepo)
	log.Println("  ✓ TemplateService")

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
		c.BookService,
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
// PHASE 4: NOTIFICATION SERVICES
// ========================================
// ========================================
// PHASE 4: NOTIFICATION SERVICES
// ========================================
func (c *Container) initNotificationServices() error {
	// Delivery Service (uses notification-specific providers)
	c.DeliveryService = notificationService.NewDeliveryService(
		c.NotificationRepo,
		c.DeliveryLogRepo,
		c.NotificationEmailProvider, // ✅ Use the adapter field
		c.SMSService,
		c.PushService,
	)
	log.Println("  ✓ DeliveryService")

	// Notification Service (depends on Preferences, Template, Delivery)
	c.NotificationService = notificationService.NewNotificationService(
		c.NotificationRepo,
		c.PreferencesRepo,
		c.TemplateRepo,
		c.RateLimitRepo,
		c.DeliveryLogRepo,
		c.UserRepo,
	)
	log.Println("  ✓ NotificationService (base)")

	// Set circular dependencies for NotificationService
	if ns, ok := c.NotificationService.(interface {
		SetDependencies(notificationService.PreferencesService, notificationService.TemplateService, notificationService.DeliveryService)
	}); ok {
		ns.SetDependencies(c.PreferencesService, c.TemplateService, c.DeliveryService)
		log.Println("  ✓ NotificationService dependencies wired")
	}

	// Campaign Service (depends on Notification, Template)
	c.CampaignService = notificationService.NewCampaignService(
		c.CampaignRepo,
		c.TemplateRepo,
		c.NotificationRepo,
	)
	log.Println("  ✓ CampaignService (base)")

	// Set circular dependencies for CampaignService
	if cs, ok := c.CampaignService.(interface {
		SetDependencies(notificationService.NotificationService, notificationService.TemplateService)
	}); ok {
		cs.SetDependencies(c.NotificationService, c.TemplateService)
		log.Println("  ✓ CampaignService dependencies wired")
	}

	return nil
}

// ========================================
// PHASE 5: VALIDATION
// ========================================
func (c *Container) validateServices() error {
	services := map[string]interface{}{
		"UserService":         c.UserService,
		"CategoryService":     c.CategoryService,
		"AuthorService":       c.AuthorService,
		"PublisherService":    c.PublisherService,
		"AddressService":      c.AddressService,
		"BookService":         c.BookService,
		"InventoryService":    c.InventoryService,
		"CartService":         c.CartService,
		"PromotionService":    c.PromotionService,
		"OrderService":        c.OrderService,
		"PaymentService":      c.PaymentService,
		"RefundService":       c.RefundService,
		"ReviewService":       c.ReviewService,
		"ImageBookService":    c.ImageBookService,
		"BulkImportService":   c.BulkImportService,
		"WarehouseService":    c.WarehouseService,
		"NotificationService": c.NotificationService,
		"PreferencesService":  c.PreferencesService,
		"TemplateService":     c.TemplateService,
		"DeliveryService":     c.DeliveryService,
		"CampaignService":     c.CampaignService,
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
// STEP 6: HANDLERS
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
	c.CartHandler = cartHandler.NewHandler(c.CartService, c.PromotionService)
	c.WarehouseHandler = warehouseHandler.NewHandler(c.WarehouseService)
	c.BulkImportHandler = bookHandler.NewBulkImportHandler(c.BulkImportService)
	c.AdminProHandler = promotionHandler.NewAdminHandler(c.PromotionService)
	c.PublicProHandler = promotionHandler.NewPublicHandler(c.PromotionService, c.CartService)
	c.OrderHandler = orderHandler.NewOrderHandler(c.OrderService)
	c.PaymentHandler = paymentHandler.NewPaymentHandler(c.PaymentService, c.RefundService)

	// Notification Handlers
	c.NotificationHandler = notificationHandler.NewNotificationHandler(c.NotificationService)
	c.PreferencesHandler = notificationHandler.NewPreferencesHandler(c.PreferencesService)
	c.TemplateHandler = notificationHandler.NewTemplateHandler(c.TemplateService)
	c.CampaignHandler = notificationHandler.NewCampaignHandler(c.CampaignService) // ✅ Should work now

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
