package vnpay

import (
	"fmt"
)

// =====================================================
// VNPAY CONFIGURATION
// =====================================================

type Config struct {
	TmnCode    string // Merchant code (provided by VNPay)
	HashSecret string // Secret key for HMAC-SHA512 signature
	APIUrl     string // VNPay payment gateway URL
	ReturnURL  string // Frontend callback URL
	IPNURL     string // Backend webhook URL
	Version    string // VNPay API version (default: "2.1.0")
	Command    string // Command type (default: "pay")
	CurrCode   string // Currency code (default: "VND")
	Locale     string // Language (default: "vn")
}

// NewConfig creates VNPay configuration
func NewConfig(tmnCode, hashSecret, apiURL, returnURL, ipnURL string) *Config {
	return &Config{
		TmnCode:    tmnCode,
		HashSecret: hashSecret,
		APIUrl:     apiURL,
		ReturnURL:  returnURL,
		IPNURL:     ipnURL,
		Version:    "2.1.0",
		Command:    "pay",
		CurrCode:   "VND",
		Locale:     "vn",
	}
}

// Validate validates configuration
func (c *Config) Validate() error {
	if c.TmnCode == "" {
		return fmt.Errorf("VNPay TmnCode is required")
	}
	if c.HashSecret == "" {
		return fmt.Errorf("VNPay HashSecret is required")
	}
	if c.APIUrl == "" {
		return fmt.Errorf("VNPay APIUrl is required")
	}
	if c.ReturnURL == "" {
		return fmt.Errorf("VNPay ReturnURL is required")
	}
	if c.IPNURL == "" {
		return fmt.Errorf("VNPay IPNURL is required")
	}
	return nil
}

// GetPaymentURL returns full payment URL
func (c *Config) GetPaymentURL() string {
	return c.APIUrl + "/vpcpay.html"
}

// GetRefundURL returns refund API URL
func (c *Config) GetRefundURL() string {
	return c.APIUrl + "/merchant_webapi/api/transaction"
}

// =====================================================
// VNPAY CONSTANTS
// =====================================================

const (
	// Response codes
	ResponseCodeSuccess               = "00"
	ResponseCodeTransactionTimeout    = "07"
	ResponseCodeTransactionProcessing = "09"
	ResponseCodeCardLocked            = "10"
	ResponseCodeOTPExpired            = "11"
	ResponseCodeIncorrectOTP          = "13"
	ResponseCodeUserCancelled         = "24"
	ResponseCodeInsufficientBalance   = "51"
	ResponseCodeLimitExceeded         = "65"
	ResponseCodeBankMaintenance       = "75"
	ResponseCodeTimeout               = "79"
)

// GetResponseMessage returns Vietnamese message for response code
func GetResponseMessage(code string) string {
	messages := map[string]string{
		ResponseCodeSuccess:               "Giao dịch thành công",
		ResponseCodeTransactionTimeout:    "Giao dịch hết hạn (timeout)",
		ResponseCodeTransactionProcessing: "Giao dịch đang xử lý",
		ResponseCodeCardLocked:            "Thẻ bị khóa",
		ResponseCodeOTPExpired:            "Mã OTP hết hạn",
		ResponseCodeIncorrectOTP:          "OTP không chính xác (nhập sai quá số lần)",
		ResponseCodeUserCancelled:         "Người dùng hủy giao dịch",
		ResponseCodeInsufficientBalance:   "Số dư tài khoản không đủ",
		ResponseCodeLimitExceeded:         "Vượt quá hạn mức thanh toán",
		ResponseCodeBankMaintenance:       "Ngân hàng đang bảo trì",
		ResponseCodeTimeout:               "Giao dịch hết hạn (timeout)",
	}

	if msg, exists := messages[code]; exists {
		return msg
	}
	return "Lỗi không xác định"
}
