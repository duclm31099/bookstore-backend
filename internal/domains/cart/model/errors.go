package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Checkout Error Codes Reference

const (
	// Authentication
	ErrCheckoutUnauthenticated = "UNAUTHENTICATED"

	// Cart
	ErrCheckoutCartNotFound = "CART_NOT_FOUND"
	ErrCheckoutCartEmpty    = "EMPTY_CART"
	ErrCheckoutCartExpired  = "CART_EXPIRED"

	// Stock
	ErrCheckoutInsufficientStock = "INSUFFICIENT_STOCK"
	ErrCheckoutPartialStock      = "PARTIAL_STOCK"
	ErrCheckoutOutOfStock        = "OUT_OF_STOCK"

	// Price
	ErrCheckoutPriceChanged = "PRICE_CHANGED"

	// Address
	ErrCheckoutInvalidAddress  = "INVALID_ADDRESS"
	ErrCheckoutAddressNotFound = "ADDRESS_NOT_FOUND"

	// Promo
	ErrCheckoutInvalidPromo = "INVALID_PROMO"
	ErrCheckoutPromoExpired = "PROMO_EXPIRED"
	ErrCheckoutPromoUsed    = "PROMO_ALREADY_USED"

	// Payment
	ErrCheckoutInvalidPayment = "INVALID_PAYMENT"
	ErrCheckoutPaymentFailed  = "PAYMENT_FAILED"

	// System
	ErrCheckoutLockFailed        = "LOCK_FAILED"
	ErrCheckoutTransactionFailed = "TRANSACTION_FAILED"
)
const (
	// ==================== VALIDATION ERRORS (400) ====================
	ErrCodeInvalidRequest       = "ORD001"
	ErrCodeInvalidItems         = "ORD002"
	ErrCodeEmptyCart            = "ORD003"
	ErrCodeInvalidQuantity      = "ORD004"
	ErrCodeInvalidAddress       = "ORD005"
	ErrCodeInvalidPaymentMethod = "ORD006"
	ErrCodeInvalidPromoCode     = "ORD007"
	ErrCodeMaxItemsExceeded     = "ORD008"
	ErrCodeMinOrderAmountNotMet = "ORD009"

	// ==================== CART ERRORS (400/409) ====================
	ErrCodeCartNotFound        = "ORD010"
	ErrCodeCartExpired         = "ORD011"
	ErrCodeCartModified        = "ORD012" // ✅ Already defined
	ErrCodeCartLocked          = "ORD013"
	ErrCodeCartEmpty           = "ORD014"
	ErrCodeCartVersionMismatch = "ORD015"

	// ==================== INVENTORY ERRORS (409/422) ====================
	ErrCodeInsufficientStock      = "ORD020" // ✅ Already defined
	ErrCodeStockLocked            = "ORD021" // ✅ Already defined
	ErrCodeStockReservationFailed = "ORD022"
	ErrCodeWarehouseUnavailable   = "ORD023"
	ErrCodeItemDiscontinued       = "ORD024"
	ErrCodeItemOutOfStock         = "ORD025"

	// ==================== PRICING ERRORS (409/422) ====================
	ErrCodePriceMismatch    = "ORD030" // ✅ Already defined
	ErrCodePriceChanged     = "ORD031"
	ErrCodeDiscountInvalid  = "ORD032"
	ErrCodeTotalMismatch    = "ORD033"
	ErrCodeCurrencyMismatch = "ORD034"

	// ==================== PROMOTION ERRORS (400/422) ====================
	ErrCodePromoInvalid           = "ORD040"
	ErrCodePromoExpired           = "ORD041"
	ErrCodePromoNotStarted        = "ORD042"
	ErrCodePromoUsageLimitReached = "ORD043"
	ErrCodePromoMinAmountNotMet   = "ORD044"
	ErrCodePromoNotApplicable     = "ORD045"
	ErrCodePromoAlreadyUsed       = "ORD046"

	// ==================== ADDRESS ERRORS (400/404) ====================
	ErrCodeAddressNotFound           = "ORD050"
	ErrCodeAddressInvalid            = "ORD051"
	ErrCodeAddressNotBelongToUser    = "ORD052"
	ErrCodeAddressMissingCoordinates = "ORD053"
	ErrCodeAddressOutOfDeliveryZone  = "ORD054"

	// ==================== PAYMENT ERRORS (402/422) ====================
	ErrCodePaymentFailed             = "ORD060"
	ErrCodePaymentMethodNotSupported = "ORD061"
	ErrCodePaymentTimeout            = "ORD062"
	ErrCodePaymentDeclined           = "ORD063"
	ErrCodeInsufficientFunds         = "ORD064"
	ErrCodePaymentGatewayError       = "ORD065"

	// ==================== ORDER ERRORS (404/409) ====================
	ErrCodeOrderNotFound          = "ORD070"
	ErrCodeOrderAlreadyExists     = "ORD071"
	ErrCodeOrderCancelled         = "ORD072"
	ErrCodeOrderCompleted         = "ORD073"
	ErrCodeOrderInvalidStatus     = "ORD074"
	ErrCodeOrderCannotBeCancelled = "ORD075"
	ErrCodeOrderCannotBeModified  = "ORD076"

	// ==================== CONCURRENCY ERRORS (409/429) ====================
	ErrCodeConcurrentCheckout     = "ORD080" // ✅ Already defined
	ErrCodeConcurrentModification = "ORD081"
	ErrCodeResourceLocked         = "ORD082"
	ErrCodeTooManyRequests        = "ORD083"
	ErrCodeDuplicateRequest       = "ORD084"

	// ==================== AUTHORIZATION ERRORS (401/403) ====================
	ErrCodeUnauthorized = "ORD090"
	ErrCodeForbidden    = "ORD091"
	ErrCodeAccessDenied = "ORD092"
	ErrCodeInvalidToken = "ORD093"
	ErrCodeTokenExpired = "ORD094"

	// ==================== BUSINESS RULE ERRORS (422) ====================
	ErrCodeCheckoutExpired        = "ORD100"
	ErrCodeCheckoutSessionInvalid = "ORD101"
	ErrCodeMaxOrdersPerDayReached = "ORD102"
	ErrCodeUserBlacklisted        = "ORD103"
	ErrCodeRegionRestricted       = "ORD104"
	ErrCodeBusinessHoursClosed    = "ORD105"

	// ==================== SYSTEM ERRORS (500/503) ====================
	ErrCodeInternalError        = "ORD500"
	ErrCodeDatabaseError        = "ORD501"
	ErrCodeTransactionFailed    = "ORD502"
	ErrCodeServiceUnavailable   = "ORD503"
	ErrCodeTimeoutError         = "ORD504"
	ErrCodeExternalServiceError = "ORD505"
)

