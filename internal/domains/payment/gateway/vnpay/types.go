package vnpay

// =====================================================
// VNPAY REQUEST/RESPONSE TYPES
// =====================================================

// WebhookRequest represents complete VNPay IPN callback
type WebhookRequest struct {
	VnpAmount            string `form:"vnp_Amount" json:"vnp_Amount"`
	VnpBankCode          string `form:"vnp_BankCode" json:"vnp_BankCode"`
	VnpBankTranNo        string `form:"vnp_BankTranNo" json:"vnp_BankTranNo"` // ← Important!
	VnpCardType          string `form:"vnp_CardType" json:"vnp_CardType"`
	VnpOrderInfo         string `form:"vnp_OrderInfo" json:"vnp_OrderInfo"`
	VnpPayDate           string `form:"vnp_PayDate" json:"vnp_PayDate"`
	VnpResponseCode      string `form:"vnp_ResponseCode" json:"vnp_ResponseCode"`
	VnpTmnCode           string `form:"vnp_TmnCode" json:"vnp_TmnCode"`
	VnpTransactionNo     string `form:"vnp_TransactionNo" json:"vnp_TransactionNo"`
	VnpTransactionStatus string `form:"vnp_TransactionStatus" json:"vnp_TransactionStatus"`
	VnpTxnRef            string `form:"vnp_TxnRef" json:"vnp_TxnRef"`
	VnpSecureHash        string `form:"vnp_SecureHash" json:"vnp_SecureHash"`
}

// PaymentResponse represents VNPay payment creation response
type PaymentResponse struct {
	PaymentURL string
	TxnRef     string
	ExpiresAt  string
}

// RefundRequest represents VNPay refund request
type RefundRequest struct {
	RequestID       string
	Version         string
	Command         string
	TmnCode         string
	TransactionType string
	TxnRef          string
	Amount          string
	OrderInfo       string
	TransactionNo   string
	TransactionDate string
	CreateDate      string
	CreateBy        string
	IpAddr          string
}

// RefundResponse represents VNPay refund response
type RefundResponse struct {
	ResponseCode  string
	Message       string
	TxnRef        string
	Amount        string
	BankCode      string
	TransactionNo string
}
