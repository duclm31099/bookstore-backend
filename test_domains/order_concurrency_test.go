package test_domains

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// ORDER CONCURRENCY TESTS
// ========================================

func TestConcurrentOrderCreation(t *testing.T) {
	cleanBeforeTest(t)

	// Setup test data
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Create 5 users concurrently creating orders
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			email := fmt.Sprintf("concurrent%d@test.com", index)
			userID, token := createAndLoginUser(t, email, "Test1234")
			addressID := createTestAddress(t, userID, true)

			// Add to cart
			addPayload := map[string]interface{}{
				"book_id":  bookID,
				"quantity": 2,
			}
			headers := map[string]string{"Authorization": "Bearer " + token}
			resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
			if resp.StatusCode == 200 {
				// Create order
				orderPayload := map[string]interface{}{
					"address_id": addressID,
				}
				resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
				if resp.StatusCode == 201 {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// All 5 orders should succeed (we have 100 items in stock)
	assert.Equal(t, 5, successCount)

	// Verify inventory was properly reserved
	ctx := context.Background()
	var reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&reserved)

	assert.NoError(t, err)
	assert.Equal(t, 10, reserved) // 5 orders * 2 items each
}

func TestConcurrentOrderCreation_SameBook_LimitedStock(t *testing.T) {
	cleanBeforeTest(t)

	// Setup test data with limited stock
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 10) // Only 10 items

	// Try to create 10 orders concurrently, each wanting 2 items
	// Only 5 should succeed
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			email := fmt.Sprintf("limited%d@test.com", index)
			userID, token := createAndLoginUser(t, email, "Test1234")
			addressID := createTestAddress(t, userID, true)

			// Add to cart
			addPayload := map[string]interface{}{
				"book_id":  bookID,
				"quantity": 2,
			}
			headers := map[string]string{"Authorization": "Bearer " + token}
			resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
			if resp.StatusCode == 200 {
				// Create order
				orderPayload := map[string]interface{}{
					"address_id": addressID,
				}
				resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
				mu.Lock()
				if resp.StatusCode == 201 {
					successCount++
				} else {
					failCount++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Only 5 orders should succeed (10 items / 2 per order)
	assert.Equal(t, 5, successCount)
	assert.Equal(t, 5, failCount)

	// Verify no overselling occurred
	ctx := context.Background()
	var quantity, reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT quantity, reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&quantity, &reserved)

	assert.NoError(t, err)
	assert.Equal(t, 10, quantity)
	assert.Equal(t, 10, reserved)
	assert.LessOrEqual(t, reserved, quantity) // Reserved should never exceed quantity
}

func TestConcurrentOrderCancellation(t *testing.T) {
	cleanBeforeTest(t)

	// Setup and create an order
	userID, token := createAndLoginUser(t, "cancel@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add to cart and create order
	addPayload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
	assertSuccess(t, resp, 200)

	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	result := assertSuccess(t, resp, 201)
	data := result["data"].(map[string]interface{})
	orderID := data["id"].(string)
	version := int(data["version"].(float64))

	// Try to cancel the same order concurrently 3 times
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cancelPayload := map[string]interface{}{
				"reason":  "Changed my mind",
				"version": version,
			}
			resp := makeRequest(t, "POST", "/api/v1/orders/"+orderID+"/cancel", cancelPayload, headers)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Only one cancellation should succeed due to optimistic locking
	assert.Equal(t, 1, successCount)
}
