package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentMethod represents valid payment methods
type PaymentMethod string

const (
	PaymentMethodCOD          PaymentMethod = "cod"
	PaymentMethodVNPay        PaymentMethod = "vnpay"
	PaymentMethodMomo         PaymentMethod = "momo"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
)

func (pm PaymentMethod) IsValid() bool {
	switch pm {
	case PaymentMethodCOD, PaymentMethodVNPay, PaymentMethodMomo, PaymentMethodBankTransfer:
		return true
	}
	return false
}

func (pm PaymentMethod) String() string {
	return string(pm)
}

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

func (ps PaymentStatus) IsValid() bool {
	switch ps {
	case PaymentStatusPending, PaymentStatusPaid, PaymentStatusFailed, PaymentStatusRefunded:
		return true
	}
	return false
}

func (ps PaymentStatus) String() string {
	return string(ps)
}

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipping   OrderStatus = "shipping"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusReturned   OrderStatus = "returned"
)

func (os OrderStatus) IsValid() bool {
	switch os {
	case OrderStatusPending, OrderStatusConfirmed, OrderStatusProcessing,
		OrderStatusShipping, OrderStatusDelivered, OrderStatusCancelled, OrderStatusReturned:
		return true
	}
	return false
}

func (os OrderStatus) String() string {
	return string(os)
}

// PaymentDetails stores payment gateway response data
type PaymentDetails map[string]interface{}

// Value implements driver.Valuer for JSONB
func (pd PaymentDetails) Value() (driver.Value, error) {
	if pd == nil {
		return nil, nil
	}
	return json.Marshal(pd)
}

// Scan implements sql.Scanner for JSONB
func (pd *PaymentDetails) Scan(value interface{}) error {
	if value == nil {
		*pd = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return ErrInvalidPaymentDetails
	}

	return json.Unmarshal(bytes, pd)
}

// Order represents customer orders with payment and delivery tracking
type Order struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	OrderNumber string     `json:"order_number" db:"order_number"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	AddressID   uuid.UUID  `json:"address_id" db:"address_id"`
	PromotionID *uuid.UUID `json:"promotion_id" db:"promotion_id"`

	// Pricing
	Subtotal       decimal.Decimal `json:"subtotal" db:"subtotal"`
	ShippingFee    decimal.Decimal `json:"shipping_fee" db:"shipping_fee"`
	DiscountAmount decimal.Decimal `json:"discount_amount" db:"discount_amount"`
	Total          decimal.Decimal `json:"total" db:"total"`

	// Payment
	PaymentMethod  string         `json:"payment_method" db:"payment_method"`
	PaymentStatus  string         `json:"payment_status" db:"payment_status"`
	PaymentDetails PaymentDetails `json:"payment_details" db:"payment_details"`
	PaidAt         *time.Time     `json:"paid_at" db:"paid_at"`

	// Status
	Status string `json:"status" db:"status"`

	// Shipping
	TrackingNumber      *string    `json:"tracking_number" db:"tracking_number"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at" db:"estimated_delivery_at"`
	DeliveredAt         *time.Time `json:"delivered_at" db:"delivered_at"`

	// Notes
	CustomerNote       *string `json:"customer_note" db:"customer_note"`
	AdminNote          *string `json:"admin_note" db:"admin_note"`
	CancellationReason *string `json:"cancellation_reason" db:"cancellation_reason"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CancelledAt *time.Time `json:"cancelled_at" db:"cancelled_at"`
}

// OrderItem represents order line items with snapshot prices
type OrderItem struct {
	ID      uuid.UUID `json:"id" db:"id"`
	OrderID uuid.UUID `json:"order_id" db:"order_id"`
	BookID  uuid.UUID `json:"book_id" db:"book_id"`

	// Snapshot data
	BookTitle    string  `json:"book_title" db:"book_title"`
	BookSlug     string  `json:"book_slug" db:"book_slug"`
	BookCoverURL *string `json:"book_cover_url" db:"book_cover_url"`
	AuthorName   *string `json:"author_name" db:"author_name"`

	Quantity int             `json:"quantity" db:"quantity"`
	Price    decimal.Decimal `json:"price" db:"price"`
	Subtotal decimal.Decimal `json:"subtotal" db:"subtotal"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// OrderStatusHistory tracks order status changes
