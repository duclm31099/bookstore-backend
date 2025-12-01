package main

import (
	"bookstore-backend/internal/shared/middleware"
	"bookstore-backend/pkg/container"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter(c *container.Container) *gin.Engine {
	router := gin.New()

	// Global middlewares
	router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.CORS(),
		middleware.ClientIPMiddleware(),
	)

	// Cart middleware configuration
	cartMiddlewareConfig := middleware.DefaultCartMiddlewareConfig(c.CartService)
	if os.Getenv("ENV") == "development" {
		cartMiddlewareConfig.CookieSecure = false
	}

	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", healthCheckHandler(c))
		v1.GET("/db-test", databaseTestHandler(c))

		setupAuthRoutes(v1, c)
		setupUserRoutes(v1, c)
		setupAdminRoutes(v1, c)
		setupCategoryRoutes(v1, c)
		setupAuthorRoutes(v1, c)
		setupPublisherRoutes(v1, c)
		setupAddressRoutes(v1, c)
		setupBookRoutes(v1, c)
		setupWarehouseRoutes(v1, c)
		setupInventoryRoutes(v1, c)
		setupCartRoutes(v1, c, &cartMiddlewareConfig)
		setupPromotionRoutes(v1, c)
		setupOrderRoutes(v1, c)
		setupPaymentRoutes(v1, c)
		setupWebhookRoutes(v1, c)
		setupAdminOrderRoutes(v1, c)
		setupAdminPaymentRoutes(v1, c)
		setupReviewRoutes(v1, c)
		setupNotificationRoutes(v1, c)
	}

	return router
}

// ========================================
// AUTH ROUTES
// ========================================
func setupAuthRoutes(v1 *gin.RouterGroup, c *container.Container) {
	auth := v1.Group("/auth")
	{
		auth.POST("/register", c.UserHandler.Register)
		auth.POST("/login", c.UserHandler.Login)
		auth.POST("/logout", middleware.AuthMiddleware(c.Config.JWT.Secret), c.UserHandler.Logout)
		auth.POST("/refresh", c.UserHandler.RefreshToken)
		auth.GET("/verify-email", c.UserHandler.VerifyEmail)
		auth.POST("/resend-verification", c.UserHandler.ResendVerification)
		auth.POST("/forgot-password", c.UserHandler.ForgotPassword)
		auth.POST("/reset-password", c.UserHandler.ResetPassword)
	}
}

// ========================================
// USER ROUTES
// ========================================
func setupUserRoutes(v1 *gin.RouterGroup, c *container.Container) {
	users := v1.Group("/users")
	users.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
	{
		users.GET("/me", c.UserHandler.GetProfile)
		users.PUT("/me", c.UserHandler.UpdateProfile)
		users.PUT("/change-password", c.UserHandler.ChangePassword)
	}
}

// ========================================
// ADMIN ROUTES
// ========================================
func setupAdminRoutes(v1 *gin.RouterGroup, c *container.Container) {
	// TODO: Add Auth + Role middleware
	admin := v1.Group("/admin")
	{
		admin.GET("/users", c.UserHandler.ListUsers)
		admin.PUT("/users/:id/role", c.UserHandler.UpdateUserRole)
		admin.PUT("/users/:id/status", c.UserHandler.UpdateUserStatus)
	}
}

