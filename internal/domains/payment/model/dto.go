package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =====================================================
// CREATE PAYMENT REQUEST/RESPONSE
// =====================================================

type CreatePaymentRequest struct {
	OrderID uuid.UUID `json:"order_id" binding:"required"`
	Gateway string    `json:"gateway" binding:"required,oneof=cod vnpay momo bank_transfer"`
}

func (r *CreatePaymentRequest) Validate() error {
	if r.OrderID == uuid.Nil {
		return fmt.Errorf("order_id is required")
	}

	// Validate gateway
	validGateway := false
	for _, g := range ValidGateways {
		if r.Gateway == g {
			validGateway = true
			break
		}
	}
	if !validGateway {
		return fmt.Errorf("invalid gateway: %s", r.Gateway)
	}

	return nil
}

type CreatePaymentResponse struct {
	PaymentTransactionID uuid.UUID       `json:"payment_transaction_id"`
	Gateway              string          `json:"gateway"`
	Amount               decimal.Decimal `json:"amount"`
	Currency             string          `json:"currency"`
	PaymentURL           *string         `json:"payment_url,omitempty"`  // For VNPay/Momo
	QRCode               *string         `json:"qr_code,omitempty"`      // For Bank Transfer
	BankAccount          *string         `json:"bank_account,omitempty"` // For Bank Transfer
	ExpiresAt            time.Time       `json:"expires_at"`
	Message              *string         `json:"message,omitempty"` // For COD
}

// =====================================================
// GET PAYMENT STATUS RESPONSE
// =====================================================

type PaymentStatusResponse struct {
	TransactionID  uuid.UUID              `json:"transaction_id"`
	OrderID        uuid.UUID              `json:"order_id"`
	Gateway        string                 `json:"gateway"`
	Status         string                 `json:"status"`
	Amount         decimal.Decimal        `json:"amount"`
	Currency       string                 `json:"currency"`
	TransactionRef *string                `json:"transaction_ref,omitempty"`
	ErrorCode      *string                `json:"error_code,omitempty"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
	PaymentDetails map[string]interface{} `json:"payment_details,omitempty"`
	InitiatedAt    time.Time              `json:"initiated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	FailedAt       *time.Time             `json:"failed_at,omitempty"`
}

// =====================================================
// LIST PAYMENTS REQUEST/RESPONSE
// =====================================================

type ListPaymentsRequest struct {
	OrderID *uuid.UUID `form:"order_id"`
	Status  *string    `form:"status"`
	Gateway *string    `form:"gateway"`
	Page    int        `form:"page" binding:"min=1"`
	Limit   int        `form:"limit" binding:"min=1,max=100"`
}

func (r *ListPaymentsRequest) Validate() error {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.Limit < 1 || r.Limit > 100 {
		r.Limit = 20
	}
	return nil
}

