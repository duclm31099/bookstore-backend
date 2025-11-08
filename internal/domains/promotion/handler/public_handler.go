package handler

import (
	"net/http"
	"strconv"
	"time"

	cart "bookstore-backend/internal/domains/cart/service"
	dto "bookstore-backend/internal/domains/promotion/model"
	"bookstore-backend/internal/domains/promotion/service"
	"bookstore-backend/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PublicHandler xử lý các API công khai (user-facing)
type PublicHandler struct {
	service service.ServiceInterface
	cart    cart.ServiceInterface
}

// NewPublicHandler tạo handler instance
func NewPublicHandler(promotionService service.ServiceInterface, cartService cart.ServiceInterface) *PublicHandler {
	return &PublicHandler{
		service: promotionService,
		cart:    cartService,
	}
}

// ValidatePromotion validates promotion code với cart items
//
// @Summary      Validate promotion code
// @Description  Kiểm tra mã giảm giá có hợp lệ với giỏ hàng không
// @Tags         promotions
// @Accept       json
// @Produce      json
// @Param        request body dto.ValidatePromotionRequest true "Validate request"
// @Success      200 {object} commonDTO.SuccessResponse{data=dto.ValidationResult}
// @Failure      400 {object} commonDTO.ErrorResponse
// @Failure      404 {object} commonDTO.ErrorResponse
// @Router       /v1/promotions/validate [post]
func (h *PublicHandler) ValidatePromotion(c *gin.Context) {
	var req dto.ValidatePromotionRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {

		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      string(dto.ErrCodeValidationFailed),
		})
		return
	}

	// Verify subtotal matches cart items
	calculatedSubtotal := req.CalculateSubtotal()
	if !calculatedSubtotal.Equal(req.Subtotal) {
		response.Error(c, http.StatusBadRequest, "Tổng tiền giỏ hàng không khớp", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      string(dto.ErrCodeInvalidSubtotal),
		})
		return
	}

	// Get user ID from JWT if authenticated (optional)
	userID := getUserIDFromContext(c) // nil if not authenticated
	req.UserID = userID

	// Call service
	result, err := h.service.ValidatePromotion(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Validate promotion successfully", result)
}

// ListActivePromotions lấy danh sách promotion đang active
//
// @Summary      List active promotions
// @Description  Lấy danh sách các mã giảm giá đang hoạt động
// @Tags         promotions
// @Accept       json
// @Produce      json
// @Param        category_id query string false "Filter by category ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} commonDTO.SuccessResponseWithMeta{data=[]model.Promotion}
// @Failure      400 {object} commonDTO.ErrorResponse
// @Router       /v1/promotions [get]
func (h *PublicHandler) ListActivePromotions(c *gin.Context) {
	// Parse query params
	var categoryID *uuid.UUID
	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		id, err := uuid.Parse(categoryIDStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Category ID không hợp lệ", gin.H{
				"RequestID": c.GetString("request_id"),
				"Timestamp": time.Now(),
				"Code":      string(dto.ErrCodeValidationFailed),
			})
			return
		}
		categoryID = &id
	}

	// Pagination
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	// Call service
	promotions, total, err := h.service.ListActivePromotions(c.Request.Context(), categoryID, page, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, "List active promotions", gin.H{
		"promotions": promotions,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages, // auto calc
		},
	})
}

// ApplyPromotionToCart áp dụng mã giảm giá vào giỏ hàng
// @Summary      Apply promotion to cart
// @Description  Áp dụng mã giảm giá vào giỏ hàng của user
// @Router       /v1/cart/promo [post]
func (h *PublicHandler) ApplyPromotionToCart(c *gin.Context) {
	var req dto.ApplyPromoRequest

	// Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      string(dto.ErrCodeValidationFailed),
		})
		return
	}

	// Validate
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      string(dto.ErrCodeValidationFailed),
		})
		return
	}

	// Get user ID from JWT (required)
	userID := getUserIDFromContext(c)
	if userID == nil {
		response.Error(c, http.StatusBadRequest, "Vui lòng đăng nhập", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      "UNAUTHORIZED",
		})
		return
	}

	// ===================================================================
	// BUSINESS LOGIC: Apply promotion to cart
	// ===================================================================
	// 1. Get user's cart from CartService
	// 2. Build ValidatePromotionRequest from cart items
	// 3. Validate promotion
	// 4. If valid: Store promo code in cart (Redis/DB)
	// 5. Recalculate cart total with discount
	// 6. Return updated cart
	// ===================================================================

	ctx := c.Request.Context()

	// Step 1: Get cart (giả sử có CartService)
	cart, err := h.cart.GetOrCreateCart(ctx, userID, nil)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Step 2: Build validation request
	cartItems := make([]dto.CartItem, len(cart.Items))
	subtotal := decimal.Zero

	for i, item := range cart.Items {
		cartItems[i] = dto.CartItem{
			BookID:   item.BookID,
			Price:    item.Price,
			Quantity: item.Quantity,
		}
		subtotal = subtotal.Add(item.Price.Mul(decimal.NewFromInt(int64(item.Quantity))))
	}

	validateReq := &dto.ValidatePromotionRequest{
		Code:      req.Code,
		CartItems: cartItems,
		Subtotal:  subtotal,
		UserID:    userID,
	}

	// Step 3: Validate promotion
	validationResult, err := h.service.ValidatePromotion(ctx, validateReq)
	if err != nil || !validationResult.IsValid {
		h.handleError(c, err)
		return
	}

	// Step 4: Store promo code in cart
	updatedCart, err := h.service.ApplyPromotionToCart(ctx, *userID, req.Code)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Step 6: Success response
	response.Success(c, http.StatusOK, "Apply promotion to cart", updatedCart)
}

// RemovePromotionFromCart xóa mã giảm giá khỏi giỏ hàng
// @Summary      Remove promotion from cart
// @Description  Xóa mã giảm giá đang áp dụng trong giỏ hàng
// @Router       /v1/cart/promo [delete]
func (h *PublicHandler) RemovePromotionFromCart(c *gin.Context) {
	// Get user ID
	userID := getUserIDFromContext(c)
	if userID == nil {
		response.Error(c, http.StatusBadRequest, "Vui lòng đăng nhập", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      "UNAUTHORIZED",
		})
		return
	}

	// Remove promotion from cart
	cart, err := h.service.RemovePromotionFromCart(c.Request.Context(), *userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Step 6: Success response
	response.Success(c, http.StatusOK, "Remove promotion to cart", cart)
}

// -------------------------------------------------------------------
// HELPER FUNCTIONS
// -------------------------------------------------------------------

// handleError xử lý errors và trả về response phù hợp
func (h *PublicHandler) handleError(c *gin.Context, err error) {
	// Check if it's AppError
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Đã có lỗi xảy ra, vui lòng thử lại sau", gin.H{
			"RequestID": c.GetString("request_id"),
			"Timestamp": time.Now(),
			"Code":      err.Error(),
		})
		return
	}
}

// getUserIDFromContext lấy user ID từ JWT context
func getUserIDFromContext(c *gin.Context) *uuid.UUID {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return nil
	}

	switch v := userIDValue.(type) {
	case uuid.UUID:
		return &v
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return nil
		}
		return &id
	default:
		return nil
	}
}

// parseIntQuery parse integer query param với default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
