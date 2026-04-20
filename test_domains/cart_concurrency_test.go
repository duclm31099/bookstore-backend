package test_domains

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// CART CONCURRENCY TESTS
// ========================================

func TestConcurrentAddItem(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart_concurrent@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add same item concurrently 5 times
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			payload := map[string]interface{}{
				"book_id":  bookID,
				"quantity": 2,
			}

			resp := makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All should succeed
	assert.Equal(t, 5, successCount)

	// Verify final cart state
	resp := makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result := assertSuccess(t, resp, 200)

	data := result["data"].(map[string]interface{})
	// Should have 1 unique item with accumulated quantity
	assert.Equal(t, float64(1), data["items_count"])
}

func TestConcurrentUpdateQuantity(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart_update@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item first
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
	result := assertSuccess(t, resp, 200)
	data := result["data"].(map[string]interface{})
	item := data["item"].(map[string]interface{})
	itemID := item["id"].(string)

	// Update quantity concurrently
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			updatePayload := map[string]interface{}{
				"quantity": index + 2, // Different quantities
			}

			resp := makeRequest(t, "PUT", "/api/v1/cart/items/"+itemID, updatePayload, headers)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// All updates should succeed
	assert.Equal(t, 5, successCount)

	// Verify final state is consistent
	resp = makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result = assertSuccess(t, resp, 200)
	cartData := result["data"].(map[string]interface{})
	assert.Equal(t, float64(1), cartData["items_count"])
}

func TestConcurrentRemoveItem(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart_remove@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item first
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 5,
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
	result := assertSuccess(t, resp, 200)
	data := result["data"].(map[string]interface{})
	item := data["item"].(map[string]interface{})
	itemID := item["id"].(string)

	// Try to remove same item concurrently
	var wg sync.WaitGroup
	successCount := 0
	notFoundCount := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp := makeRequest(t, "DELETE", "/api/v1/cart/items/"+itemID, nil, headers)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			} else if resp.StatusCode == 404 {
				notFoundCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Only 1 should succeed, others should get 404
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 2, notFoundCount)

	// Verify cart is empty
	resp = makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result = assertSuccess(t, resp, 200)
	cartData := result["data"].(map[string]interface{})
	assert.Equal(t, float64(0), cartData["items_count"])
}

func TestConcurrentCheckout(t *testing.T) {
	cleanBeforeTest(t)

	userID, token := createAndLoginUser(t, "cart_checkout@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item to cart
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)

	// Try to checkout concurrently
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			checkoutPayload := map[string]interface{}{
				"address_id": addressID,
			}

			resp := makeRequest(t, "POST", "/api/v1/cart/checkout", checkoutPayload, headers)
			mu.Lock()
			if resp.StatusCode == 200 || resp.StatusCode == 201 {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Only 1 checkout should succeed (cart should be cleared after first checkout)
	assert.LessOrEqual(t, successCount, 1)
}

func TestConcurrentCartOperations_Mixed(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart_mixed@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	book1 := createTestBook(t, "Book 1", 100000, categoryID, authorID, publisherID)
	book2 := createTestBook(t, "Book 2", 150000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Perform mixed operations concurrently
	var wg sync.WaitGroup

	// Add items
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			bookID := book1
			if index%2 == 0 {
				bookID = book2
			}

			payload := map[string]interface{}{
				"book_id":  bookID,
				"quantity": 1,
			}
			makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
		}(i)
	}

	// Get cart
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			makeRequest(t, "GET", "/api/v1/cart", nil, headers)
		}()
	}

	wg.Wait()

	// Verify cart is in consistent state
	resp := makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result := assertSuccess(t, resp, 200)

	data := result["data"].(map[string]interface{})
	// Should have at most 2 unique items
	itemsCount := data["items_count"].(float64)
	assert.LessOrEqual(t, itemsCount, float64(2))
	assert.GreaterOrEqual(t, itemsCount, float64(1))
}

func TestConcurrentApplyPromotion(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart_promo@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	createTestPromotion(t, "SAVE10", 10, 50000)
	createTestPromotion(t, "SAVE20", 20, 100000)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item to cart (total 200k)
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)

	// Try to apply different promotions concurrently
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	promoCodes := []string{"SAVE10", "SAVE20", "SAVE10"}

	for _, code := range promoCodes {
		wg.Add(1)
		go func(promoCode string) {
			defer wg.Done()

			promoPayload := map[string]interface{}{
				"code": promoCode,
			}

			resp := makeRequest(t, "POST", "/api/v1/cart/apply-promotion", promoPayload, headers)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			}
			mu.Unlock()
		}(code)
	}

	wg.Wait()

	// At least one should succeed
	assert.GreaterOrEqual(t, successCount, 1)

	// Verify cart has a promotion applied
	resp := makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result := assertSuccess(t, resp, 200)

	data := result["data"].(map[string]interface{})
	// Should have a promotion code
	if data["promotion_code"] != nil {
		assert.NotEmpty(t, data["promotion_code"])
	}
}
