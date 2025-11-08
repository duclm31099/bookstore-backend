package momo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// =====================================================
// MOMO SIGNATURE GENERATION & VERIFICATION
// =====================================================

// GenerateSignature generates HMAC-SHA256 signature for Momo request
//
// Algorithm (different from VNPay):
// 1. Build raw signature string (specific order, not sorted)
// 2. HMAC-SHA256(rawString, secretKey)
// 3. Hex encode result
func GenerateSignature(rawSignature, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(rawSignature))
	signature := hex.EncodeToString(mac.Sum(nil))
	return signature
}

// BuildPaymentSignatureString builds signature string for payment request
// Format: accessKey=$accessKey&amount=$amount&extraData=$extraData&ipnUrl=$ipnUrl&orderId=$orderId&orderInfo=$orderInfo&partnerCode=$partnerCode&redirectUrl=$redirectUrl&requestId=$requestId&requestType=$requestType
func BuildPaymentSignatureString(
	accessKey, amount, extraData, ipnUrl, orderId, orderInfo,
	partnerCode, redirectUrl, requestId, requestType string,
) string {
	parts := []string{
		fmt.Sprintf("accessKey=%s", accessKey),
		fmt.Sprintf("amount=%s", amount),
		fmt.Sprintf("extraData=%s", extraData),
		fmt.Sprintf("ipnUrl=%s", ipnUrl),
		fmt.Sprintf("orderId=%s", orderId),
		fmt.Sprintf("orderInfo=%s", orderInfo),
		fmt.Sprintf("partnerCode=%s", partnerCode),
		fmt.Sprintf("redirectUrl=%s", redirectUrl),
		fmt.Sprintf("requestId=%s", requestId),
		fmt.Sprintf("requestType=%s", requestType),
	}
	return strings.Join(parts, "&")
}

// VerifyWebhookSignature verifies Momo webhook signature
func VerifyWebhookSignature(
	partnerCode, orderId, requestId, amount, orderInfo, orderType,
	transId string, resultCode int, message, payType, responseTime,
	extraData, receivedSignature, secretKey string,
) bool {
	// Build raw signature string for webhook
	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%s&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%s&resultCode=%d&transId=%s",
		"", // accessKey not included in webhook signature
		amount,
		extraData,
		message,
		orderId,
		orderInfo,
		orderType,
		partnerCode,
		payType,
		requestId,
		responseTime,
		resultCode,
		transId,
	)

	expectedSignature := GenerateSignature(rawSignature, secretKey)
	return strings.EqualFold(receivedSignature, expectedSignature)
}