// ========================================
// CATEGORY ROUTES
// ========================================
func setupCategoryRoutes(v1 *gin.RouterGroup, c *container.Container) {
	category := v1.Group("/categories")
	{
		category.POST("", c.CategoryHandler.Create)
		category.GET("", c.CategoryHandler.GetAll)
		category.GET("/tree", c.CategoryHandler.GetTree)
		category.GET("/:id", c.CategoryHandler.GetByID)
		category.GET("/:id/breadcrumb", c.CategoryHandler.GetBreadcrumb)
		category.GET("/by-slug/:slug", c.CategoryHandler.GetBySlug)
		category.PUT("/:id", c.CategoryHandler.Update)
		category.PATCH("/:id/parent", c.CategoryHandler.MoveToParent)
		category.POST("/:id/activate", c.CategoryHandler.Activate)
		category.POST("/:id/deactivate", c.CategoryHandler.Deactivate)
		category.DELETE("/:id", c.CategoryHandler.Delete)
		category.DELETE("/bulk", c.CategoryHandler.BulkDelete)
		category.POST("/bulk/activate", c.CategoryHandler.BulkActivate)
		category.POST("/bulk/deactivate", c.CategoryHandler.BulkDeactivate)
		category.GET("/:id/books", c.CategoryHandler.GetBooksInCategory)
		category.GET("/:id/book-count", c.CategoryHandler.GetCategoryBookCount)
	}
}

// ========================================
// AUTHOR ROUTES
// ========================================
func setupAuthorRoutes(v1 *gin.RouterGroup, c *container.Container) {
	author := v1.Group("/authors")
	{
		author.POST("", c.AuthorHandler.Create)
		author.GET("/:id", c.AuthorHandler.GetByID)
		author.GET("/slug/:slug", c.AuthorHandler.GetBySlug)
		author.GET("", c.AuthorHandler.GetAll)
		author.GET("/search", c.AuthorHandler.Search)
		author.PUT("/:id", c.AuthorHandler.Update)
		author.DELETE("/:id", c.AuthorHandler.Delete)
		author.DELETE("/bulk", c.AuthorHandler.BulkDelete)
		author.GET("/:id/books", c.AuthorHandler.GetWithBookCount)
	}
}

// ========================================
// PUBLISHER ROUTES
// ========================================
func setupPublisherRoutes(v1 *gin.RouterGroup, c *container.Container) {
	publisher := v1.Group("/publishers")
	{
		publisher.POST("", c.PublisherHandler.CreatePublisher)
		publisher.GET("", c.PublisherHandler.ListPublishers)
		publisher.GET("/books", c.PublisherHandler.ListPublishersWithBooks)
		publisher.GET("/slug/:slug", c.PublisherHandler.GetPublisherBySlug)
		publisher.GET("/:id", c.PublisherHandler.GetPublisher)
		publisher.GET("/:id/books", c.PublisherHandler.GetPublisherWithBooks)
		publisher.PUT("/:id", c.PublisherHandler.UpdatePublisher)
		publisher.DELETE("/:id", c.PublisherHandler.DeletePublisher)
	}
}

// ========================================
// ADDRESS ROUTES
// ========================================
func setupAddressRoutes(v1 *gin.RouterGroup, c *container.Container) {
	addresses := v1.Group("/addresses")
	addresses.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
	{
		addresses.POST("", c.AddressHandler.CreateAddress)
		addresses.GET("", c.AddressHandler.ListUserAddresses)
		addresses.GET("/default", c.AddressHandler.GetDefaultAddress)
		addresses.GET("/:id", c.AddressHandler.GetAddressById)
		addresses.PUT("/:id", c.AddressHandler.UpdateAddress)
		addresses.PUT("/:id/set-default", c.AddressHandler.SetDefaultAddress)
		addresses.PUT("/:id/unset-default", c.AddressHandler.UnsetDefaultAddress)
		addresses.DELETE("/:id", c.AddressHandler.DeleteAddress)
	}

	// Admin address routes
	// TODO: Add admin middleware
	adminAddresses := v1.Group("/admin/addresses")
	{
		adminAddresses.GET("", c.AddressHandler.ListAllAddresses)
		adminAddresses.GET("/:id", c.AddressHandler.GetAddressWithUser)
	}
}

