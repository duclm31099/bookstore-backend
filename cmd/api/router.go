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

	// ... middlewares giữ nguyên ...
	router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.CORS(),
	)
	// Cart middleware configuration
	cartMiddlewareConfig := middleware.DefaultCartMiddlewareConfig(c.CartService)

	// For development: disable Secure flag
	if os.Getenv("ENV") == "development" {
		cartMiddlewareConfig.CookieSecure = false
	}
	// ========================================
	// API V1 ROUTES
	// ========================================
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", healthCheckHandler(c))
		v1.GET("/db-test", databaseTestHandler(c))

		// ========================================
		// AUTH ROUTES (PUBLIC)
		// ========================================
		auth := v1.Group("/auth")
		auth.Use()
		{
			// FR-AUTH-001: User Registration
			auth.POST("/register", c.UserHandler.Register)
			auth.POST("/refresh", c.UserHandler.RefreshToken)

			// FR-AUTH-002: User Login
			auth.POST("/login", c.UserHandler.Login)

			// FR-AUTH-001: Email Verification
			auth.GET("/verify-email", c.UserHandler.VerifyEmail)
			auth.POST("/resend-verification", c.UserHandler.ResendVerification)

			// FR-AUTH-003: Password Reset
			auth.POST("/forgot-password", c.UserHandler.ForgotPassword)
			auth.POST("/reset-password", c.UserHandler.ResetPassword)
		}

		// ========================================
		// USER ROUTES (PROTECTED)
		// ========================================
		users := v1.Group("/users")
		// TODO: Add Auth middleware
		users.Use(
			middleware.AuthMiddleware(c.Config.JWT.Secret),
			middleware.IPExtractorMiddleware(),
		)
		{
			// Profile endpoints
			users.GET("/me", c.UserHandler.GetProfile)
			users.PUT("/me", c.UserHandler.UpdateProfile)
			users.PUT("/change-password", c.UserHandler.ChangePassword)
		}

		// ========================================
		// ADMIN ROUTES (PROTECTED + ADMIN ROLE)
		// ========================================
		admin := v1.Group("/admin")
		// TODO: Add Auth + Role middleware
		// admin.Use(middleware.Auth())
		// admin.Use(middleware.RequireRole("admin"))
		{
			// FR-ADM-003: User Management
			admin.GET("/users", c.UserHandler.ListUsers)
			admin.PUT("/users/:id/role", c.UserHandler.UpdateUserRole)
			admin.PUT("/users/:id/status", c.UserHandler.UpdateUserStatus)
		}

		// // --------------------------------------- CATEGORIES --------------------------------------
		category := v1.Group("/categories")
		{
			category.POST("", c.CategoryHandler.Create)
			// Read
			category.GET("", c.CategoryHandler.GetAll)                       // List
			category.GET("/tree", c.CategoryHandler.GetTree)                 // Tree
			category.GET("/:id", c.CategoryHandler.GetByID)                  // Get by ID
			category.GET("/:id/breadcrumb", c.CategoryHandler.GetBreadcrumb) // Breadcrumb
			category.GET("/by-slug/:slug", c.CategoryHandler.GetBySlug)      // Get by slug

			// Update
			category.PUT("/:id", c.CategoryHandler.Update)                 // Update
			category.PATCH("/:id/parent", c.CategoryHandler.MoveToParent)  // Move to parent
			category.POST("/:id/activate", c.CategoryHandler.Activate)     // Activate
			category.POST("/:id/deactivate", c.CategoryHandler.Deactivate) // Deactivate

			// Delete
			category.DELETE("/:id", c.CategoryHandler.Delete)      // Delete single
			category.DELETE("/bulk", c.CategoryHandler.BulkDelete) // Bulk delete

			// Bulk operations
			category.POST("/bulk/activate", c.CategoryHandler.BulkActivate)     // Bulk activate
			category.POST("/bulk/deactivate", c.CategoryHandler.BulkDeactivate) // Bulk deactivate

			// Book-related
			category.GET("/:id/books", c.CategoryHandler.GetBooksInCategory)        // Get books
			category.GET("/:id/book-count", c.CategoryHandler.GetCategoryBookCount) // Book count
		}

		// --------------------------------------- AUTHORS --------------------------------------
		author := v1.Group("/authors")
		{
			// Create
			author.POST("", c.AuthorHandler.Create)
			// Read single
			author.GET("/:id", c.AuthorHandler.GetByID)
			author.GET("/slug/:slug", c.AuthorHandler.GetBySlug)
			// Read multiple
			author.GET("", c.AuthorHandler.GetAll)
			author.GET("/search", c.AuthorHandler.Search)
			// Update
			author.PUT("/:id", c.AuthorHandler.Update)
			// Delete
			author.DELETE("/:id", c.AuthorHandler.Delete)
			author.DELETE("/bulk", c.AuthorHandler.BulkDelete)
			// Books
			author.GET("/:id/books", c.AuthorHandler.GetWithBookCount)
		}

		//  --------------------------------------- PUBLISHER ------------------------
		publisherGroup := v1.Group("/publishers")
		{
			// Create publisher
			publisherGroup.POST("", c.PublisherHandler.CreatePublisher)

			// Get all publishers with pagination
			publisherGroup.GET("", c.PublisherHandler.ListPublishers)

			// Get publisher with books (needs to come BEFORE /:id)
			publisherGroup.GET("/books", c.PublisherHandler.ListPublishersWithBooks)

			// Get publisher by slug
			publisherGroup.GET("/slug/:slug", c.PublisherHandler.GetPublisherBySlug)

			// Get publisher by ID
			publisherGroup.GET("/:id", c.PublisherHandler.GetPublisher)

			// Get publisher with books by ID
			publisherGroup.GET("/:id/books", c.PublisherHandler.GetPublisherWithBooks)

			// Update publisher
			publisherGroup.PUT("/:id", c.PublisherHandler.UpdatePublisher)

			// Delete publisher
			publisherGroup.DELETE("/:id", c.PublisherHandler.DeletePublisher)
		}

		// ---------------------------------- ADDRESS ----------------------------------
		addressGroup := v1.Group("/addresses")
		addressGroup.Use(middleware.AuthMiddleware(c.Config.JWT.Secret)) // User must be authenticated
		{
			// Create address
			addressGroup.POST("", c.AddressHandler.CreateAddress)

			// Get all user addresses
			addressGroup.GET("", c.AddressHandler.ListUserAddresses)

			// Get default address
			addressGroup.GET("/default", c.AddressHandler.GetDefaultAddress)

			// Get address by ID
			addressGroup.GET("/:id", c.AddressHandler.GetAddressById)

			// Update address
			addressGroup.PUT("/:id", c.AddressHandler.UpdateAddress)

			// Set address as default
			addressGroup.PUT("/:id/set-default", c.AddressHandler.SetDefaultAddress)

			// Unset default (remove default flag)
			addressGroup.PUT("/:id/unset-default", c.AddressHandler.UnsetDefaultAddress)

			// Delete address
			addressGroup.DELETE("/:id", c.AddressHandler.DeleteAddress)
		}

		// ========== ADMIN ADDRESS ROUTES ==========
		adminAddressGroup := v1.Group("/admin/addresses")
		// adminAddressGroup.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
		// adminAddressGroup.Use(middleware.AdminMiddleware()) // Only admin
		{
			// Get all addresses (paginated)
			adminAddressGroup.GET("", c.AddressHandler.ListAllAddresses)

			// Get address with user info
			adminAddressGroup.GET("/:id", c.AddressHandler.GetAddressWithUser)
		}

		// ------------------------------ BOOK ROUTES ---------------------------------------
		bookRouter := v1.Group("/books")
		{
			bookRouter.GET("", c.BookHandler.ListBooks)
			bookRouter.GET("/:id", c.BookHandler.GetBookDetail)
			bookRouter.PUT("/:id", c.BookHandler.UpdateBook)
			bookRouter.DELETE("/:id", c.BookHandler.DeleteBook)
			bookRouter.POST("", c.BookHandler.CreateBook)
			bookRouter.GET("/search", c.BookHandler.SearchBooks)
			// In admin routes
			bookRouter.POST("/bulk-import", c.BulkImportHandler.ImportBooks)
			bookRouter.GET("/export", c.BookHandler.ExportBooks)
		}

		// ----------------------------------- WAREHOUSE ----------------------------
		warehouses := v1.Group("/warehouses")
		{
			warehouses.POST("", c.WarehouseHandler.CreateWarehouse)          // Tạo kho mới
			warehouses.PUT(":id", c.WarehouseHandler.UpdateWarehouse)        // Cập nhật thông tin kho
			warehouses.DELETE(":id", c.WarehouseHandler.SoftDeleteWarehouse) // Xóa (soft)
			warehouses.GET(":id", c.WarehouseHandler.GetWarehouseByID)       // Lấy chi tiết kho (by id)
			warehouses.GET("", c.WarehouseHandler.ListWarehouses)            // List warehouse (filter, paging)

			warehouses.GET(":id", c.WarehouseHandler.GetWarehouseByID)                              // Xem chi tiết kho
			warehouses.GET("/code/:code", c.WarehouseHandler.GetWarehouseByCode)                    // Lấy kho theo code
			warehouses.GET("", c.WarehouseHandler.ListActiveWarehouses)                             // List các kho đang hoạt động (không phân trang)
			warehouses.GET("/nearest-with-stock", c.WarehouseHandler.FindNearestWarehouseWithStock) // Tìm kho gần nhất còn stock cho book
			warehouses.GET("/validate-stock", c.WarehouseHandler.ValidateWarehouseHasStock)         // Validate kho còn hàng theo đầu sách (optional)

		}
		// ---------------------------------- INVENTORY ROUTES ------------------------------------

		inventoryRouter := v1.Group("/inventories")
		{
			// ========================================
			// INVENTORY CRUD
			// ========================================

			// Create inventory for warehouse(s)
			inventoryRouter.POST("", c.InventoryHandler.CreateInventory)

			// Get inventory by composite key (warehouse_id + book_id)
			inventoryRouter.GET("/:warehouse_id/:book_id", c.InventoryHandler.GetInventoryByWarehouseAndBook)

			// List inventories with filters
			inventoryRouter.GET("", c.InventoryHandler.ListInventories)

			// Update inventory (composite key)
			inventoryRouter.PATCH("/:warehouse_id/:book_id", c.InventoryHandler.UpdateInventory)

			// Delete inventory (composite key)
			inventoryRouter.DELETE("/:warehouse_id/:book_id", c.InventoryHandler.DeleteInventory)

			// ========================================
			// STOCK OPERATIONS (FR-INV-003)
			// ========================================

			// Reserve stock for checkout (15min timeout)
			inventoryRouter.POST("/reserve", c.InventoryHandler.ReserveStock)

			// Release reserved stock (cancel/timeout)
			inventoryRouter.POST("/release", c.InventoryHandler.ReleaseStock)

			// Complete sale after payment success
			inventoryRouter.POST("/complete-sale", c.InventoryHandler.CompleteSale)

			// ========================================
			// WAREHOUSE SELECTION (FR-INV-002)
			// ========================================

			// Find nearest warehouse with stock
			inventoryRouter.POST("/find-warehouse", c.InventoryHandler.FindOptimalWarehouse)

			// Check availability for order items
			inventoryRouter.POST("/check-availability", c.InventoryHandler.CheckAvailability)

			// Get total stock summary for book (all warehouses)
			inventoryRouter.GET("/summary/:book_id", c.InventoryHandler.GetStockSummary)

			// ========================================
			// STOCK ADJUSTMENT (FR-INV-005)
			// ========================================

			// Manual stock adjustment (admin only)
			inventoryRouter.POST("/adjust", c.InventoryHandler.AdjustStock)

			// Restock from supplier
			inventoryRouter.POST("/restock", c.InventoryHandler.RestockInventory)

			// Bulk update from CSV (FR-INV-006)
			inventoryRouter.POST("/bulk-update", c.InventoryHandler.BulkUpdateStock)
			inventoryRouter.GET("/bulk-update/:job_id", c.InventoryHandler.GetBulkUpdateStatus)

			// ========================================
			// AUDIT TRAIL (FR-INV-005)
			// ========================================

			// Get audit log with filters
			inventoryRouter.GET("/audit", c.InventoryHandler.GetAuditTrail)

			// Get inventory history for specific warehouse+book
			inventoryRouter.GET("/:warehouse_id/:book_id/history", c.InventoryHandler.GetInventoryHistory)

			// Export audit log to CSV/Excel
			inventoryRouter.POST("/audit/export", c.InventoryHandler.ExportAuditLog)

			// ========================================
			// ALERTS (FR-INV-004)
			// ========================================

			// Get low stock alerts (unresolved)
			inventoryRouter.GET("/alerts/low-stock", c.InventoryHandler.GetLowStockAlerts)

			// Get out of stock items
			inventoryRouter.GET("/alerts/out-of-stock", c.InventoryHandler.GetOutOfStockItems)

			// Mark alert as resolved (admin)
			inventoryRouter.PATCH("/alerts/:alert_id/resolve", c.InventoryHandler.MarkAlertResolved)

			// ========================================
			// DASHBOARD & ANALYTICS
			// ========================================

			// Comprehensive dashboard
			inventoryRouter.GET("/dashboard", c.InventoryHandler.GetDashboardSummary)

			// Movement trends (last N days)
			// inventoryRouter.GET("/trends", c.InventoryHandler.GetMovementTrends)

			// Reservation analytics
			inventoryRouter.GET("/analysis/reservations", c.InventoryHandler.GetReservationAnalysis)

			// Inventory value (financial reporting)
			// inventoryRouter.GET("/value", c.InventoryHandler.GetInventoryValue)
		}

		// ========================================
		// WAREHOUSE MANAGEMENT (Separate Group)
		// ========================================

		warehouseRouter := v1.Group("/warehouses")
		{
			// Create warehouse (admin only)
			warehouseRouter.POST("", c.InventoryHandler.CreateWarehouse)

			// List warehouses with filters
			warehouseRouter.GET("", c.InventoryHandler.ListWarehouses)

			// Get warehouse by ID
			warehouseRouter.GET("/:id", c.InventoryHandler.GetWarehouseByID)

			// Update warehouse (admin only)
			warehouseRouter.PATCH("/:id", c.InventoryHandler.UpdateWarehouse)

			// Deactivate warehouse (admin only)
			warehouseRouter.DELETE("/:id", c.InventoryHandler.DeactivateWarehouse)

			// Get warehouse performance metrics
			warehouseRouter.GET("/:id/performance", c.InventoryHandler.GetWarehousePerformance)
		}

		// ===================== CART =========================
		cartRoutes := v1.Group("/cart")
		// 🔑 KEY: Use OptionalAuthMiddleware instead of AuthMiddleware
		// This allows both authenticated and anonymous users
		cartRoutes.Use(middleware.OptionalAuthMiddleware(c.Config.JWT.Secret))

		// 🛒 Then apply CartMiddleware
		// CartMiddleware handles user_id (from OptionalAuth) + session_id
		cartRoutes.Use(middleware.CartMiddleware(cartMiddlewareConfig))
		{
			// Step 1 APIs
			cartRoutes.GET("", c.CartHandler.GetCart)
			cartRoutes.POST("/items", c.CartHandler.AddItem)
			cartRoutes.GET("/items", c.CartHandler.ListItems)

			cartRoutes.PUT("/items/:item_id", c.CartHandler.UpdateItemQuantity) // Update qty
			cartRoutes.DELETE("/items/:item_id", c.CartHandler.RemoveItem)      // Remove item
			cartRoutes.DELETE("", c.CartHandler.ClearCart)

			cartRoutes.POST("/validate", c.CartHandler.ValidateCart)
			cartRoutes.POST("/apply-promotion", c.CartHandler.ApplyPromoCode)
			cartRoutes.DELETE("/remove-promotion", c.CartHandler.RemovePromoCode)
			cartRoutes.POST("/checkout", c.CartHandler.Checkout)
		}

		//  ======================== PROMOTION ========================================
		promotion := v1.Group("/promotion")

		// Public endpoints
		promotion.POST("/validate", c.PublicProHandler.ValidatePromotion) // POST /v1/promotions/validate
		promotion.GET("", c.PublicProHandler.ListActivePromotions)        // GET /v1/promotions

		promotion.POST("/create", c.AdminProHandler.CreatePromotion)            // POST /v1/admin/promotions
		promotion.GET("/list-promotion", c.AdminProHandler.ListPromotions)      // GET /v1/admin/promotions
		promotion.GET("/:id", c.AdminProHandler.GetPromotionByID)               // GET /v1/admin/promotions/:id
		promotion.PUT("/:id", c.AdminProHandler.UpdatePromotion)                // PUT /v1/admin/promotions/:id
		promotion.PATCH("/:id/status", c.AdminProHandler.UpdatePromotionStatus) // PATCH /v1/admin/promotions/:id/status
		promotion.DELETE("/:id", c.AdminProHandler.DeletePromotion)             // DELETE /v1/admin/promotions/:id
		promotion.GET("/:id/usage", c.AdminProHandler.GetUsageHistory)          // GET /v1/admin/promotions/:id/usage
		promotion.POST("/:id/export", c.AdminProHandler.ExportUsageReport)      // POST /v1/admin/promotions/:id/export

		// ========================================
		// 📦 ORDER ROUTES (USER - PROTECTED)
		// ========================================
		orderRoutes := v1.Group("/orders")
		orderRoutes.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
		{
			// ========================================
			// ORDER CREATION & MANAGEMENT
			// ========================================

			// Create order from cart
			orderRoutes.POST("", c.OrderHandler.CreateOrder)

			// Get order list for current user (with filters)
			orderRoutes.GET("", c.OrderHandler.ListOrders)

			// Get order detail by ID
			orderRoutes.GET("/:id", c.OrderHandler.GetOrderDetail)

			// Cancel order (user-initiated)
			orderRoutes.POST("/:id/cancel", c.OrderHandler.CancelOrder)

			// ========================================
			// ORDER TRACKING
			// ========================================

			// Track order by order number
			orderRoutes.GET("/track/:order_number", c.OrderHandler.GetOrderByNumber)

			// Get order status history
			// orderRoutes.GET("/:id/history", c.OrderHandler.GetOrderStatusHistory)

		}

		// ========================================
		// 🛡️ ADMIN ORDER ROUTES (PROTECTED + ADMIN)
		// ========================================
		adminOrderRoutes := v1.Group("/admin/orders")
		// TODO: Add admin middleware
		// adminOrderRoutes.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
		// adminOrderRoutes.Use(middleware.RequireRole("admin"))
		{
			// ========================================
			// ORDER MANAGEMENT
			// ========================================

			// List all orders with filters & pagination
			adminOrderRoutes.GET("", c.OrderHandler.ListAllOrders)

			// Update order status
			adminOrderRoutes.PATCH("/:id/status", c.OrderHandler.UpdateOrderStatus)

			// ========================================
			// SHIPPING MANAGEMENT
			// ========================================

			// // Assign order to warehouse
			// adminOrderRoutes.PATCH("/:id/warehouse", c.OrderHandler.AssignWarehouse)

			// // Update shipping information
			// adminOrderRoutes.PATCH("/:id/shipping", c.OrderHandler.UpdateShippingInfo)

			// // Mark as shipped
			// adminOrderRoutes.POST("/:id/ship", c.OrderHandler.MarkAsShipped)

			// // Mark as delivered
			// adminOrderRoutes.POST("/:id/deliver", c.OrderHandler.MarkAsDelivered)

			// // ========================================
			// // ANALYTICS & REPORTING
			// // ========================================

			// // Get order statistics
			// adminOrderRoutes.GET("/stats/summary", c.OrderHandler.GetOrderStatistics)

			// // Get order analytics by date range
			// adminOrderRoutes.GET("/stats/analytics", c.OrderHandler.GetOrderAnalytics)

			// // Export orders to CSV/Excel
			// adminOrderRoutes.POST("/export", c.OrderHandler.ExportOrders)
		}

		// ========================================
		// 💳 PAYMENT ROUTES (USER - PROTECTED)
		// ========================================
		paymentRoutes := v1.Group("/payments")
		paymentRoutes.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
		{
			// ========================================
			// PAYMENT CREATION
			// ========================================

			// Create payment (initiate payment gateway)
			paymentRoutes.POST("/create", c.PaymentHandler.CreatePayment)

			// Get payment status (for polling)
			paymentRoutes.GET("/:payment_id", c.PaymentHandler.GetPaymentStatus)

			// List user's payment history
			paymentRoutes.GET("", c.PaymentHandler.ListUserPayments)

			// ========================================
			// REFUND MANAGEMENT
			// ========================================

			// Request refund for a payment
			paymentRoutes.POST("/:payment_id/refund-request", c.PaymentHandler.RequestRefund)

			// Get refund request status
			paymentRoutes.GET("/:payment_id/refund-request", c.PaymentHandler.GetRefundStatus)
		}

		// ========================================
		// 🔔 WEBHOOK ROUTES (PUBLIC - NO AUTH)
		// ========================================
		webhookRoutes := v1.Group("/webhooks")
		{
			// VNPay IPN webhook
			webhookRoutes.GET("/vnpay", c.PaymentHandler.VNPayWebhook)
			webhookRoutes.POST("/vnpay", c.PaymentHandler.VNPayWebhook)

			// Momo IPN webhook
			webhookRoutes.POST("/momo", c.PaymentHandler.MomoWebhook)
		}

		// ========================================
		// 🛡️ ADMIN PAYMENT ROUTES (PROTECTED + ADMIN)
		// ========================================
		adminPaymentRoutes := v1.Group("/admin/payments")
		// TODO: Add admin middleware
		// adminPaymentRoutes.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
		// adminPaymentRoutes.Use(middleware.RequireRole("admin"))
		{
			// ========================================
			// PAYMENT MANAGEMENT
			// ========================================

			// List all payments with filters
			adminPaymentRoutes.GET("", c.PaymentHandler.AdminListPayments)

			// Get payment detail (with full gateway info)
			adminPaymentRoutes.GET("/:payment_id", c.PaymentHandler.AdminGetPaymentDetail)

			// Manual reconciliation (fix failed webhook)
			adminPaymentRoutes.POST("/:payment_id/reconcile", c.PaymentHandler.AdminReconcilePayment)

			// ========================================
			// REFUND MANAGEMENT
			// ========================================

			// List pending refund requests
			adminPaymentRoutes.GET("/refunds/pending", c.PaymentHandler.AdminListPendingRefunds)

			// Get refund request detail
			adminPaymentRoutes.GET("/refunds/:refund_id", c.PaymentHandler.AdminGetRefundDetail)

			// Approve refund request
			adminPaymentRoutes.POST("/refunds/:refund_id/approve", c.PaymentHandler.AdminApproveRefund)

			// Reject refund request
			adminPaymentRoutes.POST("/refunds/:refund_id/reject", c.PaymentHandler.AdminRejectRefund)

			//  ===================================== REVIEW ====================
			reviewRoutes := v1.Group("/reviews")
			{
				// Public endpoints (no auth)
				reviewRoutes.GET("", c.ReviewHandler.ListReviews)   // List reviews with filters
				reviewRoutes.GET("/:id", c.ReviewHandler.GetReview) // Get review by ID
			}

			// User review endpoints (auth required)
			userReviewRoutes := v1.Group("/reviews")
			userReviewRoutes.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
			{
				userReviewRoutes.POST("", c.ReviewHandler.CreateReview)       // Create review
				userReviewRoutes.PUT("/:id", c.ReviewHandler.UpdateReview)    // Update review
				userReviewRoutes.DELETE("/:id", c.ReviewHandler.DeleteReview) // Delete review
				userReviewRoutes.GET("/me", c.ReviewHandler.ListMyReviews)    // List my reviews
			}

			// Book-specific review endpoints (public)
			userReviewRoutes.GET("/books/:book_id/reviews", c.ReviewHandler.GetBookReviews)

			// ========================================
			// ADMIN REVIEW ROUTES
			// ========================================

			adminReviewRoutes := v1.Group("/admin/reviews")
			// TODO: Add admin middleware
			// adminReviewRoutes.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
			// adminReviewRoutes.Use(middleware.RequireRole("admin"))
			{
				// List & View
				adminReviewRoutes.GET("", c.ReviewHandler.AdminListReviews)              // List all reviews
				adminReviewRoutes.GET("/:id", c.ReviewHandler.AdminGetReview)            // Get review detail
				adminReviewRoutes.GET("/statistics", c.ReviewHandler.AdminGetStatistics) // Dashboard stats

				// Moderation
				adminReviewRoutes.PATCH("/:id/moderate", c.ReviewHandler.AdminModerateReview) // Approve/Hide
				adminReviewRoutes.PATCH("/:id/feature", c.ReviewHandler.AdminFeatureReview)   // Feature
				adminReviewRoutes.DELETE("/:id", c.ReviewHandler.AdminDeleteReview)           // Delete
			}
		}
	}

	return router
}

