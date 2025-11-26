package model

// =====================================================
// PAYMENT GATEWAYS
// =====================================================
const (
	GatewayCOD          = "cod"
	GatewayVNPay        = "vnpay"
	GatewayMomo         = "momo"
	GatewayBankTransfer = "bank_transfer"
)

var ValidGateways = []string{
	GatewayCOD,
	GatewayVNPay,
	GatewayMomo,
	GatewayBankTransfer,
}

// =====================================================
// PAYMENT STATUS
// =====================================================
const (
	PaymentStatusPending    = "pending"
	PaymentStatusProcessing = "processing"
	PaymentStatusSuccess    = "success"
	PaymentStatusFailed     = "failed"
	PaymentStatusRefunded   = "refunded"
	PaymentStatusCancelled  = "cancelled"
)

var ValidPaymentStatuses = []string{
	PaymentStatusPending,
	PaymentStatusProcessing,
	PaymentStatusSuccess,
	PaymentStatusFailed,
	PaymentStatusRefunded,
	PaymentStatusCancelled,
}

// =====================================================
// REFUND REQUEST STATUS
// =====================================================
const (
	RefundStatusPending    = "pending"
	RefundStatusApproved   = "approved"
	RefundStatusRejected   = "rejected"
	RefundStatusProcessing = "processing"
	RefundStatusCompleted  = "completed"
	RefundStatusFailed     = "failed"
)

var ValidRefundStatuses = []string{
	RefundStatusPending,
	RefundStatusApproved,
	RefundStatusRejected,
	RefundStatusProcessing,
	RefundStatusCompleted,
	RefundStatusFailed,
}

// =====================================================
// INTERNAL ERROR CODES
// =====================================================
const (
	// Payment creation errors
	ErrCodePaymentNotFound    = "PAY001"
	ErrCodeOrderAlreadyPaid   = "PAY002"
	ErrCodeRetryLimitExceeded = "PAY003"
	ErrCodeOrderNotPending    = "PAY004"
	ErrCodeInvalidGateway     = "PAY005"

	// Refund errors
	ErrCodeRefundNotAllowed     = "PAY006"
	ErrCodePaymentNotSuccessful = "PAY007"
	ErrCodeOrderCannotRefund    = "PAY008"
	ErrCodeRefundWindowExpired  = "PAY009"
	ErrCodeRefundAlreadyExists  = "PAY010"
	ErrCodeCODNoRefund          = "PAY011"

	// Webhook errors
	ErrCodeInvalidSignature        = "PAY012"
	ErrCodeWebhookAlreadyProcessed = "PAY013"
	ErrCodeWebhookProcessingFailed = "PAY014"

	// Gateway errors
	ErrCodeGatewayTimeout       = "PAY015"
	ErrCodeGatewayUnavailable   = "PAY016"
	ErrCodeInsufficientBalance  = "PAY017"
	ErrCodeCardLocked           = "PAY018"
	ErrCodeOTPExpired           = "PAY019"
	ErrCodeTransactionCancelled = "PAY020"

	// System errors
	ErrCodeUnauthorized   = "PAY021"
	ErrCodeOrderCancelled = "PAY022"
	ErrCodeRefundFailed   = "PAY023"
	ErrCodeInternalError  = "PAY024" // Internal system error
)

// =====================================================
// VNPAY ERROR CODE MAPPING
// =====================================================
var VNPayErrorCodeMap = map[string]struct {
	InternalCode string
	Message      string
}{
	"00": {ErrCodePaymentNotFound, "Transaction successful"},
	"07": {ErrCodeGatewayTimeout, "Transaction timeout - please retry"},
	"09": {PaymentStatusProcessing, "Transaction is being processed"},
	"10": {ErrCodeCardLocked, "Card is locked or restricted"},
	"11": {ErrCodeOTPExpired, "OTP has expired"},
	"12": {ErrCodeCardLocked, "Card is locked"},
	"13": {ErrCodeOTPExpired, "Incorrect OTP entered too many times"},
	"24": {ErrCodeTransactionCancelled, "Transaction cancelled by user"},
	"51": {ErrCodeInsufficientBalance, "Insufficient account balance"},
	"65": {ErrCodeGatewayTimeout, "Bank account limit exceeded"},
	"75": {ErrCodeGatewayTimeout, "Bank is under maintenance"},
	"79": {ErrCodeGatewayTimeout, "Transaction timeout - please retry"},
}

// MapVNPayErrorCode maps VNPay response code to internal code
func MapVNPayErrorCode(vnpCode string) (string, string) {
	if mapping, exists := VNPayErrorCodeMap[vnpCode]; exists {
		return mapping.InternalCode, mapping.Message
	}
	return ErrCodeGatewayUnavailable, "Unknown payment error"
}

// =====================================================
// MOMO ERROR CODE MAPPING
// =====================================================
var MomoErrorCodeMap = map[int]struct {
	InternalCode string
	Message      string
}{
	0:    {ErrCodePaymentNotFound, "Transaction successful"},
	9000: {ErrCodeTransactionCancelled, "Transaction cancelled by user"},
	1001: {ErrCodeInsufficientBalance, "Insufficient balance"},
	1002: {ErrCodeGatewayTimeout, "Transaction timeout"},
	1003: {ErrCodeGatewayUnavailable, "Payment method unavailable"},
	1004: {ErrCodeInvalidGateway, "Invalid payment request"},
	1005: {ErrCodeGatewayTimeout, "Transaction timeout"},
	1006: {ErrCodeCardLocked, "Account locked"},
	4001: {ErrCodeInvalidSignature, "Invalid signature"},
}

// MapMomoErrorCode maps Momo result code to internal code
func MapMomoErrorCode(momoCode int) (string, string) {
	if mapping, exists := MomoErrorCodeMap[momoCode]; exists {
		return mapping.InternalCode, mapping.Message
	}
	return ErrCodeGatewayUnavailable, "Unknown payment error"
}

// =====================================================
// PAYMENT CONFIGURATION
// =====================================================
const (
	// Timeout duration for payment (15 minutes)
	PaymentTimeoutMinutes = 15

	// Max retry attempts
	MaxRetryAttempts = 3

	// Refund window (7 days after delivery)
	RefundWindowDays = 7

	// Default currency
	DefaultCurrency = "VND"
)

// =====================================================
// WEBHOOK EVENTS
// =====================================================
var (
	WebhookEventPaymentSuccess = "payment.success"
	WebhookEventPaymentFailed  = "payment.failed"
	WebhookEventRefundSuccess  = "refund.success"
	WebhookEventRefundFailed   = "refund.failed"
)
