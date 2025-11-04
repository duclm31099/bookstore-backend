package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/author"
	"bookstore-backend/internal/shared/response"
)

type AuthorHandler struct {
	service author.Service
}

func NewAuthorHandler(svc author.Service) *AuthorHandler {
	return &AuthorHandler{
		service: svc,
	}
}

// ════════════════════════════════════════════════════════════════
// CREATE: POST /v1/authors
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) Create(c *gin.Context) {
	var req author.CreateAuthorRequest

	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	resp, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		statusCode := author.ToHTTPStatus(err)
		response.Error(c, statusCode, "Bad Request", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Create author successfully", resp.ToResponse())
}

// ════════════════════════════════════════════════════════════════
// READ: GetByID - GET /v1/authors/:id
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", "Invalid UUID format")
		return
	}

	resp, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == author.ErrAuthorNotFound {
			response.Error(c, http.StatusNotFound, "Not Found", err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Get author successfully", resp.ToResponse())
}

// ════════════════════════════════════════════════════════════════
// READ: GetBySlug - GET /v1/authors/slug/:slug
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	resp, err := h.service.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		if err == author.ErrAuthorNotFound {
			response.Error(c, http.StatusNotFound, "Not Found", err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Get author successfully", resp.ToResponse())
}

// ════════════════════════════════════════════════════════════════
// READ: GetAll - GET /v1/authors?limit=20&offset=0&sort_by=created_at&order=desc&search=
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) GetAll(c *gin.Context) {
	// Parse pagination
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build filter
	filter := author.AuthorFilter{
		Limit:  limit,
		Offset: offset,
		SortBy: c.DefaultQuery("sort_by", "created_at"),
		Order:  c.DefaultQuery("order", "desc"),
		Search: c.Query("search"),
	}

	// Call service
	authors, total, err := h.service.GetAll(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	// Build pagination response
	totalPages := (int(total) + filter.Limit - 1) / filter.Limit
	currentPage := (filter.Offset / filter.Limit) + 1

	authorResponses := make([]author.AuthorResponse, len(authors))
	for i, a := range authors {
		authorResponses[i] = *(a.ToResponse()) // Use the ToResponse() method
	}
	res := &author.AuthorListResponse{
		Data: authorResponses,
		Pagination: author.PaginationMeta{
			CurrentPage: currentPage,
			PageSize:    filter.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}

	response.Success(c, http.StatusOK, "Success", res)
}

// ════════════════════════════════════════════════════════════════
// READ: Search - GET /v1/authors/search?q=keyword&limit=20&offset=0
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "Bad Request", "Missing search query")
		return
	}

	// Parse pagination
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filter := author.AuthorFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Call service
	results, total, err := h.service.Search(c.Request.Context(), query, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	var authors []author.AuthorResponse
	for _, a := range results {
		temp := a.ToResponse()
		authors = append(authors, *temp)
	}
	res := author.SearchAuthorResponse{
		Authors: authors,
		Total:   int32(total),
	}

	response.Success(c, http.StatusOK, "Success", res)
}

// ════════════════════════════════════════════════════════════════
// UPDATE: PUT /v1/authors/:id
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) Update(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", "Invalid UUID format")
		return
	}

	var req author.UpdateAuthorRequest
	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	resp, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		if err == author.ErrAuthorNotFound {
			response.Error(c, http.StatusNotFound, "Not Found", err.Error())
		} else if err == author.ErrVersionMismatch {
			response.Error(c, http.StatusConflict, "Conflict", err.Error())
		} else {
			statusCode := author.ToHTTPStatus(err)
			response.Error(c, statusCode, "Bad Request", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Update author successfully", resp.ToResponse())
}

// ════════════════════════════════════════════════════════════════
// DELETE: DELETE /v1/authors/:id
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", "Invalid UUID format")
		return
	}

	err = h.service.Delete(c.Request.Context(), id)
	if err != nil {
		if err == author.ErrAuthorNotFound {
			response.Error(c, http.StatusNotFound, "Not Found", err.Error())
		} else if err == author.ErrAuthorHasBooks {
			response.Error(c, http.StatusConflict, "Conflict", err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Delete author successfully", nil)
}

// ════════════════════════════════════════════════════════════════
// BULK: BulkDelete - DELETE /v1/authors/bulk
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) BulkDelete(c *gin.Context) {
	var req author.BulkDeleteRequest

	if err := c.BindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	if len(req.IDs) == 0 {
		response.Error(c, http.StatusBadRequest, "Bad Request", "IDs cannot be empty")
		return
	}

	successCount, bulkErrors, err := h.service.BulkDelete(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	res := author.BulkDeleteResponse{
		SuccessCount: successCount,
		FailedCount:  len(bulkErrors),
		Errors:       bulkErrors,
	}

	response.Success(c, http.StatusOK, "Bulk delete completed", res)
}

// ════════════════════════════════════════════════════════════════
// BOOKS: GetWithBookCount - GET /v1/authors/:id/books
// ════════════════════════════════════════════════════════════════

func (h *AuthorHandler) GetWithBookCount(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Bad Request", "Invalid UUID format")
		return
	}

	a, bookCount, err := h.service.GetWithBookCount(c.Request.Context(), id)
	if err != nil {
		if err == author.ErrAuthorNotFound {
			response.Error(c, http.StatusNotFound, "Not Found", err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}
	response.Success(c, http.StatusOK, "Success", a.ToDetailResponse(bookCount))
}

// ════════════════════════════════════════════════════════════════
// ROUTES REGISTRATION
// ════════════════════════════════════════════════════════════════