// ========================================
// BOOK ROUTES
// ========================================
func setupBookRoutes(v1 *gin.RouterGroup, c *container.Container) {
	books := v1.Group("/books")
	{
		books.GET("", c.BookHandler.ListBooks)
		books.GET("/search", c.BookHandler.SearchBooks)
		books.GET("/:id", c.BookHandler.GetBookDetail)
		books.POST("", c.BookHandler.CreateBook)
		books.PUT("/:id", c.BookHandler.UpdateBook)
		books.DELETE("/:id", c.BookHandler.DeleteBook)
		books.POST("/bulk-import", c.BulkImportHandler.ImportBooks)
		books.GET("/export", c.BookHandler.ExportBooks)
	}
}

// ========================================
// WAREHOUSE ROUTES
// ========================================
func setupWarehouseRoutes(v1 *gin.RouterGroup, c *container.Container) {
	warehouses := v1.Group("/warehouses")
	{
		warehouses.POST("", c.WarehouseHandler.CreateWarehouse)
		warehouses.GET("", c.WarehouseHandler.ListWarehouses)
		warehouses.GET("/active", c.WarehouseHandler.ListActiveWarehouses)
		warehouses.GET("/nearest-with-stock", c.WarehouseHandler.FindNearestWarehouseWithStock)
		warehouses.GET("/validate-stock", c.WarehouseHandler.ValidateWarehouseHasStock)
		warehouses.GET("/code/:code", c.WarehouseHandler.GetWarehouseByCode)
		warehouses.GET("/:id", c.WarehouseHandler.GetWarehouseByID)
		warehouses.GET("/:id/performance", c.InventoryHandler.GetWarehousePerformance)
		warehouses.PUT("/:id", c.WarehouseHandler.UpdateWarehouse)
		warehouses.DELETE("/:id", c.WarehouseHandler.SoftDeleteWarehouse)
		warehouses.DELETE("/deactive", c.InventoryHandler.DeactivateWarehouse)
	}
}

// ========================================
// INVENTORY ROUTES
// ========================================
func setupInventoryRoutes(v1 *gin.RouterGroup, c *container.Container) {
	inventory := v1.Group("/inventories")
	{
		// CRUD
		inventory.POST("", c.InventoryHandler.CreateInventory)
		inventory.GET("", c.InventoryHandler.ListInventories)
		inventory.GET("/:warehouse_id/:book_id", c.InventoryHandler.GetInventoryByWarehouseAndBook)
		inventory.PATCH("/:warehouse_id/:book_id", c.InventoryHandler.UpdateInventory)
		inventory.DELETE("/:warehouse_id/:book_id", c.InventoryHandler.DeleteInventory)

		// Stock operations
		inventory.POST("/reserve", c.InventoryHandler.ReserveStock)
		inventory.POST("/release", c.InventoryHandler.ReleaseStock)
		inventory.POST("/complete-sale", c.InventoryHandler.CompleteSale)
		inventory.POST("/find-warehouse", c.InventoryHandler.FindOptimalWarehouse)
		inventory.POST("/check-availability", c.InventoryHandler.CheckAvailability)
		inventory.GET("/summary/:book_id", c.InventoryHandler.GetStockSummary)

		// Stock adjustment
		inventory.POST("/adjust", c.InventoryHandler.AdjustStock)
		inventory.POST("/restock", c.InventoryHandler.RestockInventory)
		inventory.POST("/bulk-update", c.InventoryHandler.BulkUpdateStock)
		inventory.GET("/bulk-update/:job_id", c.InventoryHandler.GetBulkUpdateStatus)

		// Audit & alerts
		inventory.GET("/audit", c.InventoryHandler.GetAuditTrail)
		inventory.GET("/:warehouse_id/:book_id/history", c.InventoryHandler.GetInventoryHistory)
		inventory.POST("/audit/export", c.InventoryHandler.ExportAuditLog)
		inventory.GET("/alerts/low-stock", c.InventoryHandler.GetLowStockAlerts)
		inventory.GET("/alerts/out-of-stock", c.InventoryHandler.GetOutOfStockItems)
		inventory.PATCH("/alerts/:alert_id/resolve", c.InventoryHandler.MarkAlertResolved)

		// Dashboard
		inventory.GET("/dashboard", c.InventoryHandler.GetDashboardSummary)
		inventory.GET("/analysis/reservations", c.InventoryHandler.GetReservationAnalysis)
	}
}

