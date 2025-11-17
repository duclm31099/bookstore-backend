package model

import (
	address "bookstore-backend/internal/domains/address/model"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Cart represents shopping cart for authenticated and anonymous users
type Cart struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	UserID     *uuid.UUID      `json:"user_id" db:"user_id"`
	SessionID  *string         `json:"session_id" db:"session_id"`
	ItemsCount int             `json:"items_count" db:"items_count"`
	Subtotal   decimal.Decimal `json:"subtotal" db:"subtotal"`
	Version    int             `json:"version" db:"version"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
	ExpiresAt  time.Time       `json:"expires_at" db:"expires_at"`

	// Promotion fields
	PromoCode     *string                `json:"promo_code" db:"promo_code"`
	Discount      decimal.Decimal        `json:"discount" db:"discount"` // ✅ Not pointer
	Total         decimal.Decimal        `json:"total" db:"total"`
	PromoMetadata map[string]interface{} `json:"promo_metadata" db:"promo_metadata"` // ✅ JSONB
}

// IsExpired checks if cart has expired

// HasPromo checks if cart has an active promo code
func (c *Cart) HasPromo() bool {
	return c.PromoCode != nil && *c.PromoCode != ""
}

// CartItem represents items in shopping cart
type CartItem struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	CartID    uuid.UUID       `json:"cart_id" db:"cart_id"`
	BookID    uuid.UUID       `json:"book_id" db:"book_id"`
	Quantity  int             `json:"quantity" db:"quantity"`
	Price     decimal.Decimal `json:"price" db:"price"` // Snapshot price at time of adding
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// ReservedItem tracks inventory reservation for rollback
type ReservedItem struct {
	WarehouseID   uuid.UUID
	BookID        uuid.UUID
	BookTitle     string
	Quantity      int
	ReservationID uuid.UUID
	ExpiresAt     time.Time
}

// CheckoutValidationResult holds all validation results
type CheckoutValidationResult struct {
	Cart         *Cart
	Items        []CartItemWithBook
	Validation   *CartValidationResult
	ShippingAddr *address.AddressResponse
	BillingAddr  *address.AddressResponse
	WarehouseID  *uuid.UUID
}
