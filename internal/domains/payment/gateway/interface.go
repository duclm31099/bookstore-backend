package gateway

import (
	"context"

	"github.com/shopspring/decimal"

	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// GATEWAY INTERFACES
// =====================================================

// VNPayGateway interface for VNPay payment gateway integration
type VNPayGateway interface {
	// CreatePaymentURL generates VNPay payment URL
	CreatePaymentURL(ctx context.Context, req VNPayPaymentRequest) (string, error)

	// VerifySignature verifies VNPay webhook signature
	VerifySignature(webhookData model.VNPayWebhookRequest) bool

	// InitiateRefund initiates refund via VNPay API
	InitiateRefund(ctx context.Context, req VNPayRefundRequest) (*VNPayRefundResponse, error)

	// GetReturnURL gets frontend return URL
	GetReturnURL() string
}

// MomoGateway interface for Momo payment gateway integration
type MomoGateway interface {
	// CreatePaymentURL generates Momo payment URL
	CreatePaymentURL(ctx context.Context, req MomoPaymentRequest) (string, error)

	// VerifySignature verifies Momo webhook signature
	VerifySignature(webhookData model.MomoWebhookRequest) bool

	// InitiateRefund initiates refund via Momo API
	InitiateRefund(ctx context.Context, req MomoRefundRequest) (*MomoRefundResponse, error)
}

// =====================================================
// COMMON REQUEST/RESPONSE TYPES
// =====================================================

// VNPayPaymentRequest request to create VNPay payment
type VNPayPaymentRequest struct {
	TransactionRef string          // payment_transaction.id
	Amount         decimal.Decimal // Order total
	OrderInfo      string          // Description
	ReturnURL      string          // Frontend callback URL
}

// VNPayRefundRequest request to initiate VNPay refund
type VNPayRefundRequest struct {
	TransactionID   string          // Original transaction ID from VNPay
	Amount          decimal.Decimal // Original amount
	RefundAmount    decimal.Decimal // Amount to refund
	TransactionDate string          // Original transaction date (yyyyMMddHHmmss)
	Reason          string          // Refund reason
}

// VNPayRefundResponse response from VNPay refund API
type VNPayRefundResponse struct {
	RefundTransactionID string                 // VNPay refund transaction ID
	ResponseCode        string                 // "00" = success
	Message             string                 // Response message
	RawResponse         map[string]interface{} // Full response for audit
}

// MomoPaymentRequest request to create Momo payment
type MomoPaymentRequest struct {
	OrderID   string          // payment_transaction.id
	Amount    decimal.Decimal // Order total
	OrderInfo string          // Description
}

// MomoRefundRequest request to initiate Momo refund
type MomoRefundRequest struct {
	TransactionID string          // Original transaction ID from Momo
	Amount        decimal.Decimal // Amount to refund
	Reason        string          // Refund reason
}

// MomoRefundResponse response from Momo refund API
type MomoRefundResponse struct {
	RefundTransactionID string                 // Momo refund transaction ID
	ResultCode          int                    // 0 = success
	Message             string                 // Response message
	RawResponse         map[string]interface{} // Full response for audit
}