// ========================================
// CART ROUTES
// ========================================
func setupCartRoutes(v1 *gin.RouterGroup, c *container.Container, config *middleware.CartMiddlewareConfig) {
	cart := v1.Group("/cart")
	cart.Use(
		middleware.AuthMiddleware(c.Config.JWT.Secret),
		middleware.CartMiddleware(*config),
	)
	{
		cart.GET("", c.CartHandler.GetCart)
		cart.POST("/items", c.CartHandler.AddItem)
		cart.GET("/items", c.CartHandler.ListItems)
		cart.PUT("/items/:item_id", c.CartHandler.UpdateItemQuantity)
		cart.DELETE("/items/:item_id", c.CartHandler.RemoveItem)
		cart.DELETE("", c.CartHandler.ClearCart)
		cart.POST("/validate", c.CartHandler.ValidateCart)
		cart.POST("/apply-promotion", c.CartHandler.ApplyPromoCode)
		cart.DELETE("/remove-promotion", c.CartHandler.RemovePromoCode)
		cart.POST("/checkout", c.CartHandler.Checkout)
		cart.GET("/:cart_id/promotions", c.CartHandler.GetAvailablePromotions)
	}
}

// ========================================
// PROMOTION ROUTES
// ========================================
func setupPromotionRoutes(v1 *gin.RouterGroup, c *container.Container) {
	promotion := v1.Group("/promotion")
	{
		// Public routes
		promotion.POST("/validate", c.PublicProHandler.ValidatePromotion)
		promotion.GET("", c.PublicProHandler.ListActivePromotions)

		// Admin routes (TODO: add auth middleware)
		promotion.POST("/create", c.AdminProHandler.CreatePromotion)
		promotion.GET("/list-promotion", c.AdminProHandler.ListPromotions)
		promotion.GET("/:id", c.AdminProHandler.GetPromotionByID)
		promotion.PUT("/:id", c.AdminProHandler.UpdatePromotion)
		promotion.PATCH("/:id/status", c.AdminProHandler.UpdatePromotionStatus)
		promotion.DELETE("/:id", c.AdminProHandler.DeletePromotion)
		promotion.GET("/:id/usage", c.AdminProHandler.GetUsageHistory)
		promotion.POST("/:id/export", c.AdminProHandler.ExportUsageReport)
	}
}

// ========================================
// ORDER ROUTES
// ========================================
func setupOrderRoutes(v1 *gin.RouterGroup, c *container.Container) {
	orders := v1.Group("/orders")
	orders.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
	{
		orders.POST("", c.OrderHandler.CreateOrder)
		orders.GET("", c.OrderHandler.ListOrders)
		orders.GET("/:id", c.OrderHandler.GetOrderDetail)
		orders.POST("/:id/cancel", c.OrderHandler.CancelOrder)
		orders.GET("/track/:order_number", c.OrderHandler.GetOrderByNumber)
	}
}

// ========================================
// PAYMENT ROUTES
// ========================================
func setupPaymentRoutes(v1 *gin.RouterGroup, c *container.Container) {
	payments := v1.Group("/payments")
	payments.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
	{
		payments.POST("/create", c.PaymentHandler.CreatePayment)
		payments.GET("/:payment_id", c.PaymentHandler.GetPaymentStatus)
		payments.GET("", c.PaymentHandler.ListUserPayments)
		payments.POST("/:payment_id/refund-request", c.PaymentHandler.RequestRefund)
		payments.GET("/:payment_id/refund-request", c.PaymentHandler.GetRefundStatus)
	}
}

// ========================================
// WEBHOOK ROUTES
// ========================================
func setupWebhookRoutes(v1 *gin.RouterGroup, c *container.Container) {
	webhooks := v1.Group("/webhooks")
	{
		webhooks.GET("/vnpay", c.PaymentHandler.VNPayWebhook)
		webhooks.POST("/vnpay", c.PaymentHandler.VNPayWebhook)
		webhooks.POST("/momo", c.PaymentHandler.MomoWebhook)
	}
}