type OrderStatusHistory struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	OrderID    uuid.UUID  `json:"order_id" db:"order_id"`
	FromStatus *string    `json:"from_status" db:"from_status"`
	ToStatus   string     `json:"to_status" db:"to_status"`
	ChangedBy  *uuid.UUID `json:"changed_by" db:"changed_by"`
	Notes      *string    `json:"notes" db:"notes"`
	ChangedAt  time.Time  `json:"changed_at" db:"changed_at"`
}

// CreateOrderRequest represents checkout request
type CreateOrderRequest struct {
	AddressID     uuid.UUID `json:"address_id" validate:"required"`
	PaymentMethod string    `json:"payment_method" validate:"required,oneof=cod vnpay momo bank_transfer"`
	PromoCode     *string   `json:"promo_code" validate:"omitempty,min=3,max=50"`
	CustomerNote  *string   `json:"customer_note" validate:"omitempty,max=500"`
}

// UpdateOrderStatusRequest represents request to update order status
type UpdateOrderStatusRequest struct {
	Status            string     `json:"status" validate:"required,oneof=confirmed processing shipping delivered cancelled returned"`
	TrackingNumber    *string    `json:"tracking_number" validate:"omitempty,max=100"`
	EstimatedDelivery *time.Time `json:"estimated_delivery" validate:"omitempty"`
	AdminNote         *string    `json:"admin_note" validate:"omitempty,max=1000"`
}

// CancelOrderRequest represents request to cancel order
type CancelOrderRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

// OrderResponse represents order response with full details
type OrderResponse struct {
	ID          uuid.UUID `json:"id"`
	OrderNumber string    `json:"order_number"`
	UserID      uuid.UUID `json:"user_id"`

	// Address details
	ShippingAddress *AddressResponse `json:"shipping_address,omitempty"`

	// Promotion details
	Promotion *PromotionListResponse `json:"promotion,omitempty"`

	// Items
	Items []OrderItemResponse `json:"items"`

	// Pricing
	Subtotal       decimal.Decimal `json:"subtotal"`
	ShippingFee    decimal.Decimal `json:"shipping_fee"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	Total          decimal.Decimal `json:"total"`

	// Payment
	PaymentMethod  string         `json:"payment_method"`
	PaymentStatus  string         `json:"payment_status"`
	PaymentDetails PaymentDetails `json:"payment_details,omitempty"`
	PaidAt         *time.Time     `json:"paid_at,omitempty"`

	// Status
	Status        string                       `json:"status"`
	StatusHistory []OrderStatusHistoryResponse `json:"status_history,omitempty"`

	// Shipping
	TrackingNumber      *string    `json:"tracking_number,omitempty"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at,omitempty"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`

	// Notes
	CustomerNote       *string `json:"customer_note,omitempty"`
	AdminNote          *string `json:"admin_note,omitempty"`
	CancellationReason *string `json:"cancellation_reason,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
}

// OrderListResponse represents simplified order for list views
type OrderListResponse struct {
	ID          uuid.UUID       `json:"id"`
	OrderNumber string          `json:"order_number"`
	Total       decimal.Decimal `json:"total"`
	Status      string          `json:"status"`
	ItemCount   int             `json:"item_count"`
	CreatedAt   time.Time       `json:"created_at"`
}

// OrderItemResponse represents order item with details
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

// OrderStatusHistoryResponse represents status history
type OrderStatusHistoryResponse struct {
	ID         uuid.UUID  `json:"id"`
	FromStatus *string    `json:"from_status,omitempty"`
	ToStatus   string     `json:"to_status"`
	ChangedBy  *uuid.UUID `json:"changed_by,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	ChangedAt  time.Time  `json:"changed_at"`
}

