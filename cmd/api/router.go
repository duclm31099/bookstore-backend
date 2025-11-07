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
		users.Use(middleware.AuthMiddleware(c.Config.JWT.Secret))
		{
			// Profile endpoints
			users.GET("/me", c.UserHandler.GetProfile)
			users.PUT("/me", c.UserHandler.UpdateProfile)
			users.PUT("/me/password", c.UserHandler.ChangePassword)
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
		}

		// ---------------------------------- INVENTORY ROUTES ------------------------------------

		inventoryRouter := v1.Group("/inventories")

		{
			inventoryRouter.POST("", c.InventoryHandler.CreateInventory)

			// Read
			inventoryRouter.GET("/:id", c.InventoryHandler.GetInventoryByID)
			inventoryRouter.GET("/search", c.InventoryHandler.SearchInventory)
			inventoryRouter.GET("", c.InventoryHandler.ListInventories)
			inventoryRouter.PUT("/:id", c.InventoryHandler.UpdateInventory)
			inventoryRouter.DELETE("/:id", c.InventoryHandler.DeleteInventory)

			inventoryRouter.POST("/reverse", c.InventoryHandler.ReserveStock)
			inventoryRouter.POST("/release", c.InventoryHandler.ReleaseStock)

			inventoryRouter.POST("/check-availability", c.InventoryHandler.CheckAvailability)
			inventoryRouter.GET("/summary", c.InventoryHandler.GetStockSummary)

			inventoryRouter.POST("/:inventory_id/movements", c.InventoryHandler.CreateMovement)
			inventoryRouter.GET("/movements", c.InventoryHandler.ListMovements)
			inventoryRouter.GET("/movements/stats", c.InventoryHandler.GetMovementStats)

			inventoryRouter.GET("/dashboard", c.InventoryHandler.GetInventoryDashboard)
			inventoryRouter.GET("/alerts/low-stock", c.InventoryHandler.GetLowStockAlerts)
			inventoryRouter.GET("/alerts/out-of-stock", c.InventoryHandler.GetOutOfStockItems)
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
			cartRoutes.POST("/promo", c.CartHandler.ApplyPromoCode)
			cartRoutes.DELETE("/promo", c.CartHandler.RemovePromoCode)
			cartRoutes.POST("/checkout", c.CartHandler.Checkout)
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