// ========================================
// ADMIN ORDER ROUTES
// ========================================
func setupAdminOrderRoutes(v1 *gin.RouterGroup, c *container.Container) {
	// TODO: Add admin middleware
	adminOrders := v1.Group("/admin/orders")
	{
		adminOrders.GET("", c.OrderHandler.ListAllOrders)
		adminOrders.PATCH("/:id/status", c.OrderHandler.UpdateOrderStatus)
	}
}

// ========================================
// ADMIN PAYMENT ROUTES
// ========================================
func setupAdminPaymentRoutes(v1 *gin.RouterGroup, c *container.Container) {
	// TODO: Add admin middleware
	adminPayments := v1.Group("/admin/payments")
	{
		adminPayments.GET("", c.PaymentHandler.AdminListPayments)
		adminPayments.GET("/:payment_id", c.PaymentHandler.AdminGetPaymentDetail)
		adminPayments.POST("/:payment_id/reconcile", c.PaymentHandler.AdminReconcilePayment)
		adminPayments.GET("/refunds/pending", c.PaymentHandler.AdminListPendingRefunds)
		adminPayments.GET("/refunds/:refund_id", c.PaymentHandler.AdminGetRefundDetail)
		adminPayments.POST("/refunds/:refund_id/approve", c.PaymentHandler.AdminApproveRefund)
		adminPayments.POST("/refunds/:refund_id/reject", c.PaymentHandler.AdminRejectRefund)
	}
}

// ========================================
// REVIEW ROUTES
// ========================================
func setupReviewRoutes(v1 *gin.RouterGroup, c *container.Container) {
	// Public review routes
	reviews := v1.Group("/reviews")
	{
		reviews.GET("", c.ReviewHandler.ListReviews)
		reviews.GET("/:id", c.ReviewHandler.GetReview)
	}

	// User review routes
	userReviews := v1.Group("/reviews")
	userReviews.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
	{
		userReviews.POST("", c.ReviewHandler.CreateReview)
		userReviews.PUT("/:id", c.ReviewHandler.UpdateReview)
		userReviews.DELETE("/:id", c.ReviewHandler.DeleteReview)
		userReviews.GET("/me", c.ReviewHandler.ListMyReviews)
		userReviews.GET("/books/:book_id/reviews", c.ReviewHandler.GetBookReviews)
	}

	// Admin review routes
	// TODO: Add admin middleware
	adminReviews := v1.Group("/admin/reviews")
	{
		adminReviews.GET("", c.ReviewHandler.AdminListReviews)
		adminReviews.GET("/:id", c.ReviewHandler.AdminGetReview)
		adminReviews.GET("/statistics", c.ReviewHandler.AdminGetStatistics)
		adminReviews.PATCH("/:id/moderate", c.ReviewHandler.AdminModerateReview)
		adminReviews.PATCH("/:id/feature", c.ReviewHandler.AdminFeatureReview)
		adminReviews.DELETE("/:id", c.ReviewHandler.AdminDeleteReview)
	}
}

