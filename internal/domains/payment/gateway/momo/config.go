package momo

// =====================================================
// MOMO CONFIGURATION
// =====================================================

type Config struct {
	PartnerCode string // Partner code (provided by Momo)
	AccessKey   string // Access key
	SecretKey   string // Secret key for HMAC-SHA256 signature
	APIUrl      string // Momo API endpoint
	ReturnURL   string // Frontend callback URL
	IPNURL      string // Backend webhook URL
}

// NewConfig creates Momo configuration
func NewConfig(partnerCode, accessKey, secretKey, apiURL, returnURL, ipnURL string) *Config {
	return &Config{
		PartnerCode: partnerCode,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		APIUrl:      apiURL,
		ReturnURL:   returnURL,
		IPNURL:      ipnURL,
	}
}

// GetPaymentURL returns payment API endpoint
func (c *Config) GetPaymentURL() string {
	return c.APIUrl + "/v2/gateway/api/create"
}

// GetRefundURL returns refund API endpoint
func (c *Config) GetRefundURL() string {
	return c.APIUrl + "/v2/gateway/api/refund"
}

// =====================================================
// MOMO CONSTANTS
// =====================================================

const (
	// Result codes
	ResultCodeSuccess           = 0
	ResultCodeUserCancelled     = 9000
	ResultCodeInsufficientFunds = 1001
	ResultCodeTimeout           = 1002
	ResultCodeUnavailable       = 1003
	ResultCodeInvalidRequest    = 1004
	ResultCodeTransactionFailed = 1005
	ResultCodeAccountLocked     = 1006
	ResultCodeInvalidSignature  = 4001
)

// GetResultMessage returns Vietnamese message for result code
func GetResultMessage(code int) string {
	messages := map[int]string{
		ResultCodeSuccess:           "Giao dịch thành công",
		ResultCodeUserCancelled:     "Người dùng hủy giao dịch",
		ResultCodeInsufficientFunds: "Số dư tài khoản không đủ",
		ResultCodeTimeout:           "Giao dịch hết hạn",
		ResultCodeUnavailable:       "Phương thức thanh toán không khả dụng",
		ResultCodeInvalidRequest:    "Yêu cầu không hợp lệ",
		ResultCodeTransactionFailed: "Giao dịch thất bại",
		ResultCodeAccountLocked:     "Tài khoản bị khóa",
		ResultCodeInvalidSignature:  "Chữ ký không hợp lệ",
	}

	if msg, exists := messages[code]; exists {
		return msg
	}
	return "Lỗi không xác định"
}
