package model

import (
	"errors"
	"fmt"
)

// =====================================================
// PREDEFINED ERRORS
// =====================================================

var (
	ErrPaymentNotFound         = errors.New("payment transaction not found")
	ErrOrderAlreadyPaid        = errors.New("order already paid")
	ErrRetryLimitExceeded      = errors.New("payment retry limit exceeded")
	ErrOrderNotPending         = errors.New("order is not in pending status")
	ErrInvalidGateway          = errors.New("invalid payment gateway")
	ErrRefundNotAllowed        = errors.New("refund not allowed")
	ErrPaymentNotSuccessful    = errors.New("payment is not successful")
	ErrOrderCannotRefund       = errors.New("order cannot be refunded")
	ErrRefundWindowExpired     = errors.New("refund window expired")
	ErrRefundAlreadyExists     = errors.New("refund request already exists")
	ErrCODNoRefund             = errors.New("COD orders cannot be refunded")
	ErrInvalidSignature        = errors.New("invalid webhook signature")
	ErrWebhookAlreadyProcessed = errors.New("webhook already processed")
	ErrUnauthorized            = errors.New("unauthorized access")
	ErrOrderCancelled          = errors.New("order is already cancelled")
	ErrRefundRequestNotFound   = errors.New("refund request not found")
	ErrCannotApproveRefund     = errors.New("cannot approve refund request")
	ErrCannotRejectRefund      = errors.New("cannot reject refund request")
)

// =====================================================
// CUSTOM PAYMENT ERROR
// =====================================================

type PaymentError struct {
	Code    string
	Message string
	Err     error
}

func (e *PaymentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *PaymentError) Unwrap() error {
	return e.Err
}

// NewPaymentError creates a new payment error
func NewPaymentError(code, message string, err error) *PaymentError {
	return &PaymentError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// =====================================================
// ERROR CONSTRUCTORS
// =====================================================

func NewPaymentNotFoundError(transactionID string) *PaymentError {
	return NewPaymentError(
		ErrCodePaymentNotFound,
		fmt.Sprintf("Payment transaction not found: %s", transactionID),
		ErrPaymentNotFound,
	)
}

func NewOrderAlreadyPaidError(orderID string) *PaymentError {
	return NewPaymentError(
		ErrCodeOrderAlreadyPaid,
		fmt.Sprintf("Order %s is already paid", orderID),
		ErrOrderAlreadyPaid,
	)
}

func NewRetryLimitExceededError() *PaymentError {
	return NewPaymentError(
		ErrCodeRetryLimitExceeded,
		fmt.Sprintf("Payment retry limit exceeded (max %d attempts)", MaxRetryAttempts),
		ErrRetryLimitExceeded,
	)
}

func NewOrderNotPendingError(status string) *PaymentError {
	return NewPaymentError(
		ErrCodeOrderNotPending,
		fmt.Sprintf("Order status must be 'pending', current status: %s", status),
		ErrOrderNotPending,
	)
}

func NewInvalidGatewayError(gateway string) *PaymentError {
	return NewPaymentError(
		ErrCodeInvalidGateway,
		fmt.Sprintf("Invalid payment gateway: %s", gateway),
		ErrInvalidGateway,
	)
}

func NewRefundNotAllowedError(reason string) *PaymentError {
	return NewPaymentError(
		ErrCodeRefundNotAllowed,
		fmt.Sprintf("Refund not allowed: %s", reason),
		ErrRefundNotAllowed,
	)
}

func NewInvalidSignatureError() *PaymentError {
	return NewPaymentError(
		ErrCodeInvalidSignature,
		"Invalid webhook signature - possible fraud attempt",
		ErrInvalidSignature,
	)
}

func NewWebhookAlreadyProcessedError() *PaymentError {
	return NewPaymentError(
		ErrCodeWebhookAlreadyProcessed,
		"Webhook already processed (idempotent)",
		ErrWebhookAlreadyProcessed,
	)
}
