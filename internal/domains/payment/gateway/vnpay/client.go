package vnpay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// VNPAY CLIENT - COMPLETE IMPLEMENTATION
// =====================================================

type Client struct {
	config     *Config
	httpClient *http.Client
}

func NewClient(config *Config) (gateway.VNPayGateway, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid VNPay config: %w", err)
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// =====================================================
// CREATE PAYMENT URL - FIXED
// =====================================================

func (c *Client) CreatePaymentURL(
	ctx context.Context,
	req gateway.VNPayPaymentRequest,
) (string, error) {
	// Validate request
	if req.TransactionRef == "" {
		return "", fmt.Errorf("transaction_ref is required")
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount must be positive")
	}

	// Get client IP from context (injected by middleware)
	// This is critical for VNPay fraud detection and geolocation
	clientIP := "127.0.0.1" // Safe fallback only
	if ip := ctx.Value("client_ip"); ip != nil {
		if ipStr, ok := ip.(string); ok && ipStr != "" {
			clientIP = ipStr
		}
	}

	// ⚠️ VNPay requires IPv4 format - convert IPv6 localhost to IPv4
	if clientIP == "::1" {
		clientIP = "127.0.0.1"
	}

	// Warning: If IP is localhost in production, VNPay might flag as suspicious
	// This should only happen if middleware is not properly configured
	if clientIP == "127.0.0.1" {
		fmt.Printf("⚠️ WARNING: Using localhost IP for VNPay request. Ensure IPExtractorMiddleware is registered.\n")
	}

	// Build parameters
	now := time.Now()
	params := map[string]string{
		"vnp_Version":    c.config.Version,
		"vnp_Command":    c.config.Command,
		"vnp_TmnCode":    c.config.TmnCode,
		"vnp_Amount":     c.formatAmount(req.Amount),
		"vnp_CurrCode":   c.config.CurrCode,
		"vnp_TxnRef":     req.TransactionRef,
		"vnp_OrderInfo":  req.OrderInfo,
		"vnp_OrderType":  "other", // Can be: other, billpayment, topup
		"vnp_Locale":     c.config.Locale,
		"vnp_ReturnUrl":  req.ReturnURL,
		"vnp_IpAddr":     clientIP,
		"vnp_CreateDate": now.Format("20060102150405"),
		// NOTE: vnp_IpnUrl disabled for sandbox - VNPay rejects tunnel/localhost URLs
		// Enable when deploying to production with real public domain
		// "vnp_IpnUrl":     c.config.IPNURL,
		"vnp_ExpireDate": now.Add(30 * time.Minute).Format("20060102150405"),
	}

	// Optional: Bank code (for specific bank selection)
	// params["vnp_BankCode"] = "NCB" // Uncomment to force NCB bank

	// Build payment URL with signature
	paymentURL := BuildPaymentURL(c.config.GetPaymentURL(), params, c.config.HashSecret)

	return paymentURL, nil
}

// formatAmount formats amount for VNPay
// VNPay requires amount in VND (no decimal) * 100
// Example: 100,000 VND -> "10000000"
func (c *Client) formatAmount(amount decimal.Decimal) string {
	// Round to integer (no decimal for VND)
	amountInt := amount.Round(0)

	// Multiply by 100 (VNPay requirement)
	amountInCents := amountInt.Mul(decimal.NewFromInt(100))

	return amountInCents.StringFixed(0)
}

// parseAmount parses VNPay amount back to decimal
// Example: "10000000" -> 100,000 VND
func (c *Client) parseAmount(amountStr string) (decimal.Decimal, error) {
	amountInt, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid amount: %w", err)
	}

	// Divide by 100
	amount := decimal.NewFromInt(amountInt).Div(decimal.NewFromInt(100))

	return amount, nil
}

// =====================================================
// VERIFY SIGNATURE - FIXED
// =====================================================