// ========================================
// HEALTH CHECK HANDLER
// ========================================
// healthCheckHandler trả về handler function với closure over appCtx
// Pattern này cho phép inject dependencies vào handler
func healthCheckHandler(appCtx *container.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Response structure
		health := gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   getEnv("APP_VERSION", "1.0.0"),
			"services":  gin.H{},
		}

		// ========================================
		// CHECK DATABASE
		// ========================================
		dbStatus := "ok"
		if appCtx.DB == nil || appCtx.DB.Pool == nil {
			dbStatus = "disconnected"
			health["status"] = "degraded"
		} else {
			// Tạo context với timeout 2s cho health check
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			if err := appCtx.DB.HealthCheck(ctx); err != nil {
				dbStatus = fmt.Sprintf("error: %v", err)
				health["status"] = "degraded"
			}
		}

		// ========================================
		// CHECK REDIS
		// ========================================
		redisStatus := "ok"
		if appCtx.Cache == nil {
			redisStatus = "disconnected"
			// Redis failure không làm service unhealthy (non-critical)
		} else {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			// Ping qua cache interface
			if err := appCtx.Cache.Ping(ctx); err != nil {
				redisStatus = fmt.Sprintf("error: %v", err)
				// Redis failure không làm status = "degraded"
				// Vì cache là optional, service vẫn hoạt động được
			}
		}

		// Aggregate service statuses
		health["services"] = gin.H{
			"database": dbStatus,
			"redis":    redisStatus,
		}

		// ========================================
		// RETURN RESPONSE
		// ========================================
		// 503 Service Unavailable nếu database down (critical)
		// 200 OK nếu database up (Redis down vẫn OK)
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
// databaseTestHandler test raw database queries (development/debugging only)
func databaseTestHandler(appCtx *container.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Guard: check database availability
		if appCtx.DB == nil || appCtx.DB.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Database not connected",
			})
			return
		}

		// ========================================
		// TEST QUERY: Get PostgreSQL Version
		// ========================================
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

		// ========================================
		// GET CONNECTION POOL STATS
		// ========================================
		stats := appCtx.DB.Pool.Stat()

		// ========================================
		// TEST REDIS CACHE
		// ========================================
		redisTest := "not tested"
		if appCtx.Cache != nil {
			// Test Set
			testKey := "test:connection"
			testValue := map[string]string{"test": "data", "timestamp": time.Now().Format(time.RFC3339)}

			if err := appCtx.Cache.Set(ctx, testKey, testValue, 10*time.Second); err == nil {
				// Test Get
				var retrieved map[string]string
				found, _ := appCtx.Cache.Get(ctx, testKey, &retrieved)
				if found {
					redisTest = "ok - set/get working"
				} else {
					redisTest = "warning - set ok but get failed"
				}

				// Cleanup
				_ = appCtx.Cache.Delete(ctx, testKey)
			} else {
				redisTest = fmt.Sprintf("error: %v", err)
			}
		}

		// ========================================
		// RETURN TEST RESULTS
		// ========================================
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