// OrderSearchQuery represents search/filter parameters
type OrderSearchQuery struct {
	UserID        *uuid.UUID `json:"user_id" form:"user_id" validate:"omitempty"`
	Status        *string    `json:"status" form:"status" validate:"omitempty,oneof=pending confirmed processing shipping delivered cancelled returned"`
	PaymentStatus *string    `json:"payment_status" form:"payment_status" validate:"omitempty,oneof=pending paid failed refunded"`
	DateFrom      *time.Time `json:"date_from" form:"date_from" validate:"omitempty"`
	DateTo        *time.Time `json:"date_to" form:"date_to" validate:"omitempty"`

	// Pagination
	Page  int `json:"page" form:"page" validate:"omitempty,gte=1"`
	Limit int `json:"limit" form:"limit" validate:"omitempty,gte=1,lte=100"`

	// Sorting
	Sort string `json:"sort" form:"sort" validate:"omitempty,oneof=created_at_desc created_at_asc total_desc total_asc"`
}

// ToResponse converts Order to OrderResponse
func (o *Order) ToResponse(items []OrderItemResponse) *OrderResponse {
	return &OrderResponse{
		ID:                  o.ID,
		OrderNumber:         o.OrderNumber,
		UserID:              o.UserID,
		Items:               items,
		Subtotal:            o.Subtotal,
		ShippingFee:         o.ShippingFee,
		DiscountAmount:      o.DiscountAmount,
		Total:               o.Total,
		PaymentMethod:       o.PaymentMethod,
		PaymentStatus:       o.PaymentStatus,
		PaymentDetails:      o.PaymentDetails,
		PaidAt:              o.PaidAt,
		Status:              o.Status,
		TrackingNumber:      o.TrackingNumber,
		EstimatedDeliveryAt: o.EstimatedDeliveryAt,
		DeliveredAt:         o.DeliveredAt,
		CustomerNote:        o.CustomerNote,
		AdminNote:           o.AdminNote,
		CancellationReason:  o.CancellationReason,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
		CancelledAt:         o.CancelledAt,
	}
}

type AddressResponse struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`

	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`

	Province string `json:"province"`
	District string `json:"district"`
	Ward     string `json:"ward"`
	Street   string `json:"street"`

	AddressType *string `json:"address_type,omitempty"`
	IsDefault   bool    `json:"is_default"`
	Notes       *string `json:"notes,omitempty"`
}
type PromotionListResponse struct {
	ID            uuid.UUID       `json:"id"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	DiscountType  string          `json:"discount_type"`
	DiscountValue decimal.Decimal `json:"discount_value"`
	CurrentUses   int             `json:"current_uses"`
	MaxUses       *int            `json:"max_uses,omitempty"`
	StartsAt      time.Time       `json:"starts_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	IsActive      bool            `json:"is_active"`
}

// ToItemResponse converts OrderItem to OrderItemResponse
func (oi *OrderItem) ToItemResponse() *OrderItemResponse {
	return &OrderItemResponse{
		ID:           oi.ID,
		BookID:       oi.BookID,
		BookTitle:    oi.BookTitle,
		BookSlug:     oi.BookSlug,
		BookCoverURL: oi.BookCoverURL,
		AuthorName:   oi.AuthorName,
		Quantity:     oi.Quantity,
		Price:        oi.Price,
		Subtotal:     oi.Subtotal,
	}
}

// ToHistoryResponse converts OrderStatusHistory to response
func (osh *OrderStatusHistory) ToHistoryResponse() *OrderStatusHistoryResponse {
	return &OrderStatusHistoryResponse{
		ID:         osh.ID,
		FromStatus: osh.FromStatus,
		ToStatus:   osh.ToStatus,
		ChangedBy:  osh.ChangedBy,
		Notes:      osh.Notes,
		ChangedAt:  osh.ChangedAt,
	}
}