func (c *Client) VerifySignature(webhookData model.VNPayWebhookRequest) bool {
	// Build params map from webhook struct
	params := map[string]string{
		"vnp_Amount":            webhookData.VnpAmount,
		"vnp_BankCode":          webhookData.VnpBankCode,
		"vnp_BankTranNo":        webhookData.VnpBankTranNo, // ← MISSING before!
		"vnp_CardType":          webhookData.VnpCardType,
		"vnp_OrderInfo":         webhookData.VnpOrderInfo,
		"vnp_PayDate":           webhookData.VnpPayDate,
		"vnp_ResponseCode":      webhookData.VnpResponseCode,
		"vnp_TmnCode":           webhookData.VnpTmnCode,
		"vnp_TransactionNo":     webhookData.VnpTransactionNo,
		"vnp_TransactionStatus": webhookData.VnpTransactionStatus,
		"vnp_TxnRef":            webhookData.VnpTxnRef,
		"vnp_SecureHash":        webhookData.VnpSecureHash,
	}

	// Remove empty fields
	cleanParams := make(map[string]string)
	for k, v := range params {
		if v != "" {
			cleanParams[k] = v
		}
	}

	return VerifySignature(cleanParams, c.config.HashSecret)
}

// =====================================================
// INITIATE REFUND - COMPLETE IMPLEMENTATION
// =====================================================

func (c *Client) InitiateRefund(
	ctx context.Context,
	req gateway.VNPayRefundRequest,
) (*gateway.VNPayRefundResponse, error) {
	// Validate request
	if req.TransactionID == "" {
		return nil, fmt.Errorf("transaction_id is required")
	}
	if req.TransactionDate == "" {
		return nil, fmt.Errorf("transaction_date is required")
	}
	if req.RefundAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("refund_amount must be positive")
	}

	// Generate unique refund transaction ref
	now := time.Now()
	refundTxnRef := fmt.Sprintf("RF%s", now.Format("20060102150405"))
	requestID := fmt.Sprintf("REQ%s", now.Format("20060102150405"))

	// Get IP address
	ipAddr := "127.0.0.1"
	if ip := ctx.Value("client_ip"); ip != nil {
		if ipStr, ok := ip.(string); ok {
			ipAddr = ipStr
		}
	}

	// Build refund request parameters
	params := map[string]string{
		"vnp_RequestId":       requestID,
		"vnp_Version":         c.config.Version,
		"vnp_Command":         "refund",
		"vnp_TmnCode":         c.config.TmnCode,
		"vnp_TransactionType": "02", // 02: Full refund, 03: Partial refund
		"vnp_TxnRef":          refundTxnRef,
		"vnp_Amount":          c.formatAmount(req.Amount),
		"vnp_OrderInfo":       fmt.Sprintf("Hoan tien GD %s", req.TransactionID),
		"vnp_TransactionNo":   req.TransactionID,
		"vnp_TransactionDate": req.TransactionDate,
		"vnp_CreateDate":      now.Format("20060102150405"),
		"vnp_CreateBy":        "admin",
		"vnp_IpAddr":          ipAddr,
	}

	// Generate signature for refund request
	signature := GenerateSignature(params, c.config.HashSecret)
	params["vnp_SecureHash"] = signature

	// Build request body
	requestBody := make(map[string]interface{})
	for k, v := range params {
		requestBody[k] = v
	}

	// Call VNPay refund API
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	refundURL := c.config.GetRefundURL()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", refundURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call VNPay API: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var respData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response code
	responseCode, _ := respData["vnp_ResponseCode"].(string)
	message, _ := respData["vnp_Message"].(string)

	if responseCode != "00" {
		return nil, fmt.Errorf("VNPay refund failed: [%s] %s", responseCode, message)
	}

	// Build response
	refundResponse := &gateway.VNPayRefundResponse{
		RefundTransactionID: refundTxnRef,
		ResponseCode:        responseCode,
		Message:             message,
		RawResponse:         respData,
	}

	return refundResponse, nil
}

func (c *Client) GetReturnURL() string {
	return c.config.ReturnURL
}
