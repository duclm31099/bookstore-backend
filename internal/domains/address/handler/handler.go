package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/address/model"
	"bookstore-backend/internal/domains/address/service"
	"bookstore-backend/internal/shared/response"
)

type AddressHandler struct {
	service service.ServiceInterface
}

func NewAddressHandler(service service.ServiceInterface) *AddressHandler {
	return &AddressHandler{
		service: service,
	}
}

// CreateAddress handles POST /addresses
func (h *AddressHandler) CreateAddress(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	var req model.AddressCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.CreateAddress(c.Request.Context(), userID, &req)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusCreated, "Address created successfully", result)
}

// GetAddress handles GET /addresses/:id
func (h *AddressHandler) GetAddressById(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	addressID, err := getAddressId(c)
	if err != nil {
		// Error response đã được gửi trong getAddressId
		return
	}

	result, err := h.service.GetAddressByID(c.Request.Context(), userID, addressID)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Address retrieved successfully", result)
}

// ListUserAddresses handles GET /addresses
func (h *AddressHandler) ListUserAddresses(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	results, err := h.service.ListUserAddresses(c.Request.Context(), userID)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	responseData := gin.H{
		"data":  results,
		"total": len(results),
	}

	response.Success(c, http.StatusOK, "Addresses retrieved successfully", responseData)
}

// GetDefaultAddress handles GET /addresses/default
func (h *AddressHandler) GetDefaultAddress(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	result, err := h.service.GetDefaultAddress(c.Request.Context(), userID)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Default address retrieved successfully", result)
}

// UpdateAddress handles PUT /addresses/:id
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	addressID, err := getAddressId(c)
	if err != nil {
		// Error response đã được gửi trong getAddressId
		return
	}

	var req model.AddressUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.UpdateAddress(c.Request.Context(), userID, addressID, &req)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Address updated successfully", result)
}

// DeleteAddress handles DELETE /addresses/:id
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	addressID, err := getAddressId(c)
	if err != nil {
		// Error response đã được gửi trong getAddressId
		return
	}

	err = h.service.DeleteAddress(c.Request.Context(), userID, addressID)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Address deleted successfully", nil)
}

// SetDefaultAddress handles PUT /addresses/:id/set-default
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	addressID, err := getAddressId(c)
	if err != nil {
		// Error response đã được gửi trong getAddressId
		return
	}

	result, err := h.service.SetDefaultAddress(c.Request.Context(), userID, addressID)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Address set as default successfully", result)
}

// UnsetDefaultAddress handles PUT /addresses/:id/unset-default
func (h *AddressHandler) UnsetDefaultAddress(c *gin.Context) {
	userID, err := getUserContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	addressID, err := getAddressId(c)
	if err != nil {
		// Error response đã được gửi trong getAddressId
		return
	}

	err = h.service.UnsetDefaultAddress(c.Request.Context(), userID, addressID)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Default flag removed from address", nil)
}

// Admin: GetAddressWithUser handles GET /admin/addresses/:id
func (h *AddressHandler) GetAddressWithUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(model.NewInvalidAddressID(idStr))
		response.Error(c, statusCode, message, code)
		return
	}

	result, err := h.service.GetAddressWithUser(c.Request.Context(), id)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	response.Success(c, http.StatusOK, "Address with user retrieved successfully", result)
}

// Admin: ListAllAddresses handles GET /admin/addresses
func (h *AddressHandler) ListAllAddresses(c *gin.Context) {
	page := 1
	pageSize := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	results, total, err := h.service.ListAllAddresses(c.Request.Context(), page, pageSize)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(err)
		response.Error(c, statusCode, message, code)
		return
	}

	responseData := gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	response.Success(c, http.StatusOK, "Addresses retrieved successfully", responseData)
}

// Helper: Get user ID from context
func getUserContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user not authenticated")
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID type")
	}

	if uid == uuid.Nil {
		return uuid.Nil, errors.New("user ID cannot be nil")
	}

	return uid, nil
}

// Helper: Get address ID from URL param
func getAddressId(c *gin.Context) (uuid.UUID, error) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		statusCode, message, code := model.GetErrorResponse(model.NewInvalidAddressID(idStr))
		response.Error(c, statusCode, message, code)
		return uuid.Nil, err
	}

	return id, nil
}
