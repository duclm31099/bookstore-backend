package test_domains

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"bookstore-backend/pkg/container"
)

var (
	testServer    *httptest.Server
	testContainer *container.Container
)

func TestMain(m *testing.M) {
	log.Println("🧪 Setting up test environment...")

	envPath := filepath.Join("..", ".env.test")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("⚠️  No .env.test found, trying .env")
		godotenv.Load(filepath.Join("..", ".env"))
	}

	var err error
	testContainer, err = container.NewContainer()
	if err != nil {
		log.Fatalf("❌ Failed to init container: %v", err)
	}

	cleanDatabase()

	gin.SetMode(gin.TestMode)
	router := setupTestRouter(testContainer)
	testServer = httptest.NewServer(router)

	log.Printf("✅ Test server ready at %s", testServer.URL)

	exitCode := m.Run()

	testServer.Close()
	testContainer.Cleanup()
	log.Println("✅ Cleanup complete")

	os.Exit(exitCode)
}

func setupTestRouter(c *container.Container) *gin.Engine {
	router := gin.New()

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"status": "ok"})
		})

		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", c.UserHandler.Register)
			auth.POST("/login", c.UserHandler.Login)
		}

		// User routes
		users := v1.Group("/users")
		users.Use(mockAuthMiddleware())
		{
			users.GET("/me", c.UserHandler.GetProfile)
			users.PUT("/me", c.UserHandler.UpdateProfile)
			users.PUT("/me/password", c.UserHandler.ChangePassword)
		}

		// Book routes
		books := v1.Group("/books")
		{
			books.GET("", c.BookHandler.ListBooks)
			books.GET("/search", c.BookHandler.SearchBooks)
			books.GET("/:id", c.BookHandler.GetBookDetail)
			books.POST("", c.BookHandler.CreateBook)
			books.PUT("/:id", c.BookHandler.UpdateBook)
			books.DELETE("/:id", c.BookHandler.DeleteBook)
		}

		// Category routes
		categories := v1.Group("/categories")
		{
			categories.POST("", c.CategoryHandler.Create)
			categories.GET("", c.CategoryHandler.GetAll)
			categories.GET("/:id", c.CategoryHandler.GetByID)
		}

		// Author routes
		authors := v1.Group("/authors")
		{
			authors.POST("", c.AuthorHandler.Create)
			authors.GET("", c.AuthorHandler.GetAll)
			authors.GET("/:id", c.AuthorHandler.GetByID)
		}

		// Publisher routes
		publishers := v1.Group("/publishers")
		{
			publishers.POST("", c.PublisherHandler.CreatePublisher)
			publishers.GET("", c.PublisherHandler.ListPublishers)
			publishers.GET("/:id", c.PublisherHandler.GetPublisher)
		}

		// Warehouse routes
		warehouses := v1.Group("/warehouses")
		{
			warehouses.POST("", c.WarehouseHandler.CreateWarehouse)
			warehouses.GET("", c.WarehouseHandler.ListWarehouses)
			warehouses.GET("/:id", c.WarehouseHandler.GetWarehouseByID)
		}

		// Inventory routes
		inventory := v1.Group("/inventories")
		{
			inventory.POST("", c.InventoryHandler.CreateInventory)
			inventory.GET("", c.InventoryHandler.ListInventories)
			inventory.GET("/:warehouse_id/:book_id", c.InventoryHandler.GetInventoryByWarehouseAndBook)
			inventory.PATCH("/:warehouse_id/:book_id", c.InventoryHandler.UpdateInventory)
			inventory.DELETE("/:warehouse_id/:book_id", c.InventoryHandler.DeleteInventory)
			inventory.POST("/reserve", c.InventoryHandler.ReserveStock)
			inventory.POST("/release", c.InventoryHandler.ReleaseStock)
			inventory.POST("/complete-sale", c.InventoryHandler.CompleteSale)
			inventory.POST("/find-warehouse", c.InventoryHandler.FindOptimalWarehouse)
		}

		// Cart routes
		cart := v1.Group("/cart")
		cart.Use(mockAuthMiddleware())
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

		// Promotion routes
		promotion := v1.Group("/promotion")
		{
			promotion.POST("/validate", c.PublicProHandler.ValidatePromotion)
			promotion.GET("", c.PublicProHandler.ListActivePromotions)
			promotion.POST("/create", c.AdminProHandler.CreatePromotion)
			promotion.GET("/:id", c.AdminProHandler.GetPromotionByID)
		}

		// Order routes
		orders := v1.Group("/orders")
		orders.Use(mockAuthMiddleware())
		{
			// orders.POST("", c.OrderHandler.CreateOrder)
			orders.GET("", c.OrderHandler.ListOrders)
			orders.GET("/:id", c.OrderHandler.GetOrderDetail)
			orders.POST("/:id/cancel", c.OrderHandler.CancelOrder)
			orders.GET("/track/:order_number", c.OrderHandler.GetOrderByNumber)
		}

		// Payment routes
		payments := v1.Group("/payments")
		payments.Use(mockAuthMiddleware())
		{
			payments.POST("/create", c.PaymentHandler.CreatePayment)
			payments.GET("/:payment_id", c.PaymentHandler.GetPaymentStatus)
			payments.GET("", c.PaymentHandler.ListUserPayments)
			payments.POST("/:payment_id/refund-request", c.PaymentHandler.RequestRefund)
			payments.GET("/:payment_id/refund-request", c.PaymentHandler.GetRefundStatus)
		}

		// Webhook routes (no auth)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/vnpay", c.PaymentHandler.VNPayWebhook)
			webhooks.POST("/momo", c.PaymentHandler.MomoWebhook)
		}
	}

	return router
}

