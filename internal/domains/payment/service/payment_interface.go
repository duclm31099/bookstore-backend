package service

import (
	"context"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// PAYMENT SERVICE INTERFACE
// =====================================================
type PaymentService interface {
	// ============================================
	// USER ENDPOINTS
	// ============================================

	// CreatePayment initiates payment for an order
	// Returns payment URL for VNPay/Momo, or confirmation for COD
	CreatePayment(ctx context.Context, userID uuid.UUID, req model.CreatePaymentRequest) (*model.CreatePaymentResponse, error)

	// GetPaymentStatus gets payment status (for polling after redirect)
	GetPaymentStatus(ctx context.Context, userID uuid.UUID, paymentID uuid.UUID) (*model.PaymentStatusResponse, error)

	// ListUserPayments lists payments for current user
	ListUserPayments(ctx context.Context, userID uuid.UUID, req model.ListPaymentsRequest) (*model.ListPaymentsResponse, error)

	// ============================================
	// WEBHOOK PROCESSING
	// ============================================

	// ProcessVNPayWebhook processes VNPay IPN callback
	ProcessVNPayWebhook(ctx context.Context, webhookData model.VNPayWebhookRequest) error

	// ProcessMomoWebhook processes Momo IPN callback
	ProcessMomoWebhook(ctx context.Context, webhookData model.MomoWebhookRequest) error

	// ============================================
	// ADMIN ENDPOINTS
	// ============================================

	// AdminListPayments lists all payments with filters
	AdminListPayments(ctx context.Context, req model.AdminListPaymentsRequest) (*model.AdminListPaymentsResponse, error)

	// AdminGetPaymentDetail gets detailed payment info
	AdminGetPaymentDetail(ctx context.Context, paymentID uuid.UUID) (*model.AdminPaymentDetailResponse, error)

	// AdminReconcilePayment manually updates payment status
	AdminReconcilePayment(ctx context.Context, adminID uuid.UUID, paymentID uuid.UUID, req model.ManualReconciliationRequest) error

	// ============================================
	// BACKGROUND JOBS
	// ============================================

	// CancelExpiredPayments cancels payments that exceeded timeout (15 min)
	CancelExpiredPayments(ctx context.Context) (int, error)

	// RetryFailedWebhooks retries webhooks that failed processing
	RetryFailedWebhooks(ctx context.Context) (int, error)
}
