package test_domains

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// BOOK API TESTS
// ========================================

func TestListBooks_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup test data
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	createTestBook(t, "Book 1", 100000, categoryID, authorID, publisherID)
	createTestBook(t, "Book 2", 150000, categoryID, authorID, publisherID)
	createTestBook(t, "Book 3", 200000, categoryID, authorID, publisherID)

	// List books
	resp := makeRequest(t, "GET", "/api/v1/books", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.GreaterOrEqual(t, len(books), 3)
}

func TestListBooks_WithPagination(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	// Create 5 books
	for i := 1; i <= 5; i++ {
		createTestBook(t, "Book "+string(rune(i+'0')), 100000, categoryID, authorID, publisherID)
	}

	// Get first page (limit 2)
	resp := makeRequest(t, "GET", "/api/v1/books?page=1&limit=2", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.LessOrEqual(t, len(books), 2)
}

func TestListBooks_WithFilters(t *testing.T) {
	cleanBeforeTest(t)

	category1 := createTestCategory(t, "Fiction")
	category2 := createTestCategory(t, "Science")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	createTestBook(t, "Fiction Book", 100000, category1, authorID, publisherID)
	createTestBook(t, "Science Book", 150000, category2, authorID, publisherID)

	// Filter by category
	resp := makeRequest(t, "GET", "/api/v1/books?category="+category1, nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.GreaterOrEqual(t, len(books), 1)
}

func TestListBooks_WithPriceRange(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	createTestBook(t, "Cheap Book", 50000, categoryID, authorID, publisherID)
	createTestBook(t, "Mid Book", 150000, categoryID, authorID, publisherID)
	createTestBook(t, "Expensive Book", 300000, categoryID, authorID, publisherID)

	// Filter by price range (100k - 200k)
	resp := makeRequest(t, "GET", "/api/v1/books?price_min=100000&price_max=200000", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.GreaterOrEqual(t, len(books), 1)
}

func TestGetBookDetail_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	// Get book detail
	resp := makeRequest(t, "GET", "/api/v1/books/"+bookID, nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, bookID, data["id"])
	assert.NotEmpty(t, data["title"])
	assert.NotEmpty(t, data["price"])
}

func TestGetBookDetail_NotFound(t *testing.T) {
	cleanBeforeTest(t)

	resp := makeRequest(t, "GET", "/api/v1/books/00000000-0000-0000-0000-000000000000", nil, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestCreateBook_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	payload := map[string]interface{}{
		"title":            "New Book",
		"isbn":             "ISBN-123456",
		"description":      "Test description",
		"price":            120000,
		"stock_quantity":   100,
		"category_id":      categoryID,
		"author_id":        authorID,
		"publisher_id":     publisherID,
		"language":         "vi",
		"page_count":       250,
		"publication_year": 2024,
	}

	resp := makeRequest(t, "POST", "/api/v1/books", payload, nil)
	result := assertSuccess(t, resp, http.StatusCreated)

	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "New Book", data["title"])
	assert.Equal(t, float64(120000), data["price"])
}

func TestCreateBook_DuplicateISBN(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	payload := map[string]interface{}{
		"title":            "Book 1",
		"isbn":             "ISBN-DUPLICATE",
		"description":      "Test",
		"price":            100000,
		"stock_quantity":   50,
		"category_id":      categoryID,
		"author_id":        authorID,
		"publisher_id":     publisherID,
		"language":         "vi",
		"page_count":       200,
		"publication_year": 2024,
	}

	// Create first book
	makeRequest(t, "POST", "/api/v1/books", payload, nil)

	// Try to create with same ISBN
	payload["title"] = "Book 2"
	resp := makeRequest(t, "POST", "/api/v1/books", payload, nil)
	assertError(t, resp, http.StatusConflict)
}

func TestUpdateBook_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Original Title", 100000, categoryID, authorID, publisherID)

	// Update book
	payload := map[string]interface{}{
		"title":       "Updated Title",
		"price":       150000,
		"description": "Updated description",
	}

	resp := makeRequest(t, "PUT", "/api/v1/books/"+bookID, payload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Updated Title", data["title"])
	assert.Equal(t, float64(150000), data["price"])
}

func TestDeleteBook_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "To Delete", 100000, categoryID, authorID, publisherID)

	// Delete book
	resp := makeRequest(t, "DELETE", "/api/v1/books/"+bookID, nil, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify deleted
	resp = makeRequest(t, "GET", "/api/v1/books/"+bookID, nil, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestSearchBooks_Success(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	createTestBook(t, "Harry Potter", 200000, categoryID, authorID, publisherID)
	createTestBook(t, "Lord of the Rings", 250000, categoryID, authorID, publisherID)
	createTestBook(t, "The Hobbit", 180000, categoryID, authorID, publisherID)

	// Search for "Harry"
	resp := makeRequest(t, "GET", "/api/v1/books/search?q=Harry", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.GreaterOrEqual(t, len(books), 1)
}

func TestSearchBooks_NoResults(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)

	// Search for non-existent book
	resp := makeRequest(t, "GET", "/api/v1/books/search?q=NonExistentBook123", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.Equal(t, 0, len(books))
}

func TestSearchBooks_WithLimit(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	// Create multiple books with "Test" in title
	for i := 1; i <= 5; i++ {
		createTestBook(t, "Test Book "+string(rune(i+'0')), 100000, categoryID, authorID, publisherID)
	}

	// Search with limit
	resp := makeRequest(t, "GET", "/api/v1/books/search?q=Test&limit=3", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.LessOrEqual(t, len(books), 3)
}

func TestListBooks_WithSorting(t *testing.T) {
	cleanBeforeTest(t)

	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")

	createTestBook(t, "Book A", 300000, categoryID, authorID, publisherID)
	createTestBook(t, "Book B", 100000, categoryID, authorID, publisherID)
	createTestBook(t, "Book C", 200000, categoryID, authorID, publisherID)

	// Sort by price ascending
	resp := makeRequest(t, "GET", "/api/v1/books?sort=price_asc", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	books := data["books"].([]interface{})
	assert.GreaterOrEqual(t, len(books), 3)

	// Verify first book has lowest price
	if len(books) >= 2 {
		book1 := books[0].(map[string]interface{})
		book2 := books[1].(map[string]interface{})
		assert.LessOrEqual(t, book1["price"].(float64), book2["price"].(float64))
	}
}
