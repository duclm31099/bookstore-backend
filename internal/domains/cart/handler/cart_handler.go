package cart

import (
	"errors"
	"fmt"
	"net/http"

	"bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/internal/domains/cart/service"
	promotionService "bookstore-backend/internal/domains/promotion/service"
	"bookstore-backend/internal/shared/middleware"
	cartMiddleware "bookstore-backend/internal/shared/middleware"
	"bookstore-backend/internal/shared/response"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for cart
type Handler struct {
	service          service.ServiceInterface
	promotionService promotionService.ServiceInterface
}

// NewHandler creates handler instance
func NewHandler(service service.ServiceInterface, promoService promotionService.ServiceInterface) *Handler {
	return &Handler{
		service:          service,
		promotionService: promoService,
	}
}

// ===================================
// API 1: GET /me/cart
// ===================================

// GetCart handles GET /me/cart
// @Summary Get current user's shopping cart
// @Description Retrieves cart with all items, prices, and book details
// @Router /me/cart [get]
func (h *Handler) GetCart(c *gin.Context) {
	// Get cart_id from middleware context
	// cartid, err := cartMiddleware.GetCartID(c)
	// if err != nil {
	// 	response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
	// 	return
	// }
	// logger.Info("Get cart id", map[string]interface{}{
	// 	"cartid": cartid,
	// })
	// Check if authenticated or anonymous
	// isAnonymous := cartMiddleware.IsAnonymousCart(c)
	userID, _ := c.Get(cartMiddleware.ContextKeyUserID)
	sessionID := cartMiddleware.GetSessionID(c)

	var uid *uuid.UUID
	var sid *string

	if userID != nil {
		if id, ok := userID.(uuid.UUID); ok {
			uid = &id
		}
	} else if sessionID != "" {
		sid = &sessionID
	}

	// Get or create cart
	cart, err := h.service.GetOrCreateCart(c.Request.Context(), uid, sid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get cart", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Cart retrieved successfully", cart)
}

// ===================================
// API 2: POST /me/cart/items
// ===================================

// AddItem handles POST /me/cart/items
// @Summary Add book to cart
// @Description Adds item to cart. Creates if doesn't exist, updates if already there.
// @Router /me/cart/items [post]
func (h *Handler) AddItem(c *gin.Context) {
	// Get cart_id from middleware
	cartID, err := cartMiddleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	// Parse request
	var req model.AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Add item
	item, err := h.service.AddItem(c.Request.Context(), cartID, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to add item", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Item added to cart", item)
}

// ===================================
// API 3: GET /me/cart/items?page=1&limit=20
// ===================================

// ListItems handles GET /me/cart/items
// @Summary List cart items with pagination
// @Description Returns paginated cart items with book details and prices
func (h *Handler) ListItems(c *gin.Context) {
	// Get cart_id from middleware
	cartID, err := cartMiddleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	// Parse pagination
	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// List items
	result, err := h.service.ListItems(c.Request.Context(), cartID, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list items", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Cart items retrieved", result)
}

// domains/cart/handler.go

// UpdateItemQuantity handles PUT /me/cart/items/{item_id}
// @Summary Update cart item quantity
// @Description Updates quantity of item. If quantity=0, removes item.
// @Tags Cart
// @Accept json
// @Produce json
// @Param item_id path string true "Item ID (UUID)"
// @Param request body model.UpdateCartItemRequest true "Update Request"
// @Success 200 {object} SuccessResponse{data=model.CartItemResponse}
// @Router /me/cart/items/{item_id} [put]
func (h *Handler) UpdateItemQuantity(c *gin.Context) {
	// Get cart_id from middleware
	cartID, err := middleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	// Parse item_id from path
	itemIDParam := c.Param("item_id")
	itemID, err := uuid.Parse(itemIDParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid item ID", err.Error())
		return
	}

	// Parse request
	var req model.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Update item
	item, err := h.service.UpdateItemQuantity(c.Request.Context(), cartID, itemID, req.Quantity)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidQuantity):
			response.Error(c, http.StatusBadRequest, "Invalid quantity", err.Error())
		case errors.Is(err, model.ErrCartItemNotFound):
			response.Error(c, http.StatusNotFound, "Item not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to update item", err.Error())
		}
		return
	}

	// Handle removal (quantity = 0)
	if item == nil {
		response.Success(c, http.StatusOK, "Item removed from cart", gin.H{
			"message": "Item quantity set to 0 - item removed",
		})
		return
	}

	response.Success(c, http.StatusOK, "Item quantity updated", item)
}

