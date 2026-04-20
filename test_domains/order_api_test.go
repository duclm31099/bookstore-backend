package test_domains

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// ORDER API TESTS
// ========================================

func TestCreateOrder_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup test data
	userID, token := createAndLoginUser(t, "order@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	// Add item to cart
	addPayload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 2,
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
	assertSuccess(t, resp, http.StatusOK)

	// Create order
	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	result := assertSuccess(t, resp, http.StatusCreated)

	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.NotEmpty(t, data["order_number"])
	assert.Equal(t, "pending", data["status"])
	assert.NotNil(t, data["total_amount"])
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	cleanBeforeTest(t)

	// Setup test data with low stock
	userID, token := createAndLoginUser(t, "order2@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 1) // Only 1 in stock
	addressID := createTestAddress(t, userID, true)

	// Add 5 items to cart (more than stock)
	addPayload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 5,
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
	assertSuccess(t, resp, http.StatusOK)

	// Try to create order - should fail
	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	assertError(t, resp, http.StatusUnprocessableEntity)
}

func TestCreateOrder_EmptyCart(t *testing.T) {
	cleanBeforeTest(t)

	userID, token := createAndLoginUser(t, "order3@test.com", "Test1234")
	addressID := createTestAddress(t, userID, true)

	// Try to create order with empty cart
	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	assertError(t, resp, http.StatusBadRequest)
}

func TestGetOrderDetail_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup and create order
	userID, token := createAndLoginUser(t, "order4@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	// Add to cart and create order
	addPayload := map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
	assertSuccess(t, resp, http.StatusOK)

	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	data := result["data"].(map[string]interface{})
	orderID := data["id"].(string)

	// Get order detail
	resp = makeRequest(t, "GET", "/api/v1/orders/"+orderID, nil, headers)
	result = assertSuccess(t, resp, http.StatusOK)

	orderData := result["data"].(map[string]interface{})
	assert.Equal(t, orderID, orderData["id"])
	assert.NotEmpty(t, orderData["order_number"])
	assert.NotEmpty(t, orderData["items"])
}

func TestGetOrderDetail_NotFound(t *testing.T) {
	cleanBeforeTest(t)

	_, token := createAndLoginUser(t, "order5@test.com", "Test1234")

	headers := map[string]string{"Authorization": "Bearer " + token}
	resp := makeRequest(t, "GET", "/api/v1/orders/00000000-0000-0000-0000-000000000000", nil, headers)
	assertError(t, resp, http.StatusNotFound)
}

func TestListOrders_WithPagination(t *testing.T) {
	cleanBeforeTest(t)

	userID, token := createAndLoginUser(t, "order6@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create 3 orders
	for i := 0; i < 3; i++ {
		addPayload := map[string]interface{}{
			"book_id":  bookID,
			"quantity": 1,
		}
		makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)

		orderPayload := map[string]interface{}{
			"address_id": addressID,
		}
		makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	}

	// List orders
	resp := makeRequest(t, "GET", "/api/v1/orders?page=1&limit=10", nil, headers)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	orders := data["orders"].([]interface{})
	assert.GreaterOrEqual(t, len(orders), 3)
}

func TestCancelOrder_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup and create order
	userID, token := createAndLoginUser(t, "order7@test.com", "Test1234")
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
	assertSuccess(t, resp, http.StatusOK)

	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	data := result["data"].(map[string]interface{})
	orderID := data["id"].(string)
	version := int(data["version"].(float64))

	// Cancel order
	cancelPayload := map[string]interface{}{
		"reason":  "Changed my mind",
		"version": version,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders/"+orderID+"/cancel", cancelPayload, headers)
	assertSuccess(t, resp, http.StatusOK)

	// Verify order is cancelled
	resp = makeRequest(t, "GET", "/api/v1/orders/"+orderID, nil, headers)
	result = assertSuccess(t, resp, http.StatusOK)
	orderData := result["data"].(map[string]interface{})
	assert.Equal(t, "cancelled", orderData["status"])
}

func TestCancelOrder_VersionMismatch(t *testing.T) {
	cleanBeforeTest(t)

	// Setup and create order
	userID, token := createAndLoginUser(t, "order8@test.com", "Test1234")
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
		"quantity": 1,
	}
	resp := makeRequest(t, "POST", "/api/v1/cart/items", addPayload, headers)
	assertSuccess(t, resp, http.StatusOK)

	orderPayload := map[string]interface{}{
		"address_id": addressID,
	}
	resp = makeRequest(t, "POST", "/api/v1/orders", orderPayload, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	data := result["data"].(map[string]interface{})
	orderID := data["id"].(string)

	// Try to cancel with wrong version
	cancelPayload := map[string]interface{}{
		"reason":  "Changed my mind",
		"version": 999, // Wrong version
	}
	resp = makeRequest(t, "POST", "/api/v1/orders/"+orderID+"/cancel", cancelPayload, headers)
	assertError(t, resp, http.StatusConflict)
}