func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "message": "Unauthorized"})
			return
		}

		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "message": "Invalid token format"})
			return
		}

		secret := os.Getenv("JWT_SECRET")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "message": "Invalid token"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", userID.(string))
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(401, gin.H{"success": false, "message": "Invalid claims"})
	}
}

// ========================================
// DATABASE CLEANUP
// ========================================

func cleanDatabase() {
	ctx := context.Background()

	// Truncate all tables in correct order (respecting foreign keys)
	tables := []string{
		// "order_items",
		// "orders",
		// "payments",
		// "refund_requests",
		// "cart_items",
		// "carts",
		// "inventories",
		// "inventory_audit_logs",
		// "inventory_alerts",
		// "reviews",
		// "promotion_usage",
		// "promotions",
		// "books",
		// "categories",
		// "authors",
		// "publishers",
		// "warehouses",
		// "addresses",
		// "users",
	}

	for _, table := range tables {
		_, _ = testContainer.DB.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
	}
}

func cleanBeforeTest(t *testing.T) {
	t.Helper()
	cleanDatabase()
}

// ========================================
// HTTP REQUEST HELPERS
// ========================================

func makeRequest(t *testing.T, method, path string, body interface{}, headers map[string]string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, _ := http.NewRequest(method, testServer.URL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, _ := client.Do(req)
	return resp
}

func parseJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func assertSuccess(t *testing.T, resp *http.Response, status int) map[string]interface{} {
	t.Helper()
	require.Equal(t, status, resp.StatusCode)
	result := parseJSON(t, resp)
	require.True(t, result["success"].(bool))
	return result
}

func assertError(t *testing.T, resp *http.Response, status int) map[string]interface{} {
	t.Helper()
	require.Equal(t, status, resp.StatusCode)
	result := parseJSON(t, resp)
	require.False(t, result["success"].(bool))
	return result
}

// ========================================
// USER HELPERS
// ========================================

func createVerifiedUser(t *testing.T, email, password string) string {
	t.Helper()
	payload := map[string]interface{}{
		"email":     email,
		"password":  password,
		"full_name": "Test User",
	}
	resp := makeRequest(t, "POST", "/api/v1/auth/register", payload, nil)
	result := assertSuccess(t, resp, 201)
	data := result["data"].(map[string]interface{})
	userID := data["id"].(string)

	ctx := context.Background()
	testContainer.DB.Pool.Exec(ctx, "UPDATE users SET is_verified = true WHERE id = $1", userID)
	return userID
}

func loginUser(t *testing.T, email, password string) string {
	t.Helper()
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}
	resp := makeRequest(t, "POST", "/api/v1/auth/login", payload, nil)
	result := assertSuccess(t, resp, 200)
	data := result["data"].(map[string]interface{})
	return data["access_token"].(string)
}

func createAndLoginUser(t *testing.T, email, password string) (userID string, token string) {
	t.Helper()
	userID = createVerifiedUser(t, email, password)
	token = loginUser(t, email, password)
	return
}

// ========================================
// TEST DATA HELPERS
// ========================================

