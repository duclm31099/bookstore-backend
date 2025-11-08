package momo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// MOMO CLIENT IMPLEMENTATION
// =====================================================

type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates new Momo client
func NewClient(config *Config) (gateway.MomoGateway, error) {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// =====================================================
// CREATE PAYMENT URL
// =====================================================

// CreatePaymentURL generates Momo payment URL
func (c *Client) CreatePaymentURL(
	ctx context.Context,
	req gateway.MomoPaymentRequest,
) (string, error) {
	// Step 1: Build request parameters
	requestID := fmt.Sprintf("REQ%d", time.Now().Unix())
	orderID := req.OrderID
	amount := req.Amount.StringFixed(0) // Momo uses integer amount
	orderInfo := req.OrderInfo
	requestType := "captureWallet"
	extraData := ""

	// Step 2: Build signature
	rawSignature := BuildPaymentSignatureString(
		c.config.AccessKey,
		amount,
		extraData,
		c.config.IPNURL,
		orderID,
		orderInfo,
		c.config.PartnerCode,
		c.config.ReturnURL,
		requestID,
		requestType,
	)

	signature := GenerateSignature(rawSignature, c.config.SecretKey)

	// Step 3: Build request body
	requestBody := map[string]interface{}{
		"partnerCode": c.config.PartnerCode,
		"accessKey":   c.config.AccessKey,
		"requestId":   requestID,
		"amount":      amount,
		"orderId":     orderID,
		"orderInfo":   orderInfo,
		"redirectUrl": c.config.ReturnURL,
		"ipnUrl":      c.config.IPNURL,
		"requestType": requestType,
		"extraData":   extraData,
		"signature":   signature,
		"lang":        "vi",
	}

	// Step 4: Call Momo API
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.GetPaymentURL(), bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call Momo API: %w", err)
	}
	defer resp.Body.Close()

	// Step 5: Parse response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &respData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Step 6: Check result code
	resultCode, _ := respData["resultCode"].(float64)
	if int(resultCode) != ResultCodeSuccess {
		message, _ := respData["message"].(string)
		return "", fmt.Errorf("Momo API error: %s", message)
	}

	// Step 7: Extract payment URL
	payURL, ok := respData["payUrl"].(string)
	if !ok {
		return "", fmt.Errorf("payUrl not found in response")
	}

	return payURL, nil
}

// =====================================================
// VERIFY SIGNATURE
// =====================================================

// VerifySignature verifies Momo webhook signature
func (c *Client) VerifySignature(webhookData model.MomoWebhookRequest) bool {
	return VerifyWebhookSignature(
		webhookData.PartnerCode,
		webhookData.OrderID,
		webhookData.RequestID,
		fmt.Sprintf("%d", webhookData.Amount),
		webhookData.OrderInfo,
		webhookData.OrderType,
		webhookData.TransID,
		webhookData.ResultCode,
		webhookData.Message,
		webhookData.PayType,
		fmt.Sprintf("%d", webhookData.ResponseTime),
		webhookData.ExtraData,
		webhookData.Signature,
		c.config.SecretKey,
	)
}

// =====================================================
// INITIATE REFUND
// =====================================================

// InitiateRefund initiates refund via Momo API
func (c *Client) InitiateRefund(
	ctx context.Context,
	req gateway.MomoRefundRequest,
) (*gateway.MomoRefundResponse, error) {
	// TODO: Implement Momo refund API
	// Similar structure to payment but different endpoint and parameters
	return nil, fmt.Errorf("Momo refund not implemented")
}
