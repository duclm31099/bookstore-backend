package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// domains/cart/model.go

// ValidateCartRequest (currently no params needed)
type ValidateCartRequest struct{}

// CartValidationResult represents validation result
type CartValidationResult struct {
	IsValid         bool                    `json:"is_valid"`
	CartStatus      string                  `json:"cart_status"` // "valid", "warning", "error"
	Errors          []CartValidationError   `json:"errors"`
	Warnings        []CartValidationWarning `json:"warnings"`
	ItemValidations []ItemValidation        `json:"item_validations"`
	TotalValue      decimal.Decimal         `json:"total_value"`
	EstimatedTotal  decimal.Decimal         `json:"estimated_total"`
}

// CartValidationError represents validation error
type CartValidationError struct {
	Code     string `json:"code"` // "CART_EXPIRED", "ITEM_OUT_OF_STOCK", etc
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning"
}

// CartValidationWarning represents warning
type CartValidationWarning struct {
	Code    string                 `json:"code"` // "PRICE_CHANGED", "LOW_STOCK", etc
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ItemValidation represents individual item validation
type ItemValidation struct {
	ItemID           uuid.UUID       `json:"item_id"`
	BookID           uuid.UUID       `json:"book_id"`
	BookTitle        string          `json:"book_title"`
	SnapshotPrice    decimal.Decimal `json:"snapshot_price"`    // Price when added
	CurrentPrice     decimal.Decimal `json:"current_price"`     // Current price
	SnapshotQuantity int             `json:"snapshot_quantity"` // Qty in cart
	AvailableStock   int             `json:"available_stock"`   // Stock now
	IsAvailable      bool            `json:"is_available"`
	PriceMatch       bool            `json:"price_match"` // snapshot == current
	StockSufficient  bool            `json:"stock_sufficient"`
	Warnings         []string        `json:"warnings,omitempty"`
}

// ApplyPromoRequest represents request to apply promo code
type ApplyPromoRequest struct {
	PromoCode string `json:"promo_code" binding:"required,min=3,max=50"`
}

// ApplyPromoResponse represents promo application response
type ApplyPromoResponse struct {
	Applied          bool            `json:"applied"`
	PromoCode        string          `json:"promo_code"`
	PromoDescription string          `json:"promo_description"`
	DiscountType     string          `json:"discount_type"` // "percent" or "fixed"
	DiscountValue    decimal.Decimal `json:"discount_value"`
	DiscountAmount   decimal.Decimal `json:"discount_amount"` // Actual discount in VND
	OriginalSubtotal decimal.Decimal `json:"original_subtotal"`
	DiscountedTotal  decimal.Decimal `json:"discounted_total"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
	AppliedAt        time.Time       `json:"applied_at"`
}

// Cart promo fields (add to Cart model)
type CartPromo struct {
	PromoCode *string          `json:"promo_code,omitempty" db:"promo_code"`
	Discount  *decimal.Decimal `json:"discount,omitempty" db:"discount"` // Amount discounted
	Total     *decimal.Decimal `json:"total,omitempty" db:"total"`       // After discount
}

// domains/cart/model.go

// domains/cart/model.go

// ===================================
// CHECKOUT REQUEST - COMPREHENSIVE
// ===================================

type CheckoutRequest struct {
	// Shipping & Billing
	ShippingAddressID uuid.UUID  `json:"shipping_address_id" binding:"required" validate:"required"`
	BillingAddressID  *uuid.UUID `json:"billing_address_id,omitempty"` // NULL = same as shipping

	// Payment
	PaymentMethod  string          `json:"payment_method" binding:"required,oneof=credit_card bank_transfer cash_on_delivery e_wallet" validate:"required,oneof=credit_card bank_transfer cash_on_delivery e_wallet"`
	PaymentDetails *PaymentDetails `json:"payment_details,omitempty"` // Card info, bank account, etc

	// Delivery
	ShippingMethod string     `json:"shipping_method" binding:"required,oneof=standard express overnight" validate:"required"`
	DeliveryDate   *time.Time `json:"delivery_date,omitempty"` // Requested delivery date

	// Additional
	PromoCode     *string `json:"promo_code,omitempty"` // Re-validate promo
	CustomerNotes *string `json:"customer_notes,omitempty" validate:"max=500"`

	// Internal use (set by system)
	UserAgent string `json:"-"` // Track device type
	IPAddress string `json:"-"` // Track location
}

// PaymentDetails represents payment method details
type PaymentDetails struct {
	CardToken       *string `json:"card_token,omitempty"`       // For credit card (tokenized)
	BankCode        *string `json:"bank_code,omitempty"`        // For bank transfer
	EWalletProvider *string `json:"ewallet_provider,omitempty"` // For e-wallet (Momo, ZaloPay, etc)
}

// ===================================
// CHECKOUT PHASES & RESPONSES
// ===================================

// CheckoutPhaseResult represents each phase result
type CheckoutPhaseResult struct {
	Phase     string            `json:"phase"`  // "validation", "reservation", "order_creation", etc
	Status    string            `json:"status"` // "success", "failed", "warning"
	Message   string            `json:"message"`
	Errors    []CheckoutError   `json:"errors,omitempty"`
	Warnings  []CheckoutWarning `json:"warnings,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// CheckoutError represents checkout error
type CheckoutError struct {
	Code     string                 `json:"code"` // "INSUFFICIENT_STOCK", "INVALID_ADDRESS", etc
	Message  string                 `json:"message"`
	Severity string                 `json:"severity"` // "error", "critical"
	Details  map[string]interface{} `json:"details,omitempty"`
	Field    *string                `json:"field,omitempty"` // Which field caused error
}

// CheckoutWarning represents checkout warning
type CheckoutWarning struct {
	Code    string                 `json:"code"` // "PRICE_CHANGED", "OUT_OF_STOCK_PARTIAL", etc
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Action  *string                `json:"action,omitempty"` // Suggested action
}

// ===================================
// CHECKOUT RESPONSE - COMPREHENSIVE
// ===================================

type CheckoutResponse struct {
	// Success indicator
	Success bool   `json:"success"`
	Status  string `json:"status"` // "pending", "completed", "failed", "cancelled"

	// Order info
	OrderID       uuid.UUID `json:"order_id,omitempty"`
	OrderNumber   string    `json:"order_number,omitempty"`   // Human-readable: ORD-2025-11-06-001
	ReferenceCode string    `json:"reference_code,omitempty"` // For customer support

	// Cart info (before checkout)
	CartSummary CartCheckoutSummary `json:"cart_summary"`

	// Order info (after checkout)
	OrderSummary *OrderCheckoutSummary `json:"order_summary,omitempty"`

	// Pricing breakdown
	PricingBreakdown PricingBreakdown `json:"pricing_breakdown"`

	// Items verification
	ItemsProcessed []ItemCheckoutResult `json:"items_processed,omitempty"`

	// Phase results (for troubleshooting)
	Phases []CheckoutPhaseResult `json:"phases,omitempty"`

	// Errors & warnings
	Errors   []CheckoutError   `json:"errors,omitempty"`
	Warnings []CheckoutWarning `json:"warnings,omitempty"`

	// Payment info
	PaymentInfo *PaymentCheckoutInfo `json:"payment_info,omitempty"`

	// Next steps
	NextActions   []string               `json:"next_actions"`
	WarehouseInfo *WarehouseCheckoutInfo `json:"warehouse_info,omitempty"`
	// Timestamps
	InitiatedAt time.Time  `json:"initiated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"` // If pending payment
}
type WarehouseCheckoutInfo struct {
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseName     string    `json:"warehouse_name"`
	DistanceKM        float64   `json:"distance_km"`
	EstimatedDelivery string    `json:"estimated_delivery"` // "1-2 days"
}

// CartCheckoutSummary represents cart state before checkout
type CartCheckoutSummary struct {
	CartID       uuid.UUID       `json:"cart_id"`
	ItemCount    int             `json:"item_count"`
	Subtotal     decimal.Decimal `json:"subtotal"`
	PromoCode    *string         `json:"promo_code,omitempty"`
	Discount     decimal.Decimal `json:"discount"`
	EstimatedTax decimal.Decimal `json:"estimated_tax"`
	ShippingCost decimal.Decimal `json:"shipping_cost"`
	Total        decimal.Decimal `json:"total"`
}

// OrderCheckoutSummary represents order created
type OrderCheckoutSummary struct {
	OrderID     uuid.UUID       `json:"order_id"`
	OrderNumber string          `json:"order_number"`
	Status      string          `json:"status"` // "pending", "confirmed", etc
	TotalAmount decimal.Decimal `json:"total_amount"`
	ItemCount   int             `json:"item_count"`
	CreatedAt   time.Time       `json:"created_at"`
}

// PricingBreakdown shows how total was calculated
type PricingBreakdown struct {
	// Base
	Subtotal decimal.Decimal `json:"subtotal"` // Sum of item prices

	// Deductions
	PromoDiscount  decimal.Decimal `json:"promo_discount,omitempty"`  // From promo code
	VolumeDiscount decimal.Decimal `json:"volume_discount,omitempty"` // Bulk discount
	ManualDiscount decimal.Decimal `json:"manual_discount,omitempty"` // Admin discount

	// Additions
	Tax       decimal.Decimal `json:"tax"`                 // VAT (10%)
	Shipping  decimal.Decimal `json:"shipping"`            // Delivery fee
	Insurance decimal.Decimal `json:"insurance,omitempty"` // Optional

	// Final
	Total decimal.Decimal `json:"total"` // Final amount to pay

	// Additional info
	Currency string          `json:"currency"` // "VND"
	TaxRate  decimal.Decimal `json:"tax_rate"` // e.g., 0.10 for 10%
}

// ItemCheckoutResult represents each item result
type ItemCheckoutResult struct {
	ItemID               uuid.UUID       `json:"item_id"`
	BookID               uuid.UUID       `json:"book_id"`
	BookTitle            string          `json:"book_title"`
	QuantityRequested    int             `json:"quantity_requested"`
	QuantityReserved     int             `json:"quantity_reserved"` // May differ if partial
	PriceAtCheckout      decimal.Decimal `json:"price_at_checkout"` // Snapshot
	CurrentPrice         decimal.Decimal `json:"current_price"`
	PriceChanged         bool            `json:"price_changed"`
	ItemTotal            decimal.Decimal `json:"item_total"` // qty * price
	InventoryReserved    bool            `json:"inventory_reserved"`
	Status               string          `json:"status"` // "reserved", "partial", "unavailable"
	Warnings             []string        `json:"warnings,omitempty"`
	WarehouseID          *uuid.UUID      `json:"warehouse_id,omitempty"`
	WarehouseName        string          `json:"warehouse_name,omitempty"`
	ReservationExpiresAt *time.Time      `json:"reservation_expires_at,omitempty"` // 15
}

// PaymentCheckoutInfo contains payment processing info
type PaymentCheckoutInfo struct {
	PaymentMethod  string     `json:"payment_method"`
	TransactionID  *string    `json:"transaction_id,omitempty"`
	Status         string     `json:"status"` // "pending", "authorized", "failed"
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`      // Payment deadline
	PaymentGateway string     `json:"payment_gateway,omitempty"` // Stripe, VNPay, etc
	RedirectURL    *string    `json:"redirect_url,omitempty"`    // URL to complete payment
}
