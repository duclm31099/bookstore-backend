package model

import "errors"

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

var (
	ErrItemNotFound        = errors.New("item not found")
	ErrItemNotBelongToCart = errors.New("item does not belong to cart")
)
