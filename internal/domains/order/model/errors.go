package model

import "errors"

// =====================================================
// CUSTOM ERROR CODES
// =====================================================
const (
	ErrCodeOrderNotFound          = "ORD001"
	ErrCodeOrderCannotCancel      = "ORD002"
	ErrCodeVersionMismatch        = "ORD003"
	ErrCodeInsufficientStock      = "ORD004"
	ErrCodePromoInvalid           = "ORD005"
	ErrCodePromoExpired           = "ORD006"
	ErrCodePromoUsageLimitReached = "ORD007"
	ErrCodeMinOrderAmount         = "ORD008"
	ErrCodePaymentTimeout         = "ORD009"
	ErrCodeInvalidWarehouse       = "ORD010"
	ErrCodeInvalidAddress         = "ORD011"
	ErrCodeCartEmpty              = "ORD012"
	ErrCodeInvalidPaymentMethod   = "ORD013"
	ErrCodeUnauthorized           = "ORD014"
	ErrCodeInvalidStatus          = "ORD015"
	ErrCodePromoMinAmount         = "ORD016"
	ErrCodeInvalidOrder           = "ORD017"
)

// =====================================================
// ERROR DEFINITIONS
// =====================================================
var (
	ErrOrderNotFound          = errors.New("order not found")
	ErrOrderCannotCancel      = errors.New("order cannot be cancelled")
	ErrVersionMismatch        = errors.New("version mismatch - concurrent modification detected")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrPromoInvalid           = errors.New("invalid promotion code")
	ErrPromoExpired           = errors.New("promotion code expired")
	ErrPromoUsageLimitReached = errors.New("promotion usage limit reached")
	ErrMinOrderAmount         = errors.New("order amount below minimum")
	ErrPaymentTimeout         = errors.New("payment timeout")
	ErrInvalidWarehouse       = errors.New("invalid warehouse")
	ErrInvalidAddress         = errors.New("invalid address")
	ErrCartEmpty              = errors.New("cart is empty")
	ErrInvalidPaymentMethod   = errors.New("invalid payment method")
	ErrUnauthorized           = errors.New("unauthorized access")
	ErrInvalidStatus          = errors.New("invalid order status")
	ErrPromoMinAmount         = errors.New("order amount below promotion minimum")
)

// =====================================================
// CUSTOM ERROR TYPE
// =====================================================
type OrderError struct {
	Code    string
	Message string
	Err     error
}

func (e *OrderError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *OrderError) Unwrap() error {
	return e.Err
}

// NewOrderError creates a new OrderError
func NewOrderError(code, message string, err error) *OrderError {
	return &OrderError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
