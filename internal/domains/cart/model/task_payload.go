package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ClearCartPayload for clearing cart after successful checkout
type ClearCartPayload struct {
	CartID uuid.UUID `json:"cart_id"`
	UserID uuid.UUID `json:"user_id"`
}

// SendOrderConfirmationPayload for sending order confirmation email
type SendOrderConfirmationPayload struct {
	OrderID           uuid.UUID       `json:"order_id"`
	OrderNumber       string          `json:"order_number"`
	UserID            uuid.UUID       `json:"user_id"`
	UserEmail         string          `json:"user_email"`
	Total             decimal.Decimal `json:"total"`
	PaymentMethod     string          `json:"payment_method"`
	EstimatedDelivery string          `json:"estimated_delivery"`
	ShippingAddressID uuid.UUID       `json:"shipping_address_id"`
	OrderCreatedAt    string          `json:"order_created_at"` // RFC3339 format
}

// AutoReleaseReservationPayload for auto-releasing inventory if payment not completed
type AutoReleaseReservationPayload struct {
	OrderID     uuid.UUID `json:"order_id"`
	OrderNumber string    `json:"order_number"`
	UserID      uuid.UUID `json:"user_id"`
}

// TrackCheckoutPayload for analytics tracking
type TrackCheckoutPayload struct {
	OrderID       uuid.UUID       `json:"order_id"`
	OrderNumber   string          `json:"order_number"`
	UserID        uuid.UUID       `json:"user_id"`
	Total         decimal.Decimal `json:"total"`
	ItemCount     int             `json:"item_count"`
	PaymentMethod string          `json:"payment_method"`
	PromoCode     *string         `json:"promo_code,omitempty"`
	Discount      decimal.Decimal `json:"discount"`
}
