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

// ================================================
// MODELS FOR PROMOTION REMOVAL JOB
// ================================================

// CartWithPromoInfo contains cart data with promotion and user details
// WHY THIS STRUCT?
// - Efficient batch querying: Single JOIN query instead of N+1 queries
// - Contains all data needed to decide if promotion should be removed
// - Includes user activity (last_login_at) for smart scheduling logic
type CartWithPromoInfo struct {
	// Cart fields
	CartID        uuid.UUID              `db:"cart_id"`
	UserID        uuid.UUID              `db:"user_id"`
	PromoCode     string                 `db:"promo_code"`
	Discount      decimal.Decimal        `db:"discount"`
	PromoMetadata map[string]interface{} `db:"promo_metadata"` // JSONB

	// User fields for smart scheduling
	// WHY last_login_at? Determines if user is active (< 24h) or inactive (> 24h)
	LastLoginAt *time.Time `db:"last_login_at"`

	// Promotion fields for validation
	// WHY nullable? Promotion might have been deleted from promotions table
	PromotionID *uuid.UUID `db:"promotion_id"`
	ExpiresAt   *time.Time `db:"expires_at"`
	IsActive    *bool      `db:"is_active"`
	MaxUses     *int       `db:"max_uses"`
	CurrentUses *int       `db:"current_uses"`
}

// IsPromotionExpired checks if promotion has expired
// BUSINESS LOGIC: Promotion is considered expired if expires_at < NOW()
func (c *CartWithPromoInfo) IsPromotionExpired() bool {
	if c.ExpiresAt == nil {
		// Promotion deleted from database - consider as expired
		return true
	}
	return c.ExpiresAt.Before(time.Now())
}

// IsPromotionDisabled checks if promotion was disabled by admin
// BUSINESS LOGIC: Admin can disable promotion by setting is_active = false
func (c *CartWithPromoInfo) IsPromotionDisabled() bool {
	if c.IsActive == nil {
		// Promotion deleted from database - consider as disabled
		return true
	}
	return !*c.IsActive
}

// IsPromotionMaxUsesReached checks if global usage limit reached
// BUSINESS LOGIC: Promotion exhausted if current_uses >= max_uses
func (c *CartWithPromoInfo) IsPromotionMaxUsesReached() bool {
	// If max_uses is NULL, there's no limit
	if c.MaxUses == nil {
		return false
	}
	// If current_uses is NULL (shouldn't happen), assume not reached
	if c.CurrentUses == nil {
		return false
	}
	return *c.CurrentUses >= *c.MaxUses
}

// ShouldRemovePromotion determines if promotion should be removed
// Returns: (shouldRemove bool, reason string)
// BUSINESS LOGIC: Remove if ANY condition is true
func (c *CartWithPromoInfo) ShouldRemovePromotion() (bool, string) {
	if c.IsPromotionExpired() {
		return true, "expired"
	}
	if c.IsPromotionDisabled() {
		return true, "disabled"
	}
	if c.IsPromotionMaxUsesReached() {
		return true, "max_uses_reached"
	}
	return false, ""
}

// IsUserActive checks if user is active (last_login < 24h)
// WHY 24 HOURS? Business requirement for dynamic scheduling
// - Active users: Check every 3 hours
// - Inactive users: Check every 24 hours
func (c *CartWithPromoInfo) IsUserActive() bool {
	if c.LastLoginAt == nil {
		// No login record - consider inactive
		return false
	}
	// Active if logged in within last 24 hours
	return c.LastLoginAt.After(time.Now().Add(-24 * time.Hour))
}

// GetLastCheckedAt extracts last_checked_at from promo_metadata
// WHY IN METADATA? Flexible storage without schema changes
// Returns: last check time, or zero time if never checked
func (c *CartWithPromoInfo) GetLastCheckedAt() time.Time {
	if c.PromoMetadata == nil {
		return time.Time{} // Zero time = never checked
	}

	// Try to extract last_checked_at from JSONB
	if lastCheckedStr, ok := c.PromoMetadata["last_checked_at"].(string); ok {
		if lastChecked, err := time.Parse(time.RFC3339, lastCheckedStr); err == nil {
			return lastChecked
		}
	}

	return time.Time{} // Parse failed = never checked
}

// ShouldProcessNow determines if this cart should be processed in current job run
// SMART SCHEDULING LOGIC:
// - Active users (last_login < 24h): Always process
// - Inactive users (last_login > 24h): Only if 24h passed since last check
func (c *CartWithPromoInfo) ShouldProcessNow() bool {
	// Active users: always process
	if c.IsUserActive() {
		return true
	}

	// Inactive users: check if 24h passed since last check
	lastChecked := c.GetLastCheckedAt()
	if lastChecked.IsZero() {
		// Never checked before - process now
		return true
	}

	// Process if 24 hours have passed since last check
	return time.Since(lastChecked) >= 24*time.Hour
}

// PromotionRemovalLog represents a log entry for promotion removal
// WHY SEPARATE STRUCT? Maps to promotion_removal_logs table
type PromotionRemovalLog struct {
	ID             uuid.UUID              `db:"id"`
	CartID         uuid.UUID              `db:"cart_id"`
	UserID         uuid.UUID              `db:"user_id"`
	PromoCode      string                 `db:"promo_code"`
	DiscountAmount decimal.Decimal        `db:"discount_amount"`
	RemovalReason  string                 `db:"removal_reason"`
	PromoMetadata  map[string]interface{} `db:"promo_metadata"`
	RemovedAt      time.Time              `db:"removed_at"`
	Notified       bool                   `db:"notified"`
	CreatedAt      time.Time              `db:"created_at"`
}