var (
	ErrItemNotFound        = errors.New("item not found")
	ErrItemNotBelongToCart = errors.New("item does not belong to cart")
)
var (
	ErrCartVersionMismatch = errors.New("cart was modified by another process")
	ErrCartLocked          = errors.New("cart is being processed by another request")
)
var ErrorMessages = map[string]string{
	// Validation Errors
	ErrCodeInvalidRequest:       "Invalid request format or missing required fields",
	ErrCodeInvalidItems:         "One or more items in the order are invalid",
	ErrCodeEmptyCart:            "Cart is empty. Add items before checkout",
	ErrCodeInvalidQuantity:      "Invalid quantity specified for item",
	ErrCodeInvalidAddress:       "Shipping address is invalid or incomplete",
	ErrCodeInvalidPaymentMethod: "Payment method is not supported",
	ErrCodeInvalidPromoCode:     "Invalid or expired promotional code",
	ErrCodeMaxItemsExceeded:     "Maximum number of items per order exceeded",
	ErrCodeMinOrderAmountNotMet: "Order amount does not meet minimum requirement",

	// Cart Errors
	ErrCodeCartNotFound:        "Shopping cart not found",
	ErrCodeCartExpired:         "Shopping cart has expired",
	ErrCodeCartModified:        "Cart was modified during checkout. Please review and try again",
	ErrCodeCartLocked:          "Cart is currently being processed by another request",
	ErrCodeCartEmpty:           "Cart is empty or has no valid items",
	ErrCodeCartVersionMismatch: "Cart version mismatch. Cart was updated by another process",

	// Inventory Errors
	ErrCodeInsufficientStock:      "Insufficient stock available for one or more items",
	ErrCodeStockLocked:            "Item is currently reserved by another order",
	ErrCodeStockReservationFailed: "Failed to reserve inventory for your order",
	ErrCodeWarehouseUnavailable:   "No warehouse available to fulfill this order",
	ErrCodeItemDiscontinued:       "One or more items have been discontinued",
	ErrCodeItemOutOfStock:         "Item is currently out of stock",

	// Pricing Errors
	ErrCodePriceMismatch:    "Price has changed since cart was created",
	ErrCodePriceChanged:     "One or more item prices have been updated",
	ErrCodeDiscountInvalid:  "Discount calculation error",
	ErrCodeTotalMismatch:    "Order total does not match expected amount",
	ErrCodeCurrencyMismatch: "Currency mismatch in order calculation",

	// Promotion Errors
	ErrCodePromoInvalid:           "Promotional code is invalid",
	ErrCodePromoExpired:           "Promotional code has expired",
	ErrCodePromoNotStarted:        "Promotional code is not yet active",
	ErrCodePromoUsageLimitReached: "Promotional code usage limit reached",
	ErrCodePromoMinAmountNotMet:   "Order amount does not meet promotion minimum",
	ErrCodePromoNotApplicable:     "Promotion cannot be applied to these items",
	ErrCodePromoAlreadyUsed:       "You have already used this promotional code",

	// Address Errors
	ErrCodeAddressNotFound:           "Shipping address not found",
	ErrCodeAddressInvalid:            "Shipping address is invalid or incomplete",
	ErrCodeAddressNotBelongToUser:    "Address does not belong to the current user",
	ErrCodeAddressMissingCoordinates: "Address is missing location coordinates",
	ErrCodeAddressOutOfDeliveryZone:  "Address is outside our delivery zone",

	// Payment Errors
	ErrCodePaymentFailed:             "Payment processing failed",
	ErrCodePaymentMethodNotSupported: "Payment method is not supported",
	ErrCodePaymentTimeout:            "Payment request timed out",
	ErrCodePaymentDeclined:           "Payment was declined by payment provider",
	ErrCodeInsufficientFunds:         "Insufficient funds for this transaction",
	ErrCodePaymentGatewayError:       "Payment gateway error. Please try again",

	// Order Errors
	ErrCodeOrderNotFound:          "Order not found",
	ErrCodeOrderAlreadyExists:     "Order already exists",
	ErrCodeOrderCancelled:         "Order has been cancelled",
	ErrCodeOrderCompleted:         "Order has already been completed",
	ErrCodeOrderInvalidStatus:     "Invalid order status for this operation",
	ErrCodeOrderCannotBeCancelled: "Order cannot be cancelled in current status",
	ErrCodeOrderCannotBeModified:  "Order cannot be modified in current status",

	// Concurrency Errors
	ErrCodeConcurrentCheckout:     "Another checkout is in progress for your cart. Please wait",
	ErrCodeConcurrentModification: "Resource is being modified by another process",
	ErrCodeResourceLocked:         "Resource is temporarily locked. Please try again",
	ErrCodeTooManyRequests:        "Too many requests. Please slow down",
	ErrCodeDuplicateRequest:       "Duplicate request detected. Original request is being processed",

	// Authorization Errors
	ErrCodeUnauthorized: "Authentication required",
	ErrCodeForbidden:    "You do not have permission to perform this action",
	ErrCodeAccessDenied: "Access denied to this resource",
	ErrCodeInvalidToken: "Invalid authentication token",
	ErrCodeTokenExpired: "Authentication token has expired",

	// Business Rule Errors
	ErrCodeCheckoutExpired:        "Checkout session has expired. Please start over",
	ErrCodeCheckoutSessionInvalid: "Invalid checkout session",
	ErrCodeMaxOrdersPerDayReached: "You have reached the maximum number of orders per day",
	ErrCodeUserBlacklisted:        "Your account is restricted from placing orders",
	ErrCodeRegionRestricted:       "Orders are restricted in your region",
	ErrCodeBusinessHoursClosed:    "Orders can only be placed during business hours",

	// System Errors
	ErrCodeInternalError:        "An internal error occurred. Please try again later",
	ErrCodeDatabaseError:        "Database error occurred",
	ErrCodeTransactionFailed:    "Transaction failed. Please try again",
	ErrCodeServiceUnavailable:   "Service is temporarily unavailable",
	ErrCodeTimeoutError:         "Request timed out. Please try again",
	ErrCodeExternalServiceError: "External service error. Please try again later",
}

