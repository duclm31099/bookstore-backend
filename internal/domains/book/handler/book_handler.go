package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"bookstore-backend/internal/domains/book/model"
	service "bookstore-backend/internal/domains/book/service"
	"bookstore-backend/internal/shared/response"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/cache"

	"github.com/gin-gonic/gin"
)

// Handler - HTTP Handler (single file)
type Handler struct {
	service service.ServiceInterface
	cache   cache.Cache
}

// NewHandler - Constructor with DI
func NewHandler(service service.ServiceInterface, cache cache.Cache) *Handler {
	return &Handler{
		service: service,
		cache:   cache,
	}
}

// ListBooks - GET /v1/books
// Query params: search, category, price_min, price_max, language, sort, page, limit
func (h *Handler) ListBooks(c *gin.Context) {
	// Parse query parameters
	req := model.ListBooksRequest{
		Search:     c.Query("search"),
		CategoryID: c.Query("category"),
		Language:   c.Query("language"),
		Sort:       c.DefaultQuery("sort", "newest"),
		Page:       1,
		Limit:      20,
	}

	// Parse numeric parameters
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			req.Page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			req.Limit = l
		}
	}

	if priceMinStr := c.Query("price_min"); priceMinStr != "" {
		if pm, err := strconv.ParseFloat(priceMinStr, 64); err == nil {
			req.PriceMin = pm
		}
	}

	if priceMaxStr := c.Query("price_max"); priceMaxStr != "" {
		if pm, err := strconv.ParseFloat(priceMaxStr, 64); err == nil {
			req.PriceMax = pm
		}
	}

	// Validate and call service
	if err := model.ValidateListRequest(req); err != nil {
		log.Printf("Validation error: %v", err)
		response.Error(c, http.StatusInternalServerError, "Internal server error", err.Error())
		return
	}

	data, meta, err := h.service.ListBooks(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Internal server error", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Get book successfully", model.ListBooksAPIResponse{
		Books:      data,
		Pagination: *meta,
	})
}

// GetBookDetail - GET /v1/books/:id
func (h *Handler) GetBookDetail(c *gin.Context) {
	id := c.Param("id")

	// 1. Validate ID format (UUID)
	if !utils.IsValidUUID(id) {
		response.Error(c, http.StatusBadRequest, "Bad Request", errors.New("invalid book id"))
		return
	}

	// 2. Check cache first
	cacheKey := model.GenerateBookDetailCacheKey(id)
	var cachedDetail model.BookDetailResponse
	found, err := h.cache.Get(c.Request.Context(), cacheKey, &cachedDetail)

	// Cache hit - return immediately
	if found {
		response.Success(c, http.StatusAccepted, "Get book successfully", &cachedDetail)
		return
	}

	// Log nếu có error (nhưng vẫn tiếp tục query DB)
	if err != nil {
		log.Printf("[Handler] Cache error for key %s: %v", cacheKey, err)
	}

	// 3. Cache MISS - fetch from service
	detail, err := h.service.GetBookDetail(c.Request.Context(), id)

	isInvalid := model.HandleBookError(c, err)
	if isInvalid {
		return
	}

	// 4. Cache the result (TTL 10 minutes = 600 seconds)
	// Cache.Set tự động marshal sang JSON
	if err := h.cache.Set(c.Request.Context(), cacheKey, detail, 10*time.Minute); err != nil {
		log.Printf("[Handler] Failed to cache book detail: %v", err)
	}

	response.Success(c, http.StatusOK, "Get book successfully", detail)
}

// CreateBook - POST /v1/admin/books
// Yêu cầu: Admin role (middleware check trước khi vào handler)
func (h *Handler) CreateBook(c *gin.Context) {
	var req model.CreateBookRequest

	// 1. Bind và validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Handler] Invalid create book request: %v", err)
		response.Error(c, http.StatusBadRequest, "Invalid request data", err.Error())
		return
	}

	// 2. Business validation
	if err := model.ValidateCreateRequest(&req); err != nil {
		log.Printf("[Handler] Validation failed: %v", err)
		response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// 3. Call service to create book
	err := h.service.CreateBook(c.Request.Context(), req)

	// Handle specific business errors
	isInvalid := model.HandleBookError(c, err)
	if isInvalid {
		return
	}

	// 4. Return success
	response.Success(c, http.StatusCreated, "Book created successfully", nil)
}

func (h *Handler) UpdateBook(c *gin.Context) {
	id := c.Param("id")

	// 1. Validate ID
	if !utils.IsValidUUID(id) {
		response.Error(c, http.StatusBadRequest, "Invalid book ID", "ID must be a valid UUID")
		return
	}

	// 2. Bind request
	var req model.UpdateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Handler] Invalid update book request: %v", err)
		response.Error(c, http.StatusBadRequest, "Invalid request data", err.Error())
		return
	}

	// 3. Business validation
	if err := model.ValidateUpdateRequest(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// 4. Call service
	detail, err := h.service.UpdateBook(c.Request.Context(), id, req)

	invalid := model.HandleBookError(c, err)
	if invalid == true {
		return
	}
	// 5. Return success
	response.Success(c, http.StatusOK, "Book updated successfully", detail)
}

// ============ STUB HANDLERS (implement in next APIs) ============

func (h *Handler) DeleteBook(c *gin.Context) {
	bookId, found := c.Params.Get("id")
	if found == false || !utils.IsValidUUID(bookId) {
		response.Error(c, http.StatusBadRequest, "Bad request", errors.New("Invalid book id"))
		return
	}
	_, exist := c.Get("user_id")
	if !exist {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", errors.New("User not authenticated"))
		return
	}
	deleteResponse, err := h.service.DeleteBook(c.Request.Context(), bookId)
	isInvalid := model.HandleBookError(c, err)
	if isInvalid == true {
		return
	}
	response.Success(c, http.StatusOK, "Book deleted successfully", deleteResponse)
}

// ================ SEARCH BOOK =========================
// SearchBooks - GET /v1/books/search?q=keyword&limit=10
// Full-text search using PostgreSQL tsvector
func (h *Handler) SearchBooks(c *gin.Context) {
	startTime := time.Now()

	var req model.SearchBooksRequest

	// 1. Bind and validate query params
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("[Handler] Invalid search request: %v", err)
		response.Error(c, http.StatusBadRequest, "Invalid search parameters", err.Error())
		return
	}

	// 2. Set default limit
	if req.Limit == 0 {
		req.Limit = 10
	}

	// 3. Validate query length
	if len(req.Query) < 2 {
		response.Error(c, http.StatusBadRequest, "Query too short", "Search query must be at least 2 characters")
		return
	}

	// 4. Call service
	results, err := h.service.SearchBooks(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Handler] Error searching books: %v", err)
		response.Error(c, http.StatusInternalServerError, "Search failed", "Internal server error")
		return
	}

	// 5. Calculate query time
	tookMs := time.Since(startTime).Milliseconds()

	// 6. Return results
	meta := &model.SearchMeta{
		Query:       req.Query,
		ResultCount: len(results),
		TookMs:      tookMs,
	}

	// Log for analytics (phase sau sẽ save vào DB)
	log.Printf("[Search] Query: %q, Results: %d, Took: %dms", req.Query, len(results), tookMs)

	response.Success(c, http.StatusOK, "Search completed successfully", map[string]interface{}{
		"results": results,
		"meta":    meta,
	})
}