// domains/cart/handler.go

// RemoveItem handles DELETE /me/cart/items/{item_id}
// @Summary Remove item from cart
// @Description Removes item completely from cart
// @Tags Cart
// @Produce json
// @Param item_id path string true "Item ID (UUID)"
// @Router /me/cart/items/{item_id} [delete]
func (h *Handler) RemoveItem(c *gin.Context) {
	// Get cart_id from middleware
	cartID, err := middleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	// Parse item_id from path
	itemIDParam := c.Param("item_id")
	itemID, err := uuid.Parse(itemIDParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid item ID", err.Error())
		return
	}

	// Remove item
	err = h.service.RemoveItem(c.Request.Context(), cartID, itemID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrCartItemNotFound):
			response.Error(c, http.StatusNotFound, "Item not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to remove item", err.Error())
		}
		return
	}

	// Success - 204 No Content
	c.Status(http.StatusNoContent)
}

// ClearCart handles DELETE /me/cart
// @Summary Clear entire cart
// @Description Removes all items from cart but keeps the cart
func (h *Handler) ClearCart(c *gin.Context) {
	// Get cart_id from middleware
	cartID, err := middleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	// Clear cart
	deletedCount, err := h.service.ClearCart(c.Request.Context(), cartID)
	if err != nil {
		// Map custom errors to HTTP status
		switch {
		case errors.Is(err, model.ErrCartNotFound):
			response.Error(c, http.StatusNotFound, "Cart not found", nil)
		case errors.Is(err, model.ErrCartExpired):
			response.Error(c, http.StatusGone, "Cart has expired", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to clear cart", err.Error())
		}
		return
	}

	// Return success with deleted count
	resp := map[string]interface{}{
		"deleted_count": deletedCount,
		"message":       fmt.Sprintf("Cleared %d items from cart", deletedCount),
	}
	response.Success(c, http.StatusOK, "Success", resp)
}

// ValidateCart handles POST /me/cart/validate
// @Summary Validate cart before checkout
// @Description Checks if cart is valid, items in stock, prices current
func (h *Handler) ValidateCart(c *gin.Context) {
	// Get cartID from middleware
	cartID, err := middleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	// Get userID (optional for anonymous)
	var userID uuid.UUID
	userIDValue, exists := c.Get("user_id")
	if exists {
		var ok bool
		userID, ok = userIDValue.(uuid.UUID)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "Invalid user ID", nil)
			return
		}
	}

	// Validate cart
	result, err := h.service.ValidateCart(c.Request.Context(), cartID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Validation failed", err.Error())
		return
	}

	// Determine HTTP status
	statusCode := http.StatusOK
	if result.CartStatus == "error" {
		statusCode = http.StatusUnprocessableEntity // 422
	}

	response.Success(c, statusCode, "Cart validation completed", result)
}

// domains/cart/handler.go

