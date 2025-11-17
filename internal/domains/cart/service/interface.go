package service

import (
	"bookstore-backend/internal/domains/cart/model"
	"context"

	"github.com/google/uuid"
)

type ServiceInterface interface {
	// GetOrCreateCart returns existing cart or creates new one
	// For authenticated: use userID
	// For anonymous: use sessionID
	// Returns: full cart with items
	GetOrCreateCart(ctx context.Context, userID *uuid.UUID, sessionID *string) (*model.CartResponse, error)

	// AddItem adds book to cart or updates quantity if exists
	// Validates: book exists, has stock
	AddItem(ctx context.Context, cartID uuid.UUID, req model.AddToCartRequest) (*model.CartItemResponse, error)

	// ListItems returns paginated cart items with book details
	ListItems(ctx context.Context, cartID uuid.UUID, page int, limit int) (*model.CartResponse, error)

	// GetUserCartID retrieves cart ID for authenticated user (for middleware)
	GetUserCartID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)

	// GetOrCreateCartBySession gets or creates cart for anonymous (for middleware)
	GetOrCreateCartBySession(ctx context.Context, sessionID string) (uuid.UUID, error)
	// MergeCart merges anonymous cart into authenticated user's cart
	// Used when user logs in - transfers items from session cart to user cart
	// Strategy:
	//   - If item exists in both carts → keep higher quantity
	//   - If item only in anonymous cart → move to user cart
	//   - Delete anonymous cart after merge
	// Returns: error if merge fails
	MergeCart(ctx context.Context, sessionID string, userID uuid.UUID) error

	// UpdateItemQuantity updates quantity of item in cart
	// Returns: updated item response
	// Validates: quantity > 0, quantity ≤ 100
	// If quantity = 0 → removes item
	UpdateItemQuantity(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID, quantity int) (*model.CartItemResponse, error)
	ValidatePromoCode(ctx context.Context, req *model.ValidatePromoRequest) (*model.PromotionValidationResult, error)
	// RemoveItem removes item from cart
	// Returns: error if item not found
	RemoveItem(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID) error

	// ClearCart removes all items from cart but keeps cart itself
	// Used when user wants to empty cart
	// Returns: error if failed
	ClearCart(ctx context.Context, cartID uuid.UUID) (int, error)
	// ValidateCart validates cart before checkout
	// Returns: validation result with errors and warnings
	// Does NOT modify cart
	ValidateCart(ctx context.Context, cartID uuid.UUID, userId uuid.UUID) (*model.CartValidationResult, error)

	// ApplyPromoCode applies promo code to cart
	// Returns: discount info if valid, error if invalid/expired
	ApplyPromoCode(ctx context.Context, cartID uuid.UUID, promoCode string, userId uuid.UUID) (*model.ApplyPromoResponse, error)

	// RemovePromoCode removes promo from cart
	RemovePromoCode(ctx context.Context, cartID uuid.UUID) error

	// Checkout performs complete checkout transaction
	// Includes: validation, reservation, order creation, cleanup
	//
	// Returns:
	//   - Success: CheckoutResponse with OrderID
	//   - Validation Error: CheckoutResponse with phase info
	//   - System Error: error
	//
	// Phases (in order):
	//   1. PRE_VALIDATION - Check basic requirements
	//   2. CART_VALIDATION - Check cart items, stock, prices
	//   3. ADDRESS_VALIDATION - Check shipping/billing addresses
	//   4. PROMO_VALIDATION - Validate promo code
	//   5. INVENTORY_RESERVATION - Reserve stock (CRITICAL)
	//   6. ORDER_CREATION - Create order record
	//   7. PAYMENT_PROCESSING - Process payment (async ok)
	//   8. CLEANUP - Clear cart, send confirmations
	Checkout(ctx context.Context, userID uuid.UUID, cartID uuid.UUID, req model.CheckoutRequest) (*model.CheckoutResponse, error)
}
