package model

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =====================================================
// CREATE ORDER REQUEST
// =====================================================
type CreateOrderRequest struct {
	AddressID     uuid.UUID         `json:"address_id" binding:"required"`
	PaymentMethod string            `json:"payment_method" binding:"required"`
	PromoCode     *string           `json:"promo_code,omitempty"`
	CustomerNote  *string           `json:"customer_note,omitempty"`
	Items         []CreateOrderItem `json:"items" binding:"omitempty,min=1"`
}

type CreateOrderItem struct {
	BookID   uuid.UUID `json:"book_id" binding:"required"`
	Quantity int       `json:"quantity" binding:"required,min=1"`
}

// Validate validates CreateOrderRequest
func (req CreateOrderRequest) Validate() error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.AddressID, validation.Required, is.UUIDv4),
		validation.Field(&req.PaymentMethod, validation.Required, validation.In(
			PaymentMethodCOD,
			PaymentMethodVNPay,
			PaymentMethodMomo,
			PaymentMethodBankTransfer,
		)),
		// validation.Field(&req.Items, validation.Required, validation.Length(1, 100)),
	)
}

// =====================================================
// CREATE ORDER RESPONSE
// =====================================================
type CreateOrderResponse struct {
	OrderID     uuid.UUID       `json:"order_id"`
	OrderNumber string          `json:"order_number"`
	Total       decimal.Decimal `json:"total"`
	Status      string          `json:"status"`
	PaymentURL  *string         `json:"payment_url,omitempty"` // For VNPay/Momo (will be filled by payment service)
}

// =====================================================
// ORDER DETAIL RESPONSE
// =====================================================
type OrderDetailResponse struct {
	ID                  uuid.UUID             `json:"id"`
	OrderNumber         string                `json:"order_number"`
	Status              string                `json:"status"`
	PaymentMethod       string                `json:"payment_method"`
	PaymentStatus       string                `json:"payment_status"`
	Subtotal            decimal.Decimal       `json:"subtotal"`
	ShippingFee         decimal.Decimal       `json:"shipping_fee"`
	CODFee              decimal.Decimal       `json:"cod_fee"`
	DiscountAmount      decimal.Decimal       `json:"discount_amount"`
	TaxAmount           decimal.Decimal       `json:"tax_amount"`
	Total               decimal.Decimal       `json:"total"`
	Items               []OrderItemResponse   `json:"items"`
	Address             *OrderAddressResponse `json:"address,omitempty"`
	TrackingNumber      *string               `json:"tracking_number,omitempty"`
	EstimatedDeliveryAt *time.Time            `json:"estimated_delivery_at,omitempty"`
	DeliveredAt         *time.Time            `json:"delivered_at,omitempty"`
	CustomerNote        *string               `json:"customer_note,omitempty"`
	AdminNote           *string               `json:"admin_note,omitempty"`
	CancellationReason  *string               `json:"cancellation_reason,omitempty"`
	PaidAt              *time.Time            `json:"paid_at,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
	CancelledAt         *time.Time            `json:"cancelled_at,omitempty"`
	Version             int                   `json:"version"`
}

type OrderItemResponse struct {
	ID           uuid.UUID       `json:"id"`
	BookID       uuid.UUID       `json:"book_id"`
	BookTitle    string          `json:"book_title"`
	BookSlug     string          `json:"book_slug"`
	BookCoverURL *string         `json:"book_cover_url,omitempty"`
	AuthorName   *string         `json:"author_name,omitempty"`
	Quantity     int             `json:"quantity"`
	Price        decimal.Decimal `json:"price"`
	Subtotal     decimal.Decimal `json:"subtotal"`
}

type OrderAddressResponse struct {
	ID           uuid.UUID `json:"id"`
	ReceiverName string    `json:"receiver_name"`
	Phone        string    `json:"phone"`
	Province     string    `json:"province"`
	District     string    `json:"district"`
	Ward         string    `json:"ward"`
	FullAddress  string    `json:"full_address"`
}

// =====================================================
// LIST ORDERS REQUEST
// =====================================================
type ListOrdersRequest struct {
	Status string `form:"status"` // Filter by status (optional)
	Page   int    `form:"page" binding:"min=1"`
	Limit  int    `form:"limit" binding:"min=1,max=100"`
}

// Validate validates ListOrdersRequest
func (req *ListOrdersRequest) Validate() error {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20 // Default
	}

	// Validate status if provided
	if req.Status != "" {
		validStatuses := []interface{}{
			OrderStatusPending,
			OrderStatusConfirmed,
			OrderStatusProcessing,
			OrderStatusShipping,
			OrderStatusDelivered,
			OrderStatusCancelled,
			OrderStatusReturned,
		}
		return validation.Validate(req.Status, validation.In(validStatuses...))
	}

	return nil
}

// =====================================================
// LIST ORDERS RESPONSE
// =====================================================
type ListOrdersResponse struct {
	Orders     []OrderSummaryResponse `json:"orders"`
	Pagination PaginationMeta         `json:"pagination"`
}

type OrderSummaryResponse struct {
	ID            uuid.UUID       `json:"id"`
	OrderNumber   string          `json:"order_number"`
	Status        string          `json:"status"`
	PaymentMethod string          `json:"payment_method"`
	PaymentStatus string          `json:"payment_status"`
	Total         decimal.Decimal `json:"total"`
	ItemsCount    int             `json:"items_count"`
	CreatedAt     time.Time       `json:"created_at"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// =====================================================
// CANCEL ORDER REQUEST
// =====================================================
type CancelOrderRequest struct {
	CancellationReason string `json:"cancellation_reason" binding:"required"`
	Version            int    `json:"version" binding:"required"`
}

// Validate validates CancelOrderRequest
func (req CancelOrderRequest) Validate() error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.CancellationReason, validation.Required, validation.Length(5, 500)),
		validation.Field(&req.Version, validation.Required, validation.Min(0)),
	)
}

// =====================================================
// UPDATE ORDER STATUS REQUEST (Admin)
// =====================================================
type UpdateOrderStatusRequest struct {
	Status         string  `json:"status" binding:"required"`
	Version        int     `json:"version" binding:"required"`
	AdminNote      *string `json:"admin_note,omitempty"`
	TrackingNumber *string `json:"tracking_number,omitempty"` // For shipping status
}

// Validate validates UpdateOrderStatusRequest
func (req UpdateOrderStatusRequest) Validate() error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Status, validation.Required, validation.In(
			OrderStatusConfirmed,
			OrderStatusProcessing,
			OrderStatusShipping,
			OrderStatusDelivered,
			OrderStatusCancelled,
			OrderStatusReturned,
		)),
		validation.Field(&req.Version, validation.Required, validation.Min(0)),
	)
}

// =====================================================
// REORDER REQUEST
// =====================================================
type ReorderRequest struct {
	OrderID   uuid.UUID `json:"order_id" binding:"required"`
	AddressID uuid.UUID `json:"address_id" binding:"required"`
}
