package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// PAYMENT TRANSACTION REPOSITORY INTERFACE
// =====================================================
type PaymentRepoInteface interface {
	// ============================================
	// TRANSACTION-AWARE METHODS
	// ============================================

	// CreateWithTx creates payment transaction within provided transaction
	CreateWithTx(ctx context.Context, tx pgx.Tx, payment *model.PaymentTransaction) error

	// UpdateStatusWithTx updates payment status within transaction
	UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, paymentID uuid.UUID, status string, details map[string]interface{}) error

	// ============================================
	// STANDALONE METHODS
	// ============================================

	// Create creates payment transaction
	Create(ctx context.Context, payment *model.PaymentTransaction) error

	// GetByID gets payment transaction by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.PaymentTransaction, error)

	// GetByOrderID gets latest payment transaction for an order
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*model.PaymentTransaction, error)

	// GetByTransactionID gets payment by gateway transaction ID
	GetByTransactionID(ctx context.Context, gateway, transactionID string) (*model.PaymentTransaction, error)

	// ListByUserID lists payments for a user (with pagination)
	ListByUserID(ctx context.Context, userID uuid.UUID, filters map[string]interface{}, page, limit int) ([]*model.PaymentTransaction, int, error)

	// UpdateStatus updates payment status
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	// MarkAsFailed marks payment as failed with error details
	MarkAsFailed(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) error

	// MarkAsSuccess marks payment as successful with gateway response
	MarkAsSuccess(ctx context.Context, id uuid.UUID, transactionID string, gatewayResponse map[string]interface{}, paymentDetails map[string]interface{}) error

	// MarkAsCancelled marks payment as cancelled (timeout/user cancel)
	MarkAsCancelled(ctx context.Context, id uuid.UUID, reason string) error

	// CheckRetryLimit checks if order can retry payment
	CheckRetryLimit(ctx context.Context, orderID uuid.UUID) (bool, int, error)

	// GetExpiredPayments gets payments that have exceeded timeout
	GetExpiredPayments(ctx context.Context, limit int) ([]*model.PaymentTransaction, error)

	// HasSuccessfulPayment checks if order has successful payment
	HasSuccessfulPayment(ctx context.Context, orderID uuid.UUID) (bool, error)

	// ============================================
	// ADMIN METHODS
	// ============================================
	// NEW: Verify user ownership
	GetByIDAndVerifyUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.PaymentTransaction, error)

	// NEW: Get by order and status
	GetByOrderIDAndStatus(ctx context.Context, orderID uuid.UUID, status string) (*model.PaymentTransaction, error)

	// NEW: Update refund amount
	UpdateRefundAmount(ctx context.Context, id uuid.UUID, refundAmount decimal.Decimal, reason string) error

	// NEW: Batch cancel expired payments
	MarkExpiredPaymentsAsCancelled(ctx context.Context, reason string) (int, error)

	// AdminListPayments lists all payments with filters (admin)
	AdminListPayments(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*model.PaymentTransaction, int, error)

	// AdminGetStatistics gets payment statistics
	AdminGetStatistics(ctx context.Context, filters map[string]interface{}) (*model.PaymentStatistics, error)
}

// =====================================================
// REFUND REQUEST REPOSITORY INTERFACE
// =====================================================
type RefundRepoInterface interface {
	// ============================================
	// TRANSACTION-AWARE METHODS
	// ============================================

	// CreateWithTx creates refund request within transaction
	CreateWithTx(ctx context.Context, tx pgx.Tx, refund *model.RefundRequest) error

	// UpdateStatusWithTx updates refund request status within transaction
	UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error

	// ============================================
	// STANDALONE METHODS
	// ============================================

	// Create creates refund request
	Create(ctx context.Context, refund *model.RefundRequest) error

	// GetByID gets refund request by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.RefundRequest, error)

	// GetByPaymentID gets active refund request for a payment
	GetByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.RefundRequest, error)

	// Approve approves refund request
	Approve(ctx context.Context, id uuid.UUID, approvedBy uuid.UUID, notes *string) error

	// Reject rejects refund request
	Reject(ctx context.Context, id uuid.UUID, rejectedBy uuid.UUID, reason string) error

	// UpdateGatewayRefund updates gateway refund details
	UpdateGatewayRefund(ctx context.Context, id uuid.UUID, gatewayRefundID string, response map[string]interface{}) error

	// MarkAsCompleted marks refund as completed
	MarkAsCompleted(ctx context.Context, id uuid.UUID) error

	// MarkAsFailed marks refund as failed
	MarkAsFailed(ctx context.Context, id uuid.UUID, reason string) error

	// ListPendingRefunds lists pending refund requests (admin)
	ListPendingRefunds(ctx context.Context, page, limit int) ([]*model.RefundRequest, int, error)

	// HasPendingRefund checks if payment has pending refund request
	HasPendingRefund(ctx context.Context, paymentID uuid.UUID) (bool, error)
}

// =====================================================
// WEBHOOK LOG REPOSITORY INTERFACE
// =====================================================
type WebhookRepoInterface interface {
	// Create creates webhook log
	Create(ctx context.Context, log *model.PaymentWebhookLog) error

	// MarkAsProcessed marks webhook as processed
	MarkAsProcessed(ctx context.Context, id uuid.UUID) error

	// MarkAsInvalid marks webhook as invalid (signature failed)
	MarkAsInvalid(ctx context.Context, id uuid.UUID, reason string) error

	// CheckIdempotency checks if webhook already processed
	CheckIdempotency(ctx context.Context, gateway, event, transactionID string) (bool, error)

	// GetFailedWebhooks gets webhooks that failed processing (for retry)
	GetFailedWebhooks(ctx context.Context, limit int) ([]*model.PaymentWebhookLog, error)

	// ListByPaymentID lists webhook logs for a payment (admin)
	ListByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*model.PaymentWebhookLog, error)
	MarkProcessingError(ctx context.Context, id uuid.UUID, errorMsg string) error
}

// =====================================================
// TRANSACTION MANAGER
// =====================================================
type TransactionManager interface {
	// BeginTx starts a new transaction
	BeginTx(ctx context.Context) (pgx.Tx, error)

	// CommitTx commits transaction
	CommitTx(ctx context.Context, tx pgx.Tx) error

	// RollbackTx rolls back transaction
	RollbackTx(ctx context.Context, tx pgx.Tx) error
}
