package test_domains

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// INVENTORY CONCURRENCY TESTS
// ========================================

func TestConcurrentReserveStock(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Try to reserve stock concurrently
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			payload := map[string]interface{}{
				"warehouse_id": warehouseID,
				"book_id":      bookID,
				"quantity":     10,
			}

			resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", payload, nil)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All 10 reservations should succeed (10 * 10 = 100)
	assert.Equal(t, 10, successCount)

	// Verify total reserved
	ctx := context.Background()
	var reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&reserved)

	require.NoError(t, err)
	assert.Equal(t, 100, reserved)
}

func TestConcurrentReserveStock_Overselling(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50) // Only 50 in stock

	// Try to reserve 10 items, 10 times concurrently (total 100 > 50)
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			payload := map[string]interface{}{
				"warehouse_id": warehouseID,
				"book_id":      bookID,
				"quantity":     10,
			}

			resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", payload, nil)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Only 5 should succeed (50 / 10 = 5)
	assert.Equal(t, 5, successCount)
	assert.Equal(t, 5, failCount)

	// Verify no overselling
	ctx := context.Background()
	var quantity, reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT quantity, reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&quantity, &reserved)

	require.NoError(t, err)
	assert.Equal(t, 50, quantity)
	assert.Equal(t, 50, reserved)
	assert.LessOrEqual(t, reserved, quantity) // Critical: reserved should never exceed quantity
}

func TestConcurrentUpdateInventory(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Try to update inventory concurrently with same version
	var wg sync.WaitGroup
	successCount := 0
	conflictCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			payload := map[string]interface{}{
				"quantity": 100 + (index * 10),
				"version":  1, // Same version for all
			}

			resp := makeRequest(t, "PATCH", "/api/v1/inventories/"+warehouseID+"/"+bookID, payload, nil)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			} else if resp.StatusCode == 409 {
				conflictCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Only 1 should succeed due to optimistic locking
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 4, conflictCount)
}

func TestConcurrentCompleteSale(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Reserve 50 items first
	reservePayload := map[string]interface{}{
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"quantity":     50,
	}
	resp := makeRequest(t, "POST", "/api/v1/inventories/reserve", reservePayload, nil)
	assertSuccess(t, resp, 200)

	// Complete sale concurrently (5 times, 10 items each)
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			salePayload := map[string]interface{}{
				"warehouse_id": warehouseID,
				"book_id":      bookID,
				"quantity":     10,
			}

			resp := makeRequest(t, "POST", "/api/v1/inventories/complete-sale", salePayload, nil)
			mu.Lock()
			if resp.StatusCode == 200 {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All 5 should succeed
	assert.Equal(t, 5, successCount)

	// Verify final state
	ctx := context.Background()
	var quantity, reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT quantity, reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&quantity, &reserved)

	require.NoError(t, err)
	assert.Equal(t, 50, quantity) // 100 - 50 sold
	assert.Equal(t, 0, reserved)  // 50 - 50 completed
}

func TestRaceCondition_ReserveAndRelease(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 100)

	// Reserve and release concurrently
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(2) // Reserve and release

		go func() {
			defer wg.Done()
			payload := map[string]interface{}{
				"warehouse_id": warehouseID,
				"book_id":      bookID,
				"quantity":     5,
			}
			makeRequest(t, "POST", "/api/v1/inventories/reserve", payload, nil)
		}()

		go func() {
			defer wg.Done()
			payload := map[string]interface{}{
				"warehouse_id": warehouseID,
				"book_id":      bookID,
				"quantity":     5,
			}
			makeRequest(t, "POST", "/api/v1/inventories/release", payload, nil)
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	ctx := context.Background()
	var quantity, reserved int
	err := testContainer.DB.Pool.QueryRow(ctx, `
		SELECT quantity, reserved FROM inventories 
		WHERE warehouse_id = $1 AND book_id = $2
	`, warehouseID, bookID).Scan(&quantity, &reserved)

	require.NoError(t, err)
	assert.Equal(t, 100, quantity)            // Quantity should not change
	assert.GreaterOrEqual(t, reserved, 0)     // Reserved should never be negative
	assert.LessOrEqual(t, reserved, quantity) // Reserved should never exceed quantity
}