// ========================================
// NOTIFICATION ROUTES
// ========================================
func setupNotificationRoutes(v1 *gin.RouterGroup, c *container.Container) {
	// User notification routes
	notifications := v1.Group("/notifications")

	notifications.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
	{
		// Notifications
		notifications.GET("", c.NotificationHandler.ListNotifications)
		notifications.GET("/unread-count", c.NotificationHandler.GetUnreadCount)
		notifications.GET("/:id", c.NotificationHandler.GetNotification)
		notifications.POST("/mark-read", c.NotificationHandler.MarkAsRead)
		notifications.POST("/mark-all-read", c.NotificationHandler.MarkAllAsRead)
		notifications.DELETE("/:id", c.NotificationHandler.DeleteNotification)

		// Preferences
		notifications.GET("/preferences", c.PreferencesHandler.GetPreferences)
		notifications.PUT("/preferences", c.PreferencesHandler.UpdatePreferences)
	}

	// ================================================
	// ADMIN ENDPOINTS (Admin Only)
	// ================================================

	admin := v1.Group("/admin")
	// admin.Use(middleware.AdminMiddleware(c.Config.JWT.Secret))
	{
		// Templates
		templates := admin.Group("/notification-templates")
		{
			templates.POST("", c.TemplateHandler.CreateTemplate)
			templates.GET("", c.TemplateHandler.ListTemplates)
			templates.GET("/:id", c.TemplateHandler.GetTemplate)
			templates.PUT("/:id", c.TemplateHandler.UpdateTemplate)
			templates.DELETE("/:id", c.TemplateHandler.DeleteTemplate)
		}

		// Campaigns
		campaigns := admin.Group("/notification-campaigns")
		{
			campaigns.POST("", c.CampaignHandler.CreateCampaign)
			campaigns.GET("", c.CampaignHandler.ListCampaigns)
			campaigns.GET("/:id", c.CampaignHandler.GetCampaign)
			campaigns.POST("/:id/start", c.CampaignHandler.StartCampaign)
			campaigns.POST("/:id/cancel", c.CampaignHandler.CancelCampaign)
		}

	}
}

// ========================================
// HEALTH CHECK HANDLER
// ========================================
func healthCheckHandler(appCtx *container.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   getEnv("APP_VERSION", "1.0.0"),
			"services":  gin.H{},
		}

		// Check database
		dbStatus := "ok"
		if appCtx.DB == nil || appCtx.DB.Pool == nil {
			dbStatus = "disconnected"
			health["status"] = "degraded"
		} else {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			if err := appCtx.DB.HealthCheck(ctx); err != nil {
				dbStatus = fmt.Sprintf("error: %v", err)
				health["status"] = "degraded"
			}
		}

		// Check redis
		redisStatus := "ok"
		if appCtx.Cache == nil {
			redisStatus = "disconnected"
		} else {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			if err := appCtx.Cache.Ping(ctx); err != nil {
				redisStatus = fmt.Sprintf("error: %v", err)
			}
		}

		health["services"] = gin.H{
			"database": dbStatus,
			"redis":    redisStatus,
		}

		statusCode := http.StatusOK
		if dbStatus != "ok" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, health)
	}
}

// ========================================
// DATABASE TEST HANDLER
// ========================================
func databaseTestHandler(appCtx *container.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		if appCtx.DB == nil || appCtx.DB.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Database not connected",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var version string
		err := appCtx.DB.Pool.QueryRow(ctx, "SELECT version()").Scan(&version)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Query failed: %v", err),
			})
			return
		}

		stats := appCtx.DB.Pool.Stat()

		redisTest := "not tested"
		if appCtx.Cache != nil {
			testKey := "test:connection"
			testValue := map[string]string{"test": "data", "timestamp": time.Now().Format(time.RFC3339)}

			if err := appCtx.Cache.Set(ctx, testKey, testValue, 10*time.Second); err == nil {
				var retrieved map[string]string
				found, _ := appCtx.Cache.Get(ctx, testKey, &retrieved)
				if found {
					redisTest = "ok - set/get working"
				} else {
					redisTest = "warning - set ok but get failed"
				}
				_ = appCtx.Cache.Delete(ctx, testKey)
			} else {
				redisTest = fmt.Sprintf("error: %v", err)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Database test successful",
			"database": gin.H{
				"postgres_version": version,
				"pool_stats": gin.H{
					"total_connections":    stats.TotalConns(),
					"idle_connections":     stats.IdleConns(),
					"acquired_connections": stats.AcquiredConns(),
					"max_connections":      stats.MaxConns(),
				},
			},
			"cache": gin.H{
				"status": redisTest,
			},
		})
	}
}
