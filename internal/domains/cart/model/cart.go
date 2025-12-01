package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AddToCartRequest represents request to add item to cart
type AddToCartRequest struct {
	BookID   uuid.UUID `json:"book_id" validate:"required"`
	Quantity int       `json:"quantity" validate:"required,gte=1,lte=100"`
}

// UpdateCartItemRequest represents request to update cart item quantity
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gte=1,lte=100"`
}

// CartResponse represents the full cart response with items
type CartResponse struct {
	ID         uuid.UUID          `json:"id"`
	UserID     *uuid.UUID         `json:"user_id,omitempty"`
	SessionID  *string            `json:"session_id,omitempty"`
	Items      []CartItemResponse `json:"items"`
	ItemsCount int                `json:"items_count"`
	Subtotal   decimal.Decimal    `json:"subtotal"`

	// Promo information (if applied)
	PromoCode      *string                `json:"promo_code,omitempty"`
	DiscountAmount *decimal.Decimal       `json:"discount_amount,omitempty"`
	Total          *decimal.Decimal       `json:"total,omitempty"`
	PromoMetadata  map[string]interface{} `json:"promo_metadata,omitempty" db:"promo_metadata"` // ✅ JSONB

	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// CartItemResponse represents cart item with book details
type CartItemResponse struct {
	ID             uuid.UUID       `json:"id"`
	BookID         uuid.UUID       `json:"book_id"`
	CartID         uuid.UUID       `json:"cart_id"`
	Quantity       int             `json:"quantity"`
	Price          decimal.Decimal `json:"price"`    // Snapshot price
	Subtotal       decimal.Decimal `json:"subtotal"` // quantity * price
	CompareAtPrice decimal.Decimal `json:"compare_at_price"`
	// Book details (from JOIN)
	BookTitle      string          `json:"book_title"`
	BookSlug       string          `json:"book_slug"`
	BookCoverURL   *string         `json:"book_cover_url,omitempty"`
	BookAuthor     string          `json:"book_author"`
	CurrentPrice   decimal.Decimal `json:"current_price"` // Current book price (may differ from snapshot)
	IsAvailable    bool            `json:"is_available"`
	AvailableStock int             `json:"available_stock"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	TotalStock     int             `json:"total_stock"`
	UpdatedAt      time.Time       `json:"updated_at"`
	CategoryName   *string         `json:"category_name"`
	CategoryID     *uuid.UUID      `json:"category_id"`
}

// CartItemWithBook is used for query with JOIN
type CartItemWithBook struct {
	CartItem
	BookTitle      string          `db:"book_title"`
	BookSlug       string          `db:"book_slug"`
	BookCoverURL   *string         `db:"book_cover_url"`
	BookAuthor     string          `db:"book_author"`
	CurrentPrice   decimal.Decimal `db:"current_price"`
	CompareAtPrice decimal.Decimal `db:"compare_at_price"`
	IsActive       bool            `db:"is_active"`
	TotalStock     int             `db:"total_stock"`
	CategoryName   *string         `db:"category_name"`
	CategoryID     *uuid.UUID      `db:"category_id"`
}

// ToResponse converts Cart to CartResponse
func (c *Cart) ToResponse(items []CartItemResponse) *CartResponse {
	return &CartResponse{
		ID:             c.ID,
		UserID:         c.UserID,
		SessionID:      c.SessionID,
		Items:          items,
		ItemsCount:     c.ItemsCount,
		Subtotal:       c.Subtotal,
		ExpiresAt:      c.ExpiresAt,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		PromoCode:      c.PromoCode,
		DiscountAmount: &c.Discount,
		Total:          &c.Total,
		PromoMetadata:  c.PromoMetadata,
	}
}

// ToItemResponse converts CartItemWithBook to CartItemResponse
func (ci *CartItemWithBook) ToItemResponse() *CartItemResponse {
	subtotal := ci.Price.Mul(decimal.NewFromInt(int64(ci.Quantity)))
	availableStock := ci.TotalStock
	if availableStock < 0 {
		availableStock = 0
	}

	return &CartItemResponse{
		ID:             ci.ID,
		BookID:         ci.BookID,
		Quantity:       ci.Quantity,
		Price:          ci.Price,
		Subtotal:       subtotal,
		BookTitle:      ci.BookTitle,
		BookSlug:       ci.BookSlug,
		BookCoverURL:   ci.BookCoverURL,
		CompareAtPrice: ci.CompareAtPrice,
		BookAuthor:     ci.BookAuthor,
		CurrentPrice:   ci.CurrentPrice,
		IsAvailable:    ci.IsActive && ci.TotalStock > 0,
		AvailableStock: availableStock,
		CreatedAt:      ci.CreatedAt,
		UpdatedAt:      ci.UpdatedAt,
		CategoryName:   ci.CategoryName,
		CategoryID:     ci.CategoryID,
	}
}

// IsExpired checks if cart has expired
func (c *Cart) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsAuthenticated checks if cart belongs to authenticated user
func (c *Cart) IsAuthenticated() bool {
	return c.UserID != nil
}

// IsGuest checks if cart is for guest user
func (c *Cart) IsGuest() bool {
	return c.SessionID != nil
}

// ExtendExpiration extends cart expiration by 30 days (for authenticated users)
func (c *Cart) ExtendExpiration() {
	c.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
}

// Validate validates cart data
func (c *Cart) Validate() error {
	// Ensure either UserID or SessionID is set, but not both
	if (c.UserID == nil && c.SessionID == nil) || (c.UserID != nil && c.SessionID != nil) {
		return ErrInvalidCart
	}

	// Ensure items_count and subtotal are non-negative
	if c.ItemsCount < 0 {
		return ErrInvalidItemsCount
	}

	if c.Subtotal.LessThan(decimal.Zero) {
		return ErrInvalidSubtotal
	}

	return nil
}

// ValidateCartItem validates cart item data
func (ci *CartItem) Validate() error {
	if ci.Quantity <= 0 {
		return ErrInvalidQuantity
	}

	if ci.Quantity > 100 {
		return ErrQuantityTooHigh
	}

	if ci.Price.LessThan(decimal.Zero) {
		return ErrInvalidPrice
	}

	return nil
}

// CalculateSubtotal calculates item subtotal
func (ci *CartItem) CalculateSubtotal() decimal.Decimal {
	return ci.Price.Mul(decimal.NewFromInt(int64(ci.Quantity)))
}

// HasPriceChanged checks if current book price differs from snapshot price
func (cir *CartItemResponse) HasPriceChanged() bool {
	return !cir.Price.Equal(cir.CurrentPrice)
}

// PriceChangeAmount returns the difference between current and snapshot price
func (cir *CartItemResponse) PriceChangeAmount() decimal.Decimal {
	return cir.CurrentPrice.Sub(cir.Price)
}

// IsStockSufficient checks if available stock is sufficient for quantity
func (cir *CartItemResponse) IsStockSufficient() bool {
	return cir.IsAvailable && cir.AvailableStock >= cir.Quantity
}

// Custom errors for cart operations
var (
	ErrInvalidCart       = errors.New("either user_id or session_id must be set, but not both")
	ErrInvalidItemsCount = errors.New("items_count must be >= 0")
	ErrInvalidSubtotal   = errors.New("subtotal must be >= 0")
	ErrInvalidQuantity   = errors.New("quantity must be > 0")
	ErrQuantityTooHigh   = errors.New("quantity cannot exceed 100")
	ErrInvalidPrice      = errors.New("price must be >= 0")
	ErrCartExpired       = errors.New("cart has expired")
	ErrCartNotFound      = errors.New("cart not found")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrInsufficientStock = errors.New("insufficient stock available")
	ErrBookNotAvailable  = errors.New("book is not available")
)
