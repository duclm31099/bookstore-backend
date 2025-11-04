package handler

import (
	"bookstore-backend/internal/domains/publisher"
	"bookstore-backend/internal/shared/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PublisherHandler handles HTTP requests for publisher domain
type PublisherHandler struct {
	service publisher.Service
}

// NewPublisherHandler creates a new publisher handler instance
// Dependency injection pattern - receives service from container
func NewPublisherHandler(service publisher.Service) *PublisherHandler {
	return &PublisherHandler{
		service: service,
	}
}

// CreatePublisher handles POST /publishers
func (h *PublisherHandler) CreatePublisher(c *gin.Context) {
	var req publisher.PublisherCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.CreatePublisher(c.Request.Context(), &req)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusCreated, "Publisher created successfully", result)
}

// GetPublisher handles GET /publishers/:id
func (h *PublisherHandler) GetPublisher(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(publisher.NewInvalidPublisherID(idStr))
		response.Error(c, statusCode, message, code)
		return
	}

	result, err := h.service.GetPublisher(c.Request.Context(), id)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Publisher retrieved successfully", result)
}

// GetPublisherBySlug handles GET /publishers/slug/:slug
func (h *PublisherHandler) GetPublisherBySlug(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		statusCode, message, code := publisher.GetErrorResponse(publisher.NewInvalidSlug(""))
		response.Error(c, statusCode, message, code)
		return
	}

	result, err := h.service.GetPublisherBySlug(c.Request.Context(), slug)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Publisher retrieved successfully", result)
}

// ListPublishers handles GET /publishers
func (h *PublisherHandler) ListPublishers(c *gin.Context) {
	page := 1
	pageSize := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	results, total, err := h.service.ListPublishers(c.Request.Context(), page, pageSize)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	responseData := gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	response.Success(c, http.StatusOK, "Publishers retrieved successfully", responseData)
}

// UpdatePublisher handles PUT /publishers/:id
func (h *PublisherHandler) UpdatePublisher(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(publisher.NewInvalidPublisherID(idStr))
		response.Error(c, statusCode, message, code)
		return
	}

	var req publisher.PublisherUpdateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.UpdatePublisher(c.Request.Context(), id, &req)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Publisher updated successfully", result)
}

// DeletePublisher handles DELETE /publishers/:id
func (h *PublisherHandler) DeletePublisher(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(publisher.NewInvalidPublisherID(idStr))
		response.Error(c, statusCode, message, code)
		return
	}

	err = h.service.DeletePublisher(c.Request.Context(), id)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Publisher deleted successfully", nil)
}

// GetPublisherWithBooks handles GET /publishers/:id/books
func (h *PublisherHandler) GetPublisherWithBooks(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(publisher.NewInvalidPublisherID(idStr))
		response.Error(c, statusCode, message, code)
		return
	}

	result, err := h.service.GetPublisherWithBooks(c.Request.Context(), id)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Publisher with books retrieved successfully", result)
}

// ListPublishersWithBooks handles GET /publishers/books
func (h *PublisherHandler) ListPublishersWithBooks(c *gin.Context) {
	page := 1
	pageSize := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	results, total, err := h.service.ListPublishersWithBooks(c.Request.Context(), page, pageSize)
	if err != nil {
		statusCode, message, code := publisher.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	responseData := gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	response.Success(c, http.StatusOK, "Publishers with books retrieved successfully", responseData)
}