type PaymentSummaryResponse struct {
	TransactionID uuid.UUID       `json:"transaction_id"`
	OrderID       uuid.UUID       `json:"order_id"`
	Gateway       string          `json:"gateway"`
	Status        string          `json:"status"`
	Amount        decimal.Decimal `json:"amount"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type ListPaymentsResponse struct {
	Payments   []PaymentSummaryResponse `json:"payments"`
	Pagination PaginationMeta           `json:"pagination"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// =====================================================
// REFUND REQUEST DTOs
// =====================================================

type CreateRefundRequestDTO struct {
	Reason      string   `json:"reason" binding:"required,min=10,max=500"`
	ProofImages []string `json:"proof_images,omitempty"`
}

func (r *CreateRefundRequestDTO) Validate() error {
	if len(r.Reason) < 10 {
		return fmt.Errorf("reason must be at least 10 characters")
	}
	if len(r.Reason) > 500 {
		return fmt.Errorf("reason must not exceed 500 characters")
	}
	return nil
}

type RefundRequestResponse struct {
	RefundRequestID         uuid.UUID       `json:"refund_request_id"`
	PaymentTransactionID    uuid.UUID       `json:"payment_transaction_id"`
	Status                  string          `json:"status"`
	RequestedAmount         decimal.Decimal `json:"requested_amount"`
	Reason                  string          `json:"reason"`
	RequestedAt             time.Time       `json:"requested_at"`
	EstimatedProcessingTime string          `json:"estimated_processing_time,omitempty"`
	ApprovedBy              *string         `json:"approved_by,omitempty"`
	ApprovedAt              *time.Time      `json:"approved_at,omitempty"`
	RejectedBy              *string         `json:"rejected_by,omitempty"`
	RejectedAt              *time.Time      `json:"rejected_at,omitempty"`
	RejectionReason         *string         `json:"rejection_reason,omitempty"`
	GatewayRefundID         *string         `json:"gateway_refund_id,omitempty"`
	CompletedAt             *time.Time      `json:"completed_at,omitempty"`
}

// =====================================================
// ADMIN: APPROVE/REJECT REFUND DTOs
// =====================================================

type ApproveRefundRequestDTO struct {
	AdminNotes *string `json:"admin_notes,omitempty"`
}

type RejectRefundRequestDTO struct {
	RejectionReason string `json:"rejection_reason" binding:"required,min=10"`
}

func (r *RejectRefundRequestDTO) Validate() error {
	if len(r.RejectionReason) < 10 {
		return fmt.Errorf("rejection reason must be at least 10 characters")
	}
	return nil
}

// =====================================================
// ADMIN: LIST PAYMENTS DTOs
// =====================================================

type AdminListPaymentsRequest struct {
	Status    *string    `form:"status"`
	Gateway   *string    `form:"gateway"`
	FromDate  *time.Time `form:"from_date"`
	ToDate    *time.Time `form:"to_date"`
	MinAmount *float64   `form:"min_amount"`
	MaxAmount *float64   `form:"max_amount"`
	Search    *string    `form:"search"` // Search by order_number or transaction_id
	Page      int        `form:"page" binding:"min=1"`
	Limit     int        `form:"limit" binding:"min=1,max=100"`
}

func (r *AdminListPaymentsRequest) Validate() error {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.Limit < 1 || r.Limit > 100 {
		r.Limit = 20
	}
	return nil
}

type AdminPaymentResponse struct {
	TransactionID    uuid.UUID       `json:"transaction_id"`
	OrderNumber      string          `json:"order_number"`
	UserEmail        string          `json:"user_email"`
	Gateway          string          `json:"gateway"`
	Status           string          `json:"status"`
	Amount           decimal.Decimal `json:"amount"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	HasRefundRequest bool            `json:"has_refund_request"`
	CreatedAt        time.Time       `json:"created_at"`
}

type AdminListPaymentsResponse struct {
	Payments   []AdminPaymentResponse `json:"payments"`
	Statistics PaymentStatistics      `json:"statistics"`
	Pagination PaginationMeta         `json:"pagination"`
}

type PaymentStatistics struct {
	TotalAmount  decimal.Decimal `json:"total_amount"`
	SuccessCount int             `json:"success_count"`
	PendingCount int             `json:"pending_count"`
	FailedCount  int             `json:"failed_count"`
}

// =====================================================
// ADMIN: PAYMENT DETAIL DTOs
// =====================================================

type AdminPaymentDetailResponse struct {
	TransactionID   uuid.UUID              `json:"transaction_id"`
	Order           OrderInfo              `json:"order"`
	Gateway         string                 `json:"gateway"`
	Status          string                 `json:"status"`
	Amount          decimal.Decimal        `json:"amount"`
	GatewayResponse map[string]interface{} `json:"gateway_response,omitempty"`
	PaymentDetails  map[string]interface{} `json:"payment_details,omitempty"`
	WebhookLogs     []WebhookLogSummary    `json:"webhook_logs,omitempty"`
	RetryCount      int                    `json:"retry_count"`
	InitiatedAt     time.Time              `json:"initiated_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	FailedAt        *time.Time             `json:"failed_at,omitempty"`
}

type OrderInfo struct {
	ID          uuid.UUID `json:"id"`
	OrderNumber string    `json:"order_number"`
	User        UserInfo  `json:"user"`
}

type UserInfo struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

type WebhookLogSummary struct {
	ID          uuid.UUID `json:"id"`
	Event       *string   `json:"event,omitempty"`
	IsValid     *bool     `json:"is_valid,omitempty"`
	IsProcessed bool      `json:"is_processed"`
	ReceivedAt  time.Time `json:"received_at"`
}

// =====================================================
// ADMIN: MANUAL RECONCILIATION DTO
// =====================================================

type ManualReconciliationRequest struct {
	Status               string `json:"status" binding:"required,oneof=success failed"`
	GatewayTransactionID string `json:"gateway_transaction_id" binding:"required"`
	Notes                string `json:"notes" binding:"required,min=10"`
}

func (r *ManualReconciliationRequest) Validate() error {
	if r.Status != PaymentStatusSuccess && r.Status != PaymentStatusFailed {
		return fmt.Errorf("status must be 'success' or 'failed'")
	}
	if r.GatewayTransactionID == "" {
		return fmt.Errorf("gateway_transaction_id is required")
	}
	if len(r.Notes) < 10 {
		return fmt.Errorf("notes must be at least 10 characters")
	}
	return nil
}

// =====================================================
// WEBHOOK REQUEST DTOs
// =====================================================

// MomoWebhookRequest represents Momo IPN callback
type MomoWebhookRequest struct {
	PartnerCode  string `json:"partnerCode"`
	OrderID      string `json:"orderId"` // payment_transaction.id
	RequestID    string `json:"requestId"`
	Amount       int64  `json:"amount"`
	OrderInfo    string `json:"orderInfo"`
	OrderType    string `json:"orderType"`
	TransID      string `json:"transId"`
	ResultCode   int    `json:"resultCode"`
	Message      string `json:"message"`
	PayType      string `json:"payType"`
	ResponseTime int64  `json:"responseTime"`
	ExtraData    string `json:"extraData"`
	Signature    string `json:"signature"`
}

// Update VNPayWebhookRequest to include missing field
type VNPayWebhookRequest struct {
	VnpAmount            string `form:"vnp_Amount"`
	VnpBankCode          string `form:"vnp_BankCode"`
	VnpBankTranNo        string `form:"vnp_BankTranNo"` // ← ADD THIS
	VnpCardType          string `form:"vnp_CardType"`
	VnpOrderInfo         string `form:"vnp_OrderInfo"`
	VnpPayDate           string `form:"vnp_PayDate"`
	VnpResponseCode      string `form:"vnp_ResponseCode"`
	VnpTmnCode           string `form:"vnp_TmnCode"`
	VnpTransactionNo     string `form:"vnp_TransactionNo"`
	VnpTxnRef            string `form:"vnp_TxnRef"`
	VnpSecureHash        string `form:"vnp_SecureHash"`
	VnpTransactionStatus string `form:"vnp_TransactionStatus"`
}

// =====================================================
// VERIFY PAYMENT RESPONSE (for ReturnURL verification)
// =====================================================

type VerifyPaymentResponse struct {
	Success          bool            `json:"success"`
	PaymentID        uuid.UUID       `json:"payment_id"`
	OrderID          uuid.UUID       `json:"order_id"`
	Status           string          `json:"status"`
	Amount           decimal.Decimal `json:"amount"`
	TransactionNo    string          `json:"transaction_no,omitempty"`
	BankCode         string          `json:"bank_code,omitempty"`
	PayDate          string          `json:"pay_date,omitempty"`
	Message          string          `json:"message"`
	ResponseCode     string          `json:"response_code"`
	AlreadyProcessed bool            `json:"already_processed,omitempty"`
}