// GetErrorMessage returns the user-friendly message for an error code
func GetErrorMessage(code string) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "An unknown error occurred"
}

type OrderError struct {
	Code       string                 `json:"code"`               // Error code (e.g., "ORD012")
	Message    string                 `json:"message"`            // User-friendly message
	Details    string                 `json:"details,omitempty"`  // Technical details
	Field      string                 `json:"field,omitempty"`    // Field that caused error
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Additional context
	HTTPStatus int                    `json:"-"`                  // HTTP status code
	OrigError  error                  `json:"-"`                  // Original error (not exposed)
}

// Error implements the error interface
func (e *OrderError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the original error
func (e *OrderError) Unwrap() error {
	return e.OrigError
}

// WithField adds field information to the error
func (e *OrderError) WithField(field string) *OrderError {
	e.Field = field
	return e
}

// WithMetadata adds metadata to the error
func (e *OrderError) WithMetadata(key string, value interface{}) *OrderError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// ToJSON converts error to JSON for API response
func (e *OrderError) ToJSON() []byte {
	data, _ := json.Marshal(e)
	return data
}

// NewOrderError creates a new OrderError
func NewOrderError(code string, details string, origErr error) *OrderError {
	message := GetErrorMessage(code)
	httpStatus := getHTTPStatusForErrorCode(code)

	return &OrderError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: httpStatus,
		OrigError:  origErr,
		Metadata:   make(map[string]interface{}),
	}
}

