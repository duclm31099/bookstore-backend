package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =====================================================
// ORDER STATUS CONSTANTS
// =====================================================
const (
	OrderStatusPending    = "pending"
	OrderStatusConfirmed  = "confirmed"
	OrderStatusProcessing = "processing"
	OrderStatusShipping   = "shipping"
	OrderStatusDelivered  = "delivered"
	OrderStatusCancelled  = "cancelled"
	OrderStatusReturned   = "returned"
)

// =====================================================
// PAYMENT METHOD CONSTANTS
// =====================================================
const (
	PaymentMethodCOD          = "cod"
	PaymentMethodVNPay        = "vnpay"
	PaymentMethodMomo         = "momo"
	PaymentMethodBankTransfer = "bank_transfer"
)

// =====================================================
// PAYMENT STATUS CONSTANTS
// =====================================================
const (
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

// =====================================================
// BUSINESS CONSTANTS
// =====================================================
const (
	ShippingFee        = 15000 // 15,000 VND
	CODFee             = 15000 // 15,000 VND
	MinimumOrderAmount = 0     // No minimum (set to 0, can be updated later)
	TaxRate            = 0.0   // 0% tax
)

// =====================================================
// ENTITY: Order
// =====================================================
type Order struct {
	ID                  uuid.UUID       `json:"id"`
	OrderNumber         string          `json:"order_number"`
	UserID              uuid.UUID       `json:"user_id"`
	AddressID           uuid.UUID       `json:"address_id"`
	PromotionID         *uuid.UUID      `json:"promotion_id,omitempty"`
	WarehouseID         *uuid.UUID      `json:"warehouse_id,omitempty"`
	Subtotal            decimal.Decimal `json:"subtotal"`
	ShippingFee         decimal.Decimal `json:"shipping_fee"`
	CODFee              decimal.Decimal `json:"cod_fee"`
	DiscountAmount      decimal.Decimal `json:"discount_amount"`
	TaxAmount           decimal.Decimal `json:"tax_amount"`
	Total               decimal.Decimal `json:"total"`
	PaymentMethod       string          `json:"payment_method"`
	PaymentStatus       string          `json:"payment_status"`
	PaymentDetails      *map[string]any `json:"payment_details,omitempty"`
	PaidAt              *time.Time      `json:"paid_at,omitempty"`
	Status              string          `json:"status"`
	TrackingNumber      *string         `json:"tracking_number,omitempty"`
	EstimatedDeliveryAt *time.Time      `json:"estimated_delivery_at,omitempty"`
	DeliveredAt         *time.Time      `json:"delivered_at,omitempty"`
	CustomerNote        *string         `json:"customer_note,omitempty"`
	AdminNote           *string         `json:"admin_note,omitempty"`
	CancellationReason  *string         `json:"cancellation_reason,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	CancelledAt         *time.Time      `json:"cancelled_at,omitempty"`
	Version             int             `json:"version"`
}

// CanBeCancelled checks if order can be cancelled by user
// Business rule: Only pending/confirmed orders can be cancelled
func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusConfirmed
}

// RequiresOnlinePayment checks if order requires online payment
func (o *Order) RequiresOnlinePayment() bool {
	return o.PaymentMethod == PaymentMethodVNPay ||
		o.PaymentMethod == PaymentMethodMomo ||
		o.PaymentMethod == PaymentMethodBankTransfer
}

// IsCOD checks if order is cash on delivery
func (o *Order) IsCOD() bool {
	return o.PaymentMethod == PaymentMethodCOD
}

// IsPaymentCompleted checks if payment is completed
func (o *Order) IsPaymentCompleted() bool {
	return o.PaymentStatus == PaymentStatusPaid
}

// CanBeRefunded checks if order can be refunded
func (o *Order) CanBeRefunded() bool {
	return o.IsPaymentCompleted() &&
		(o.Status == OrderStatusCancelled || o.Status == OrderStatusReturned)
}

// =====================================================
// ENTITY: OrderItem
// =====================================================
type OrderItem struct {
	ID           uuid.UUID       `json:"id"`
	OrderID      uuid.UUID       `json:"order_id"`
	BookID       uuid.UUID       `json:"book_id"`
	BookTitle    string          `json:"book_title"`
	BookSlug     string          `json:"book_slug"`
	BookCoverURL *string         `json:"book_cover_url,omitempty"`
	AuthorName   *string         `json:"author_name,omitempty"`
	Quantity     int             `json:"quantity"`
	Price        decimal.Decimal `json:"price"`
	Subtotal     decimal.Decimal `json:"subtotal"`
	CreatedAt    time.Time       `json:"created_at"`
	WarehouseID  *uuid.UUID      `json:"warehouse_id"`
}

// CalculateSubtotal calculates item subtotal
func (oi *OrderItem) CalculateSubtotal() decimal.Decimal {
	return oi.Price.Mul(decimal.NewFromInt(int64(oi.Quantity)))
}

// =====================================================
// ENTITY: OrderStatusHistory
// =====================================================
type OrderStatusHistory struct {
	ID         uuid.UUID  `json:"id"`
	OrderID    uuid.UUID  `json:"order_id"`
	FromStatus *string    `json:"from_status,omitempty"`
	ToStatus   string     `json:"to_status"`
	ChangedBy  *uuid.UUID `json:"changed_by,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	ChangedAt  time.Time  `json:"changed_at"`
}

// =====================================================
// WAREHOUSE PROVINCE MAPPING
// =====================================================
var ProvinceWarehouseMap = map[string]string{
	"Hà Nội":          "WH-HN-01",
	"TP. Hồ Chí Minh": "WH-HCM-01",
	"TP.HCM":          "WH-HCM-01",
	"Hồ Chí Minh":     "WH-HCM-01",
	"Đà Nẵng":         "WH-DN-01",
	"Cần Thơ":         "CT-01", // If you have CT warehouse
}

// DefaultWarehouseCode returns HN-01 as default warehouse
const DefaultWarehouseCode = "WH-HN-01"