// IsPaid checks if order is paid
func (o *Order) IsPaid() bool {
	return o.PaymentStatus == string(PaymentStatusPaid)
}

// IsPending checks if order is pending
func (o *Order) IsPending() bool {
	return o.Status == string(OrderStatusPending)
}

// IsCancellable checks if order can be cancelled
func (o *Order) IsCancellable() bool {
	// Can cancel before processing starts
	return o.Status == string(OrderStatusPending) ||
		o.Status == string(OrderStatusConfirmed)
}

// IsDelivered checks if order is delivered
func (o *Order) IsDelivered() bool {
	return o.Status == string(OrderStatusDelivered)
}

// IsCancelled checks if order is cancelled
func (o *Order) IsCancelled() bool {
	return o.Status == string(OrderStatusCancelled)
}

// CanTransitionTo checks if order can transition to new status
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	currentStatus := OrderStatus(o.Status)

	// Define valid transitions
	validTransitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending: {
			OrderStatusConfirmed,
			OrderStatusCancelled,
		},
		OrderStatusConfirmed: {
			OrderStatusProcessing,
			OrderStatusCancelled,
		},
		OrderStatusProcessing: {
			OrderStatusShipping,
			OrderStatusCancelled,
		},
		OrderStatusShipping: {
			OrderStatusDelivered,
		},
		OrderStatusDelivered: {
			OrderStatusReturned,
		},
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// CalculateTotal calculates order total
func (o *Order) CalculateTotal() decimal.Decimal {
	return o.Subtotal.Add(o.ShippingFee).Sub(o.DiscountAmount)
}

// Validate validates order data
func (o *Order) Validate() error {
	// Validate payment method
	paymentMethod := PaymentMethod(o.PaymentMethod)
	if !paymentMethod.IsValid() {
		return ErrInvalidPaymentMethod
	}

	// Validate payment status
	paymentStatus := PaymentStatus(o.PaymentStatus)
	if !paymentStatus.IsValid() {
		return ErrInvalidPaymentStatus
	}

	// Validate order status
	orderStatus := OrderStatus(o.Status)
	if !orderStatus.IsValid() {
		return ErrInvalidOrderStatus
	}

	// Validate amounts
	if o.Subtotal.LessThan(decimal.Zero) {
		return ErrInvalidSubtotal
	}

	if o.ShippingFee.LessThan(decimal.Zero) {
		return ErrInvalidShippingFee
	}

	if o.DiscountAmount.LessThan(decimal.Zero) {
		return ErrInvalidDiscountAmount
	}

	if o.Total.LessThan(decimal.Zero) {
		return ErrInvalidTotal
	}

	// Validate total calculation
	calculatedTotal := o.CalculateTotal()
	if !o.Total.Equal(calculatedTotal) {
		return ErrTotalMismatch
	}

	return nil
}

// Custom errors for order operations
var (
	ErrInvalidPaymentMethod    = errors.New("invalid payment method")
	ErrInvalidPaymentStatus    = errors.New("invalid payment status")
	ErrInvalidOrderStatus      = errors.New("invalid order status")
	ErrInvalidSubtotal         = errors.New("subtotal must be >= 0")
	ErrInvalidShippingFee      = errors.New("shipping fee must be >= 0")
	ErrInvalidDiscountAmount   = errors.New("discount amount must be >= 0")
	ErrInvalidTotal            = errors.New("total must be >= 0")
	ErrTotalMismatch           = errors.New("total does not match calculation")
	ErrInvalidPaymentDetails   = errors.New("invalid payment details format")
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderNotCancellable     = errors.New("order cannot be cancelled at this stage")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrOrderAlreadyPaid        = errors.New("order is already paid")
	ErrOrderNotPaid            = errors.New("order is not paid")
	ErrEmptyCart               = errors.New("cart is empty")
)