// ApplyPromoCode handles POST /cart/apply-promotion
// @Summary Apply promo code
// @Description Applies promo code and calculates discount
func (h *Handler) ApplyPromoCode(c *gin.Context) {
	userIdVal, exist := c.Get("user_id")

	// 1) Kiểm tra tồn tại
	if !exist || userIdVal == nil {
		response.Error(c, http.StatusUnauthorized, "User not login", nil)
		return
	}

	// 2) Kiểm tra kiểu cho chắc
	userIdStr := fmt.Sprintf("%v", userIdVal)
	if userIdStr == "" {
		response.Error(c, http.StatusUnauthorized, "Can not parse to string", nil)
		return
	}

	// 3) Parse UUID an toàn
	uid := utils.ParseStringToUUID(userIdStr)
	if uid == uuid.Nil {
		// tùy bạn: có thể coi là lỗi auth
		response.Error(c, http.StatusUnauthorized, "Can not parse to uuid", nil)
		return
	}

	cartID, err := middleware.GetCartID(c)

	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	var req model.ApplyPromoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("bind promo code failed", err)
		response.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	result, err := h.service.ApplyPromoCode(c.Request.Context(), cartID, req.PromoCode, uid)
	if err != nil {
		logger.Error("apply promo code failed", err)
		response.Error(c, http.StatusBadRequest, "Invalid promo code", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Promo applied successfully", result)
}

// RemovePromoCode handles DELETE /me/cart/promo
// @Summary Remove promo code
// @Description Removes promo code and recalculates total
// @Tags Cart
// @Produce json
// @Success 204 {object} string "No Content"
// @Failure 400 {object} ErrorResponse
// @Router /me/cart/promo [delete]
func (h *Handler) RemovePromoCode(c *gin.Context) {
	cartID, err := middleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart", err.Error())
		return
	}

	err = h.service.RemovePromoCode(c.Request.Context(), cartID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to remove promo", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAvailablePromotions handles GET /cart/:cart_id/promotions
// @Summary Get available promotions for cart
// @Description Returns list of promotions that can be applied to the cart
// @Tags Cart
// @Produce json
// @Param cart_id path string true "Cart ID (UUID)"
// @Success 200 {object} SuccessResponse{data=[]model.AvailablePromotionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/{cart_id}/promotions [get]
func (h *Handler) GetAvailablePromotions(c *gin.Context) {
	// Parse cart_id from URL
	cartIDParam := c.Param("cart_id")
	cartID, err := uuid.Parse(cartIDParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid cart ID", err.Error())
		return
	}

	// Get userID from JWT context
	userIDValue, exists := c.Get(middleware.ContextKeyUserID)
	if !exists || userIDValue == nil {
		response.Error(c, http.StatusUnauthorized, "Not authenticated", "User ID required")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Invalid user ID", "User ID must be UUID")
		return
	}

	// Get available promotions
	promotions, err := h.promotionService.GetAvailablePromotionsForCart(c.Request.Context(), cartID, userID)
	if err != nil {
		logger.Info("Failed to get available promotions", map[string]interface{}{
			"cart_id": cartID,
			"user_id": userID,
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "Failed to get available promotions", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Available promotions retrieved successfully", promotions)
}

// domains/cart/handler.go

// domains/cart/handler.go

// Checkout handles POST /me/cart/checkout
// @Summary Complete checkout process
// @Description Converts cart to order with full validation
// @Router /me/cart/checkout [post]
func (h *Handler) Checkout(c *gin.Context) {
	// ===================================
	// STEP 1: Extract user ID (REQUIRED)
	// ===================================
	userIDValue, exists := c.Get(middleware.ContextKeyUserID)
	if !exists || userIDValue == nil {
		response.Error(c, http.StatusUnauthorized,
			"Not authenticated",
			"User ID required for checkout")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		response.Error(c, http.StatusUnauthorized,
			"Invalid user ID",
			"User ID must be UUID")
		return
	}

	// ===================================
	// STEP 2: Extract cart ID
	// ===================================
	cartID, err := middleware.GetCartID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest,
			"Invalid cart",
			err.Error())
		return
	}

	// ===================================
	// STEP 3: Parse request
	// ===================================
	var req model.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest,
			"Invalid request",
			err.Error())
		return
	}

	// ===================================
	// STEP 4: Validate request (struct validation)
	// ===================================
	// if err := validator.New().Struct(req); err != nil {
	// 	response.Error(c, http.StatusBadRequest,
	// 		"Validation failed",
	// 		err.Error())
	// 	return
	// }

	// ===================================
	// STEP 5: Call service
	// ===================================
	result, err := h.service.Checkout(c.Request.Context(), userID, cartID, req)
	if err != nil {
		// System error (not validation error)
		response.Error(c, http.StatusInternalServerError,
			"Checkout failed",
			err.Error())
		return
	}

	// ===================================
	// STEP 6: Determine response status
	// ===================================
	statusCode := http.StatusCreated // 201

	if !result.Success {
		statusCode = http.StatusUnprocessableEntity // 422
	}

	response.Success(c, statusCode, "Checkout completed", result)
}