// NewOrderErrorWithField creates error with field information
func NewOrderErrorWithField(code string, details string, field string, origErr error) *OrderError {
	err := NewOrderError(code, details, origErr)
	err.Field = field
	return err
}
func getHTTPStatusForErrorCode(code string) int {
	// Map based on error code ranges
	switch {
	// Validation errors (400)
	case code >= "ORD001" && code <= "ORD009":
		return http.StatusBadRequest

	// Cart errors (400/409)
	case code >= "ORD010" && code <= "ORD015":
		if code == ErrCodeCartModified || code == ErrCodeCartLocked || code == ErrCodeCartVersionMismatch {
			return http.StatusConflict // 409
		}
		return http.StatusBadRequest // 400

	// Inventory errors (409/422)
	case code >= "ORD020" && code <= "ORD025":
		if code == ErrCodeStockLocked {
			return http.StatusConflict // 409
		}
		return http.StatusUnprocessableEntity // 422

	// Pricing errors (409/422)
	case code >= "ORD030" && code <= "ORD034":
		if code == ErrCodePriceMismatch || code == ErrCodePriceChanged {
			return http.StatusConflict // 409
		}
		return http.StatusUnprocessableEntity // 422

	// Promotion errors (400/422)
	case code >= "ORD040" && code <= "ORD046":
		if code == ErrCodePromoInvalid {
			return http.StatusBadRequest // 400
		}
		return http.StatusUnprocessableEntity // 422

	// Address errors (400/404)
	case code >= "ORD050" && code <= "ORD054":
		if code == ErrCodeAddressNotFound {
			return http.StatusNotFound // 404
		}
		return http.StatusBadRequest // 400

	// Payment errors (402/422)
	case code >= "ORD060" && code <= "ORD065":
		if code == ErrCodePaymentFailed || code == ErrCodeInsufficientFunds {
			return http.StatusPaymentRequired // 402
		}
		return http.StatusUnprocessableEntity // 422

	// Order errors (404/409)
	case code >= "ORD070" && code <= "ORD076":
		if code == ErrCodeOrderNotFound {
			return http.StatusNotFound // 404
		}
		return http.StatusConflict // 409

	// Concurrency errors (409/429)
	case code >= "ORD080" && code <= "ORD084":
		if code == ErrCodeTooManyRequests {
			return http.StatusTooManyRequests // 429
		}
		return http.StatusConflict // 409

	// Authorization errors (401/403)
	case code >= "ORD090" && code <= "ORD094":
		if code == ErrCodeUnauthorized || code == ErrCodeInvalidToken || code == ErrCodeTokenExpired {
			return http.StatusUnauthorized // 401
		}
		return http.StatusForbidden // 403

	// Business rule errors (422)
	case code >= "ORD100" && code <= "ORD105":
		return http.StatusUnprocessableEntity // 422

	// System errors (500/503)
	case code >= "ORD500" && code <= "ORD505":
		if code == ErrCodeServiceUnavailable || code == ErrCodeTimeoutError {
			return http.StatusServiceUnavailable // 503
		}
		return http.StatusInternalServerError // 500

	default:
		return http.StatusInternalServerError // 500
	}
}

