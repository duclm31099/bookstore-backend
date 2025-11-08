package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =====================================================
// PAYMENT TRANSACTION ENTITY
// =====================================================
type PaymentTransaction struct {
	ID      uuid.UUID `json:"id" db:"id"`
	OrderID uuid.UUID `json:"order_id" db:"order_id"`

	// Gateway information
	Gateway       string  `json:"gateway" db:"gateway"`
	TransactionID *string `json:"transaction_id,omitempty" db:"transaction_id"`

	// Amount
	Amount   decimal.Decimal `json:"amount" db:"amount"`
	Currency string          `json:"currency" db:"currency"`

	// Status tracking
	Status       string  `json:"status" db:"status"`
	ErrorCode    *string `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage *string `json:"error_message,omitempty" db:"error_message"`

	// Gateway response (raw webhook data)
	GatewayResponse  map[string]interface{} `json:"gateway_response,omitempty" db:"gateway_response"`
	GatewaySignature *string                `json:"gateway_signature,omitempty" db:"gateway_signature"`

	// Payment method details
	PaymentDetails map[string]interface{} `json:"payment_details,omitempty" db:"payment_details"`

	// Refund tracking
	RefundAmount decimal.Decimal `json:"refund_amount" db:"refund_amount"`
	RefundReason *string         `json:"refund_reason,omitempty" db:"refund_reason"`
	RefundedAt   *time.Time      `json:"refunded_at,omitempty" db:"refunded_at"`

	// Retry tracking
	RetryCount int `json:"retry_count" db:"retry_count"`

	// Timestamps
	InitiatedAt  time.Time  `json:"initiated_at" db:"initiated_at"`
	ProcessingAt *time.Time `json:"processing_at,omitempty" db:"processing_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	FailedAt     *time.Time `json:"failed_at,omitempty" db:"failed_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// IsExpired checks if payment has expired (15 minutes timeout)
func (p *PaymentTransaction) IsExpired() bool {
	if p.Status != PaymentStatusPending && p.Status != PaymentStatusProcessing {
		return false
	}

	timeout := time.Duration(PaymentTimeoutMinutes) * time.Minute
	return time.Since(p.InitiatedAt) > timeout
}

// CanRetry checks if payment can be retried
func (p *PaymentTransaction) CanRetry() bool {
	return p.RetryCount < MaxRetryAttempts
}

// IsSuccessful checks if payment was successful
func (p *PaymentTransaction) IsSuccessful() bool {
	return p.Status == PaymentStatusSuccess
}

// CanBeRefunded checks if payment can be refunded
func (p *PaymentTransaction) CanBeRefunded() bool {
	// COD cannot be refunded (customer hasn't paid yet)
	if p.Gateway == GatewayCOD {
		return false
	}

	// Only successful payments can be refunded
	if p.Status != PaymentStatusSuccess {
		return false
	}

	// Check if already fully refunded
	if p.RefundAmount.GreaterThanOrEqual(p.Amount) {
		return false
	}

	return true
}

// =====================================================
// REFUND REQUEST ENTITY
// =====================================================
type RefundRequest struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	PaymentTransactionID uuid.UUID `json:"payment_transaction_id" db:"payment_transaction_id"`
	OrderID              uuid.UUID `json:"order_id" db:"order_id"`

	// Request details
	RequestedBy     uuid.UUID       `json:"requested_by" db:"requested_by"`
	RequestedAmount decimal.Decimal `json:"requested_amount" db:"requested_amount"`
	Reason          string          `json:"reason" db:"reason"`
	ProofImages     []string        `json:"proof_images,omitempty" db:"proof_images"`

	// Status
	Status string `json:"status" db:"status"`

	// Approval details
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	AdminNotes *string    `json:"admin_notes,omitempty" db:"admin_notes"`

	// Rejection details
	RejectedBy      *uuid.UUID `json:"rejected_by,omitempty" db:"rejected_by"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty" db:"rejected_at"`
	RejectionReason *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`

	// Gateway refund tracking
	GatewayRefundID       *string                `json:"gateway_refund_id,omitempty" db:"gateway_refund_id"`
	GatewayRefundResponse map[string]interface{} `json:"gateway_refund_response,omitempty" db:"gateway_refund_response"`

	// Timestamps
	RequestedAt  time.Time  `json:"requested_at" db:"requested_at"`
	ProcessingAt *time.Time `json:"processing_at,omitempty" db:"processing_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	FailedAt     *time.Time `json:"failed_at,omitempty" db:"failed_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// IsPending checks if refund request is pending approval
func (r *RefundRequest) IsPending() bool {
	return r.Status == RefundStatusPending
}

// IsApproved checks if refund request is approved
func (r *RefundRequest) IsApproved() bool {
	return r.Status == RefundStatusApproved
}

// IsCompleted checks if refund is completed
func (r *RefundRequest) IsCompleted() bool {
	return r.Status == RefundStatusCompleted
}

// CanBeApproved checks if refund request can be approved
func (r *RefundRequest) CanBeApproved() bool {
	return r.Status == RefundStatusPending
}

// CanBeRejected checks if refund request can be rejected
func (r *RefundRequest) CanBeRejected() bool {
	return r.Status == RefundStatusPending
}

// =====================================================
// PAYMENT WEBHOOK LOG ENTITY
// =====================================================
type PaymentWebhookLog struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	PaymentTransactionID *uuid.UUID `json:"payment_transaction_id,omitempty" db:"payment_transaction_id"`
	OrderID              *uuid.UUID `json:"order_id,omitempty" db:"order_id"`

	// Webhook details
	Gateway      string  `json:"gateway" db:"gateway"`
	WebhookEvent *string `json:"webhook_event,omitempty" db:"webhook_event"`

	// Request data
	Headers   map[string]interface{} `json:"headers,omitempty" db:"headers"`
	Body      map[string]interface{} `json:"body" db:"body"`
	Signature *string                `json:"signature,omitempty" db:"signature"`

	// Processing result
	IsValid         *bool   `json:"is_valid,omitempty" db:"is_valid"`
	IsProcessed     bool    `json:"is_processed" db:"is_processed"`
	ProcessingError *string `json:"processing_error,omitempty" db:"processing_error"`

	// Timestamp
	ReceivedAt time.Time `json:"received_at" db:"received_at"`
}

// MarkAsProcessed marks webhook as processed
func (w *PaymentWebhookLog) MarkAsProcessed() {
	w.IsProcessed = true
}

// MarkAsInvalid marks webhook as invalid (signature verification failed)
func (w *PaymentWebhookLog) MarkAsInvalid(reason string) {
	isValid := false
	w.IsValid = &isValid
	w.ProcessingError = &reason
}

// MarkProcessingError marks webhook processing error
func (w *PaymentWebhookLog) MarkProcessingError(err error) {
	errMsg := err.Error()
	w.ProcessingError = &errMsg
}
