package test_domains

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// CART API TESTS
// ========================================

func TestGetCart_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart@test.com", "Test1234")

	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["id"])
	assert.Equal(t, float64(0), data["items_count"])
}

func TestAddItem_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart2@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	// Add item to cart
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	cart := data["cart"].(map[string]interface{})
	assert.Equal(t, float64(1), cart["items_count"])
}

func TestAddItem_UpdateQuantity(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart3@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item first time
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)

	// Add same item again - should update quantity
	payload = map[string]interface{}{
		"book_id":  bookID,
		"quantity": 3,
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	cart := data["cart"].(map[string]interface{})
	assert.Equal(t, float64(1), cart["items_count"]) // Still 1 unique item
}

func TestRemoveItem_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart4@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item
	payload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", payload, headers)
	result := assertSuccess(t, resp, http.StatusOK)
	data := result["data"].(map[string]interface{})
	item := data["item"].(map[string]interface{})
	itemID := item["id"].(string)

	// Remove item
	resp = makeRequest(t, "DELETE", "/api/v1/cart/items/"+itemID, nil, headers)
	assertSuccess(t, resp, http.StatusOK)

	// Verify cart is empty
	resp = makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result = assertSuccess(t, resp, http.StatusOK)
	cartData := result["data"].(map[string]interface{})
	assert.Equal(t, float64(0), cartData["items_count"])
}

func TestClearCart_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart5@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	book1 := createTestBook(t, "Book 1", 100000, categoryID, authorID, publisherID)
	book2 := createTestBook(t, "Book 2", 150000, categoryID, authorID, publisherID)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add 2 items
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  book1,
		"quantity": 1,
	}, headers)
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  book2,
		"quantity": 1,
	}, headers)

	// Clear cart
	resp := makeRequest(t, "DELETE", "/api/v1/cart", nil, headers)
	assertSuccess(t, resp, http.StatusOK)

	// Verify cart is empty
	resp = makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result := assertSuccess(t, resp, http.StatusOK)
	cartData := result["data"].(map[string]interface{})
	assert.Equal(t, float64(0), cartData["items_count"])
}

func TestValidateCart_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart6@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}, headers)

	// Validate cart
	resp := makeRequest(t, "POST", "/api/v1/cart/validate", nil, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.True(t, data["is_valid"].(bool))
}

func TestValidateCart_OutOfStock(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart7@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 1) // Only 1 in stock

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add 5 items (more than stock)
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 5,
	}, headers)

	// Validate cart - should be invalid
	resp := makeRequest(t, "POST", "/api/v1/cart/validate", nil, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.False(t, data["is_valid"].(bool))
}

func TestApplyPromoCode_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart8@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	createTestPromotion(t, "SAVE10", 10, 50000) // 10% off, min 50k

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item (total 200k)
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}, headers)

	// Apply promo code
	payload := map[string]interface{}{
		"code": "SAVE10",
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/apply-promotion", payload, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	cart := data["cart"].(map[string]interface{})
	assert.NotNil(t, cart["promotion_code"])
	assert.Greater(t, cart["discount_amount"], float64(0))
}

func TestApplyPromoCode_MinimumNotMet(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart9@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 30000, categoryID, authorID, publisherID)
	createTestPromotion(t, "SAVE10", 10, 100000) // 10% off, min 100k

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item (total only 30k, less than minimum)
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}, headers)

	// Try to apply promo code - should fail
	payload := map[string]interface{}{
		"code": "SAVE10",
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/apply-promotion", payload, headers)
	assertError(t, resp, http.StatusUnprocessableEntity)
}

func TestRemovePromoCode_Success(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "cart10@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	createTestPromotion(t, "SAVE10", 10, 50000)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Add item and apply promo
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}, headers)
	makeRequest(t, "POST", "/api/v1/cart/apply-promotion", map[string]interface{}{
		"code": "SAVE10",
	}, headers)

	// Remove promo code
	resp := makeRequest(t, "DELETE", "/api/v1/cart/remove-promotion", nil, headers)
	assertSuccess(t, resp, http.StatusOK)

	// Verify promo removed
	resp = makeRequest(t, "GET", "/api/v1/cart", nil, headers)
	result := assertSuccess(t, resp, http.StatusOK)
	cartData := result["data"].(map[string]interface{})
	assert.Nil(t, cartData["promotion_code"])
	assert.Equal(t, float64(0), cartData["discount_amount"])
}