// CreateTestCategory creates a test category
func createTestCategory(t *testing.T, name string) string {
	t.Helper()
	ctx := context.Background()

	var categoryID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO categories (name, slug, description, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id
	`, name, name+"-slug", "Test category").Scan(&categoryID)

	require.NoError(t, err)
	return categoryID
}

// CreateTestAuthor creates a test author
func createTestAuthor(t *testing.T, name string) string {
	t.Helper()
	ctx := context.Background()

	var authorID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO authors (name, slug, bio)
		VALUES ($1, $2, $3)
		RETURNING id
	`, name, name+"-slug", "Test author bio").Scan(&authorID)

	require.NoError(t, err)
	return authorID
}

// CreateTestPublisher creates a test publisher
func createTestPublisher(t *testing.T, name string) string {
	t.Helper()
	ctx := context.Background()

	var publisherID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO publishers (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, name, name+"-slug", "Test publisher").Scan(&publisherID)

	require.NoError(t, err)
	return publisherID
}

// CreateTestBook creates a test book
func createTestBook(t *testing.T, title string, price float64, categoryID, authorID, publisherID string) string {
	t.Helper()
	ctx := context.Background()

	var bookID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO books (
			title, slug, isbn, description, price, stock_quantity,
			category_id, author_id, publisher_id, language, page_count,
			publication_year, is_active
		)
		VALUES ($1, $2, $3, $4, $5, 100, $6, $7, $8, 'vi', 200, 2024, true)
		RETURNING id
	`, title, title+"-slug", "ISBN-"+uuid.New().String()[:8], "Test book", price,
		categoryID, authorID, publisherID).Scan(&bookID)

	require.NoError(t, err)
	return bookID
}

// CreateTestWarehouse creates a test warehouse
func createTestWarehouse(t *testing.T, name string, lat, lng float64) string {
	t.Helper()
	ctx := context.Background()

	var warehouseID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO warehouses (
			name, code, address, latitude, longitude, is_active
		)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id
	`, name, "WH-"+uuid.New().String()[:8], "Test address", lat, lng).Scan(&warehouseID)

	require.NoError(t, err)
	return warehouseID
}

// CreateTestInventory creates test inventory
func createTestInventory(t *testing.T, warehouseID, bookID string, quantity int) {
	t.Helper()
	ctx := context.Background()

	_, err := testContainer.DB.Pool.Exec(ctx, `
		INSERT INTO inventories (warehouse_id, book_id, quantity, reserved, min_stock_level)
		VALUES ($1, $2, $3, 0, 10)
	`, warehouseID, bookID, quantity)

	require.NoError(t, err)
}

// CreateTestPromotion creates a test promotion
func createTestPromotion(t *testing.T, code string, discountPercent float64, minAmount float64) string {
	t.Helper()
	ctx := context.Background()

	var promoID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO promotions (
			code, description, discount_type, discount_value,
			min_order_amount, max_discount_amount, usage_limit,
			start_date, end_date, is_active
		)
		VALUES ($1, $2, 'percentage', $3, $4, 100000, 100, NOW(), NOW() + INTERVAL '30 days', true)
		RETURNING id
	`, code, "Test promotion", discountPercent, minAmount).Scan(&promoID)

	require.NoError(t, err)
	return promoID
}

// CreateTestAddress creates a test address for a user
func createTestAddress(t *testing.T, userID string, isDefault bool) string {
	t.Helper()
	ctx := context.Background()

	var addressID string
	err := testContainer.DB.Pool.QueryRow(ctx, `
		INSERT INTO addresses (
			user_id, recipient_name, phone, address_line1,
			ward, district, city, is_default
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, userID, "Test Recipient", "+84912345678", "123 Test St",
		"Test Ward", "Test District", "Test City", isDefault).Scan(&addressID)

	require.NoError(t, err)
	return addressID
}

// ========================================
// CONCURRENCY TEST HELPERS
// ========================================

// RunConcurrent runs multiple functions concurrently and waits for all to complete
func runConcurrent(t *testing.T, count int, fn func(i int)) {
	t.Helper()
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(index int) {
			defer wg.Done()
			fn(index)
		}(i)
	}

	wg.Wait()
}

// RunConcurrentWithResults runs functions concurrently and collects results
func runConcurrentWithResults(t *testing.T, count int, fn func(i int) error) []error {
	t.Helper()
	results := make([]error, count)
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(index int) {
			defer wg.Done()
			results[index] = fn(index)
		}(i)
	}

	wg.Wait()
	return results
}

// WaitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}
