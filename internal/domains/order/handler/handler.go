package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/order/model"
	"bookstore-backend/internal/domains/order/service"
	"bookstore-backend/internal/shared/response"
)

// =====================================================
// ORDER HANDLER
// =====================================================
type OrderHandler struct {
	orderService service.OrderService
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// =====================================================
// ROUTES REGISTRATION
// =====================================================

// RegisterRoutes registers all order routes
func (h *OrderHandler) RegisterRoutes(router *gin.RouterGroup) {
	// User routes (protected by auth middleware)
	userRoutes := router.Group("/orders")
	{
		userRoutes.GET("", h.ListOrders)                           // GET /v1/orders?page=1&limit=20&status=pending
		userRoutes.GET("/:id", h.GetOrderDetail)                   // GET /v1/orders/:id
		userRoutes.GET("/number/:orderNumber", h.GetOrderByNumber) // GET /v1/orders/number/ORD-20251108-001
		userRoutes.PATCH("/:id/cancel", h.CancelOrder)             // PATCH /v1/orders/:id/cancel
		userRoutes.POST("/reorder", h.ReorderFromExisting)         // POST /v1/orders/reorder
	}

	// Admin routes (protected by admin middleware)
	adminRoutes := router.Group("/admin/orders")
	{
		adminRoutes.GET("", h.ListAllOrders)                  // GET /v1/admin/orders
		adminRoutes.PATCH("/:id/status", h.UpdateOrderStatus) // PATCH /v1/admin/orders/:id/status
	}
}

// =====================================================
// GET ORDER DETAIL
// =====================================================
// @Summary Get order detail
// @Description Get detailed information of a specific order
// @Router /v1/orders/{id} [get]
func (h *OrderHandler) GetOrderDetail(c *gin.Context) {
	// Extract user_id from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Parse order ID from URL parameter
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid order ID", map[string]string{
			"error": "Order ID must be a valid UUID",
		})
		return
	}

	// Call service
	result, err := h.orderService.GetOrderDetail(c.Request.Context(), orderID, userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusOK, "OK", result)
}

// =====================================================
// GET ORDER BY NUMBER
// =====================================================
// @Summary Get order by order number
// @Description Get order details by order number (e.g., ORD-20251108-001)
func (h *OrderHandler) GetOrderByNumber(c *gin.Context) {
	// Extract user_id from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Get order number from URL parameter
	orderNumber := c.Param("orderNumber")
	if orderNumber == "" {
		response.Error(c, http.StatusBadRequest, "Order number is required", nil)
		return
	}

	// Call service
	result, err := h.orderService.GetOrderByNumber(c.Request.Context(), orderNumber, userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusOK, "OK", result)
}

// =====================================================
// LIST ORDERS
// =====================================================
// @Summary List user's orders
// @Description Get paginated list of user's orders with optional status filter
func (h *OrderHandler) ListOrders(c *gin.Context) {
	// Extract user_id from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Bind query parameters
	var req model.ListOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Set defaults if not provided
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Call service
	result, err := h.orderService.ListOrders(c.Request.Context(), userID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusOK, "OK", result)
}

// =====================================================
// CANCEL ORDER
// =====================================================
// @Summary Cancel order
// @Description Cancel an order (only pending or confirmed orders can be cancelled)
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	// Extract user_id from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Parse order ID from URL parameter
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid order ID", map[string]string{
			"error": "Order ID must be a valid UUID",
		})
		return
	}

	// Bind request
	var req model.CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "Validation failed", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Call service
	err = h.orderService.CancelOrder(c.Request.Context(), orderID, userID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusOK, "Order cancelled successfully", nil)
}

// =====================================================
// REORDER FROM EXISTING ORDER
// =====================================================
// ReorderFromExisting godoc
// @Summary Reorder from existing order
// @Description Create a new order based on an existing order
func (h *OrderHandler) ReorderFromExisting(c *gin.Context) {
	// Extract user_id from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Bind request
	var req model.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Call service
	result, err := h.orderService.ReorderFromExisting(c.Request.Context(), userID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusCreated, "Order created successfully", result)
}

// =====================================================
// ADMIN: LIST ALL ORDERS
// =====================================================
// @Summary Admin: List all orders
// @Description Get paginated list of all orders (admin only)
func (h *OrderHandler) ListAllOrders(c *gin.Context) {
	// Extract user_id from context (admin middleware should validate admin role)
	_, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Bind query parameters
	var req model.ListOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Call service
	result, err := h.orderService.ListAllOrders(c.Request.Context(), req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusOK, "OK", result)
}

