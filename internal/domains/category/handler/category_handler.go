package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bookstore-backend/internal/domains/category"
	"bookstore-backend/internal/shared/response"
	"bookstore-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================
// HANDLER STRUCT
// ============================================================
type CategoryHandler struct {
	service category.CategoryService
}

func NewCategoryHandler(svc category.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		service: svc,
	}
}

// ========== CREATE: POST /v1/categories  ==========
func (h *CategoryHandler) Create(c *gin.Context) {
	// ========== Parse Request ==========
	var req category.CreateCategoryReq

	// BindJSON với binding tags automatic validation
	// c.BindJSON() checks binding:"required" tags
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	resp, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		// Map error to HTTP status
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Create a category successfully", resp)
}

// ==========  GetByID - GET /v1/categories/:id ==========
func (h *CategoryHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {

		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	// 200 OK: Request successful
	response.Success(c, http.StatusOK, "Get category successfully", resp)
}

// ========== READ: GetBySlug ==========
// GET /v1/categories/:slug
// Params: slug (URL-friendly string)
//
// FLOW:
// 1. Extract slug from path
// 2. Call service.GetBySlug()
// 3. Return response
//
// NOTE: Ambiguity between ID and slug endpoints?
// Solution: Use different path
// - GET /v1/categories/:id (assume UUID)
// - GET /v1/categories/by-slug/:slug (explicit)
//
// OR: Try parse UUID, if fail try slug
func (h *CategoryHandler) GetBySlug(c *gin.Context) {
	// ========== Extract Slug ==========
	slug := c.Param("slug")
	slug = strings.TrimSpace(slug)

	if slug == "" {
		response.Error(c, http.StatusBadRequest, "Bad Request", errors.New("invalid slug"))
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Get category success", resp)
}

// ========== READ: GetAll ==========
// GET /v1/categories?is_active=true&parent_id=...&limit=10&offset=0
// Query params: is_active, parent_id, limit, offset
//
// FLOW:
// 1. Parse query parameters
// 2. Build filter options
// 3. Call service.GetAll()
// 4. Return paginated response
func (h *CategoryHandler) GetAll(c *gin.Context) {
	// ========== Parse Query Parameters ==========
	// Query: is_active=true => string "true"
	// Need to parse to bool
	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		val := isActiveStr == "true"
		isActive = &val
	}

	// parentID: UUID or nil
	var parentID *uuid.UUID
	if parentIDStr := c.Query("parent_id"); parentIDStr != "" {
		id, err := uuid.Parse(parentIDStr)
		if err != nil {
			logger.Error("GET ALL ERROR", err)
			response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		parentID = &id
	}

	// Pagination: limit, offset
	limit := 10 // Default
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // Default
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// ========== Call Service ==========
	resp, err := h.service.GetAll(c.Request.Context(), isActive, parentID, limit, offset)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== READ: GetTree ==========
// GET /v1/categories/tree
// No parameters: Return full tree
//
// FLOW:
// 1. Call service.GetTree()
// 2. Return ordered tree items
func (h *CategoryHandler) GetTree(c *gin.Context) {
	// ========== Call Service ==========
	resp, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== READ: GetBreadcrumb ==========
// GET /v1/categories/:id/breadcrumb
// Params: id (UUID)
//
// FLOW:
// 1. Extract ID from path
// 2. Call service.GetBreadcrumb()
// 3. Return breadcrumb items + full path
func (h *CategoryHandler) GetBreadcrumb(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.GetBreadcrumb(c.Request.Context(), id)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== UPDATE: Update ==========
// PUT /v1/categories/:id
//
//	Body: {
//	  "name": "New Name",
//	  "description": "...",
//	  "icon_url": "...",
//	  "sort_order": 1
//	}
//
// PARTIAL UPDATE:
// Only provided fields are updated
func (h *CategoryHandler) Update(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	// ========== Parse Request ==========
	var req category.UpdateCategoryReq
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	// 200 OK: Updated successfully
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== UPDATE: MoveToParent ==========
// PATCH /v1/categories/:id/parent
//
//	Body: {
//	  "parent_id": "new-parent-uuid"
//	}
//
// SEPARATE ENDPOINT:
// Tách riêng để:
// - Prevent accidental parent change
// - Clear intention
// - Separate validation (circular reference, depth)
func (h *CategoryHandler) MoveToParent(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Parse Request ==========
	var req category.MoveToParentReq
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.MoveToParent(c.Request.Context(), id, &req)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== UPDATE: Activate ==========
// POST /v1/categories/:id/activate
// No body
//
// FLOW:
// 1. Extract ID
// 2. Call service.Activate()
// 3. Return updated category
func (h *CategoryHandler) Activate(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.Activate(c.Request.Context(), id)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== UPDATE: Deactivate ==========
// POST /v1/categories/:id/deactivate
func (h *CategoryHandler) Deactivate(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.Deactivate(c.Request.Context(), id)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== DELETE: Delete ==========
// DELETE /v1/categories/:id
//
// FLOW:
// 1. Extract ID
// 2. Call service.Delete()
// 3. Return 204 No Content (or 200 OK with message)
func (h *CategoryHandler) Delete(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Call Service ==========
	err = h.service.Delete(c.Request.Context(), id)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Category deleted successfully", nil)
}

// ========== BULK: BulkActivate ==========
// POST /v1/categories/bulk/activate
//
//	Body: {
//	  "category_ids": ["id1", "id2", "id3"]
//	}
func (h *CategoryHandler) BulkActivate(c *gin.Context) {
	// ========== Parse Request ==========
	var req category.BulkCategoryIDsReq
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", errors.New("invalid request"))
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.BulkActivate(c.Request.Context(), &req)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== BULK: BulkDeactivate ==========
// POST /v1/categories/bulk/deactivate
func (h *CategoryHandler) BulkDeactivate(c *gin.Context) {
	// ========== Parse Request ==========
	var req category.BulkCategoryIDsReq
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", errors.New("invalid request"))
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.BulkDeactivate(c.Request.Context(), &req)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== BULK: BulkDelete ==========
// DELETE /v1/categories/bulk
//
//	Body: {
//	  "category_ids": ["id1", "id2", "id3"]
//	}
func (h *CategoryHandler) BulkDelete(c *gin.Context) {
	// ========== Parse Request ==========
	var req category.BulkCategoryIDsReq
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", errors.New("invalid request"))
		return
	}

	// ========== Call Service ==========
	resp, err := h.service.BulkDelete(c.Request.Context(), &req)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", resp)
}

// ========== BOOKS: GetBooksInCategory ==========
// GET /v1/categories/:id/books?limit=10&offset=0
//
// FLOW:
// 1. Extract ID
// 2. Parse pagination
// 3. Call service.GetBooksInCategory()
// 4. Return book IDs list
func (h *CategoryHandler) GetBooksInCategory(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Parse Pagination ==========
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
			page = p
		}
	}

	// ========== Call Service ==========
	books, total, err := h.service.GetBooksInCategory(c.Request.Context(), id, limit, page)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Build Response ==========
	res := map[string]interface{}{
		"books":    books,
		"total":    total,
		"limit":    limit,
		"page":     page,
		"has_more": (page + limit) < int(total),
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", res)
}

// ========== BOOKS: GetCategoryBookCount ==========
// GET /v1/categories/:id/book-count
func (h *CategoryHandler) GetCategoryBookCount(c *gin.Context) {
	// ========== Extract ID ==========
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", category.ErrInvalidCateID)
		return
	}

	// ========== Call Service ==========
	count, err := h.service.GetCategoryBookCount(c.Request.Context(), id)
	if err != nil {
		statusCode := category.GetHTTPStatusCode(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	// ========== Build Response ==========
	res := map[string]interface{}{
		"category_id": id.String(),
		"book_count":  count,
	}

	// ========== Success Response ==========
	response.Success(c, http.StatusOK, "Success", res)
}