// ErrorCodeToHTTPStatus provides a public function to get HTTP status
func ErrorCodeToHTTPStatus(code string) int {
	return getHTTPStatusForErrorCode(code)
}
func ErrInvalidRequest(details string) *OrderError {
	return NewOrderError(ErrCodeInvalidRequest, details, nil)
}

func ErrInvalidItems(details string) *OrderError {
	return NewOrderError(ErrCodeInvalidItems, details, nil)
}

func ErrEmptyCart() *OrderError {
	return NewOrderError(ErrCodeEmptyCart, "", nil)
}

// Cart Error Helpers
func ErrCartNotFoundError() *OrderError {
	return NewOrderError(ErrCodeCartNotFound, "", nil)
}

func ErrCartModifiedError() *OrderError {
	return NewOrderError(ErrCodeCartModified, "", nil)
}

func ErrCartLockedError() *OrderError {
	return NewOrderError(ErrCodeCartLocked, "", nil)
}

func ErrCartVersionMismatchError(expectedVersion, actualVersion int) *OrderError {
	return NewOrderError(
		ErrCodeCartVersionMismatch,
		fmt.Sprintf("Expected version %d, but found %d", expectedVersion, actualVersion),
		nil,
	).WithMetadata("expected_version", expectedVersion).
		WithMetadata("actual_version", actualVersion)
}

// Inventory Error Helpers
func ErrInsufficientStockError(bookTitle string, requested, available int) *OrderError {
	return NewOrderError(
		ErrCodeInsufficientStock,
		fmt.Sprintf("Book '%s': Requested %d, but only %d available", bookTitle, requested, available),
		nil,
	).WithField(bookTitle).
		WithMetadata("requested", requested).
		WithMetadata("available", available)
}

func ErrStockLockedError(bookTitle string) *OrderError {
	return NewOrderError(
		ErrCodeStockLocked,
		fmt.Sprintf("Book '%s' is currently reserved by another order", bookTitle),
		nil,
	).WithField(bookTitle)
}

// Pricing Error Helpers
func ErrPriceMismatchError(bookTitle string, cartPrice, currentPrice string) *OrderError {
	return NewOrderError(
		ErrCodePriceMismatch,
		fmt.Sprintf("Price for '%s' changed from %s to %s", bookTitle, cartPrice, currentPrice),
		nil,
	).WithField(bookTitle).
		WithMetadata("cart_price", cartPrice).
		WithMetadata("current_price", currentPrice)
}

// Promotion Error Helpers
func ErrPromoInvalidError(promoCode string, reason string) *OrderError {
	return NewOrderError(
		ErrCodePromoInvalid,
		fmt.Sprintf("Promo code '%s': %s", promoCode, reason),
		nil,
	).WithField("promo_code").
		WithMetadata("promo_code", promoCode).
		WithMetadata("reason", reason)
}

func ErrPromoExpiredError(promoCode string) *OrderError {
	return NewOrderError(
		ErrCodePromoExpired,
		fmt.Sprintf("Promo code '%s' has expired", promoCode),
		nil,
	).WithField("promo_code")
}

// Concurrency Error Helpers
func ErrConcurrentCheckoutError() *OrderError {
	return NewOrderError(ErrCodeConcurrentCheckout, "", nil)
}

func ErrDuplicateRequestError(idempotencyKey string) *OrderError {
	return NewOrderError(
		ErrCodeDuplicateRequest,
		fmt.Sprintf("Request with idempotency key '%s' is already being processed", idempotencyKey),
		nil,
	).WithMetadata("idempotency_key", idempotencyKey)
}

// System Error Helpers
func ErrInternalServerError(details string, origErr error) *OrderError {
	return NewOrderError(ErrCodeInternalError, details, origErr)
}

func ErrDatabaseError(operation string, origErr error) *OrderError {
	return NewOrderError(
		ErrCodeDatabaseError,
		fmt.Sprintf("Database error during %s", operation),
		origErr,
	).WithMetadata("operation", operation)
}

func ErrTransactionFailedError(origErr error) *OrderError {
	return NewOrderError(ErrCodeTransactionFailed, "", origErr)
}
