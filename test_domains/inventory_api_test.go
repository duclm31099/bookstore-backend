package test_domains

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// INVENTORY API TESTS
// ========================================

func TestCreateInventory_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup test data
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)

	// Create inventory
	payload := map[string]interface{}{
		"warehouse_id":    warehouseID,
		"book_id":         bookID,
		"quantity":        100,
		"min_stock_level": 10,
	}

	resp := makeRequest(t, "POST", "/api/v1/inventories", payload, nil)
	result := assertSuccess(t, resp, http.StatusCreated)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, warehouseID, data["warehouse_id"])
	assert.Equal(t, bookID, data["book_id"])
	assert.Equal(t, float64(100), data["quantity"])
	assert.Equal(t, float64(0), data["reserved"])
}

func TestCreateInventory_Duplicate(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)

	// Create inventory first time
	payload := map[string]interface{}{
		"warehouse_id":    warehouseID,
		"book_id":         bookID,
		"quantity":        100,
		"min_stock_level": 10,
	}
	resp := makeRequest(t, "POST", "/api/v1/inventories", payload, nil)
	assertSuccess(t, resp, http.StatusCreated)

	// Try to create again - should fail
	resp = makeRequest(t, "POST", "/api/v1/inventories", payload, nil)
	assertError(t, resp, http.StatusConflict)
}

func TestGetInventory_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)

	// Get inventory
	resp := makeRequest(t, "GET", "/api/v1/inventories/"+warehouseID+"/"+bookID, nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, warehouseID, data["warehouse_id"])
	assert.Equal(t, bookID, data["book_id"])
	assert.Equal(t, float64(50), data["quantity"])
}

func TestListInventories_WithFilters(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	book1 := createTestBook(t, "Book 1", 100000, categoryID, authorID, publisherID)
	book2 := createTestBook(t, "Book 2", 150000, categoryID, authorID, publisherID)
	warehouse1 := createTestWarehouse(t, "Warehouse 1", 10.762622, 106.660172)
	warehouse2 := createTestWarehouse(t, "Warehouse 2", 10.762622, 106.660172)

	createTestInventory(t, warehouse1, book1, 100)
	createTestInventory(t, warehouse1, book2, 50)
	createTestInventory(t, warehouse2, book1, 75)

	// List all inventories
	resp := makeRequest(t, "GET", "/api/v1/inventories", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	inventories := data["inventories"].([]interface{})
	assert.GreaterOrEqual(t, len(inventories), 3)
}

func TestUpdateInventory_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)

	// Update inventory
	payload := map[string]interface{}{
		"quantity": 100,
		"version":  1,
	}

	resp := makeRequest(t, "PATCH", "/api/v1/inventories/"+warehouseID+"/"+bookID, payload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, float64(100), data["quantity"])
	assert.Equal(t, float64(2), data["version"]) // Version incremented
}

func TestUpdateInventory_VersionConflict(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)

	// Try to update with wrong version
	payload := map[string]interface{}{
		"quantity": 100,
		"version":  999, // Wrong version
	}

	resp := makeRequest(t, "PATCH", "/api/v1/inventories/"+warehouseID+"/"+bookID, payload, nil)
	assertError(t, resp, http.StatusConflict)
}

func TestDeleteInventory_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 0) // Empty inventory

	// Delete inventory
	resp := makeRequest(t, "DELETE", "/api/v1/inventories/"+warehouseID+"/"+bookID, nil, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify deleted
	resp = makeRequest(t, "GET", "/api/v1/inventories/"+warehouseID+"/"+bookID, nil, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestDeleteInventory_NonEmpty(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50) // Non-empty

	// Try to delete - should fail
	resp := makeRequest(t, "DELETE", "/api/v1/inventories/"+warehouseID+"/"+bookID, nil, nil)
	assertError(t, resp, http.StatusConflict)
}

// ========================================
// STOCK OPERATIONS TESTS
// ========================================

func TestReserveStock_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Reserve stock
	payload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     10,
	}

	resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", payload, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify reserved
	ctx := context.Background()
	var reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&reserved)

	require.NoError(t, err)
	assert.Equal(t, 10, reserved)
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 5) // Only 5 in stock

	// Try to reserve 10 - should fail
	payload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     10,
	}

	resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", payload, nil)
	assertError(t, resp, http.StatusConflict)
}

func TestReleaseStock_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Reserve first
	reservePayload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     10,
	}
	resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", reservePayload, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Release
	releasePayload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     10,
	}
	resp = makeRequest(t, "POST", "/api/v1/inventories/release", releasePayload, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify released
	ctx := context.Background()
	var reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&reserved)

	require.NoError(t, err)
	assert.Equal(t, 0, reserved)
}

func TestCompleteSale_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Reserve first
	reservePayload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     10,
	}
	resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", reservePayload, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Complete sale
	salePayload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     10,
	}
	resp = makeRequest(t, "POST", "/api/v1/inventories/complete-sale", salePayload, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify both quantity and reserved decreased
	ctx := context.Background()
	var quantity, reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT quantity, reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&quantity, &reserved)

	require.NoError(t, err)
	assert.Equal(t, 90, quantity)
	assert.Equal(t, 0, reserved)
}

func TestFindOptimalWarehouse_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	// Create 2 warehouses at different locations
	warehouse1 := createTestWarehouse(t, "Warehouse 1", 10.762622, 106.660172) // HCM
	warehouse2 := createTestWarehouse(t, "Warehouse 2", 21.028511, 105.804817) // Hanoi

	createTestInventory(t, warehouse1, bookID, 50)
	createTestInventory(t, warehouse2, bookID, 50)

	// Find warehouse nearest to HCM
	payload := map[string]interface{}{
		"book_id":   bookID,
		"quantity":  10,
		"latitude":  10.762622,
		"longitude": 106.660172,
	}

	resp := makeRequest(t, "POST", "/api/v1/inventories/find-warehouse", payload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, warehouse1, data["warehouse_id"]) // Should return warehouse1 (closer)
}