// =====================================================
// ADMIN: UPDATE ORDER STATUS
// =====================================================
// @Summary Admin: Update order status
// @Description Update order status with version control (admin only)
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	// Extract user_id from context (admin)
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Parse order ID from URL parameter
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid order ID", map[string]string{
			"error": "Order ID must be a valid UUID",
		})
		return
	}

	// Bind request
	var req model.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "Validation failed", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Call service
	err = h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, userID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Return success response
	response.Success(c, http.StatusOK, "Order status updated successfully", nil)
}

// =====================================================
// HELPER METHODS
// =====================================================

// getUserIDFromContext extracts user ID from gin context
// Assumes auth middleware sets "user_id" in context
func (h *OrderHandler) getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	// Try to get user_id from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id not found in context")
	}

	// Type assertion - handle both string and uuid.UUID types
	switch v := userIDInterface.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		return uuid.Parse(v)
	default:
		return uuid.Nil, errors.New("invalid user_id type in context")
	}
}

// handleServiceError handles service layer errors and maps to HTTP responses
func (h *OrderHandler) handleServiceError(c *gin.Context, err error) {
	// Check if it's a custom OrderError
	var orderErr *model.OrderError
	if errors.As(err, &orderErr) {
		// Map error code to HTTP status
		statusCode := h.getHTTPStatusFromErrorCode(orderErr.Code)
		response.Error(c, statusCode, orderErr.Message, map[string]string{
			"code": orderErr.Code,
		})
		return
	}

	// Check for common errors
	if errors.Is(err, model.ErrOrderNotFound) {
		response.Error(c, http.StatusNotFound, "Order not found", map[string]string{
			"code": model.ErrCodeOrderNotFound,
		})
		return
	}

	if errors.Is(err, model.ErrVersionMismatch) {
		response.Error(c, http.StatusConflict, "Concurrent modification detected. Please refresh and try again.", map[string]string{
			"code": model.ErrCodeVersionMismatch,
		})
		return
	}

	if errors.Is(err, model.ErrOrderCannotCancel) {
		response.Error(c, http.StatusUnprocessableEntity, "Order cannot be cancelled", map[string]string{
			"code": model.ErrCodeOrderCannotCancel,
		})
		return
	}

	if errors.Is(err, model.ErrUnauthorized) {
		response.Error(c, http.StatusForbidden, "Unauthorized access", map[string]string{
			"code": model.ErrCodeUnauthorized,
		})
		return
	}

	// Default internal server error
	response.Error(c, http.StatusInternalServerError, "Internal server error", map[string]string{
		"error": err.Error(),
	})
}

// getHTTPStatusFromErrorCode maps business error codes to HTTP status codes
func (h *OrderHandler) getHTTPStatusFromErrorCode(code string) int {
	statusMap := map[string]int{
		model.ErrCodeOrderNotFound:          http.StatusNotFound,
		model.ErrCodeOrderCannotCancel:      http.StatusUnprocessableEntity,
		model.ErrCodeVersionMismatch:        http.StatusConflict,
		model.ErrCodeInsufficientStock:      http.StatusUnprocessableEntity,
		model.ErrCodePromoInvalid:           http.StatusUnprocessableEntity,
		model.ErrCodePromoExpired:           http.StatusUnprocessableEntity,
		model.ErrCodePromoUsageLimitReached: http.StatusUnprocessableEntity,
		model.ErrCodeMinOrderAmount:         http.StatusUnprocessableEntity,
		model.ErrCodePaymentTimeout:         http.StatusRequestTimeout,
		model.ErrCodeInvalidWarehouse:       http.StatusBadRequest,
		model.ErrCodeInvalidAddress:         http.StatusBadRequest,
		model.ErrCodeCartEmpty:              http.StatusBadRequest,
		model.ErrCodeInvalidPaymentMethod:   http.StatusBadRequest,
		model.ErrCodeUnauthorized:           http.StatusForbidden,
		model.ErrCodeInvalidStatus:          http.StatusUnprocessableEntity,
		model.ErrCodePromoMinAmount:         http.StatusUnprocessableEntity,
	}

	if status, exists := statusMap[code]; exists {
		return status
	}

	return http.StatusInternalServerError
}
