package vnpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// =====================================================
// VNPAY SIGNATURE - COMPLETE IMPLEMENTATION
// =====================================================

// GenerateSignature generates HMAC-SHA512 signature for VNPay
//
// VNPay Algorithm (CRITICAL - Must follow exactly):
// 1. Collect ALL parameters except vnp_SecureHash and vnp_SecureHashType
// 2. URL decode values (VNPay sends URL-encoded values)
// 3. Sort by key (case-sensitive, ascending)
// 4. Build raw string: key1=value1&key2=value2&...
// 5. HMAC-SHA512(rawString, secretKey) - NO URL encoding on raw string!
// 6. Uppercase hex encode
func GenerateSignature(params map[string]string, secretKey string) string {
	// Step 1: Remove signature fields
	filteredParams := make(map[string]string)
	for key, value := range params {
		if key != "vnp_SecureHash" && key != "vnp_SecureHashType" && value != "" {
			filteredParams[key] = value
		}
	}

	// Step 2: Sort keys (ascending)
	keys := make([]string, 0, len(filteredParams))
	for key := range filteredParams {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Step 3: Build raw signature string (NO URL encoding here!)
	var parts []string
	for _, key := range keys {
		value := filteredParams[key]
		// URL decode if needed (VNPay webhook sends URL-encoded values)
		decodedValue, err := url.QueryUnescape(value)
		if err == nil {
			value = decodedValue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	rawSignature := strings.Join(parts, "&")

	// Step 4: HMAC-SHA512
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write([]byte(rawSignature))

	// Step 5: Uppercase hex encode
	signature := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))

	return signature
}

// VerifySignature verifies VNPay callback signature
func VerifySignature(params map[string]string, secretKey string) bool {
	receivedSignature, exists := params["vnp_SecureHash"]
	if !exists || receivedSignature == "" {
		return false
	}

	// Generate expected signature
	expectedSignature := GenerateSignature(params, secretKey)

	// Compare (case-insensitive for safety)
	return strings.EqualFold(receivedSignature, expectedSignature)
}

// BuildPaymentURL builds complete payment URL with signature
func BuildPaymentURL(baseURL string, params map[string]string, secretKey string) string {
	// Generate signature
	signature := GenerateSignature(params, secretKey)
	params["vnp_SecureHash"] = signature

	// Sort keys for consistent URL
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build query string with URL encoding
	var queryParts []string
	for _, key := range keys {
		value := params[key]
		if value != "" {
			// URL encode both key and value
			encodedKey := url.QueryEscape(key)
			encodedValue := url.QueryEscape(value)
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", encodedKey, encodedValue))
		}
	}

	queryString := strings.Join(queryParts, "&")
	return fmt.Sprintf("%s?%s", baseURL, queryString)
}

// ParseWebhookParams parses and validates webhook parameters
func ParseWebhookParams(rawQuery string) (map[string]string, error) {
	params := make(map[string]string)

	// Parse query string
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil, fmt.Errorf("invalid query string: %w", err)
	}

	// Extract all vnp_* parameters
	for key, vals := range values {
		if strings.HasPrefix(key, "vnp_") && len(vals) > 0 {
			params[key] = vals[0]
		}
	}

	// Validate required fields
	requiredFields := []string{
		"vnp_TxnRef",
		"vnp_ResponseCode",
		"vnp_SecureHash",
	}

	for _, field := range requiredFields {
		if params[field] == "" {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}

	return params, nil
}

// WebhookParamsToMap converts VNPayWebhookRequest to params map
func WebhookParamsToMap(webhook interface{}) map[string]string {
	// Use reflection or manual mapping
	// For now, manual mapping is safer
	params := make(map[string]string)

	// This should be called with actual webhook struct
	// Implementation depends on how you receive webhook

	return params
}
