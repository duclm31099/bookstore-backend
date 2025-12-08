package vnpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
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

// BuildPaymentURL builds VNPay payment URL with correct signature
//
// VNPay Algorithm (matching PHP reference exactly):
// 1. Sort parameters by key (ascending, case-sensitive)
// 2. Build hash string: urlencode(key1)=urlencode(value1)&urlencode(key2)=urlencode(value2)
// 3. HMAC-SHA512(hashString, secretKey)
// 4. Uppercase hex encode
// 5. Append to URL
func BuildPaymentURL(baseURL string, params map[string]string, hashSecret string) string {
	// Step 1: Sort keys alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "vnp_SecureHash" && k != "vnp_SecureHashType" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Step 2: Build hash data string (URL-encoded like PHP urlencode)
	// PHP urlencode: spaces -> +, special chars -> %XX
	var hashParts []string
	var queryParts []string

	for _, k := range keys {
		v := params[k]
		if v != "" {
			// URL encode key and value (PHP style: spaces become +)
			encodedKey := phpURLEncode(k)
			encodedValue := phpURLEncode(v)

			hashParts = append(hashParts, encodedKey+"="+encodedValue)
			queryParts = append(queryParts, encodedKey+"="+encodedValue)
		}
	}

	hashData := strings.Join(hashParts, "&")
	queryString := strings.Join(queryParts, "&")

	// Debug logging
	log.Println("=== VNPay Hash Debug ===")

	// Step 3: Create HMAC-SHA512 hash
	h := hmac.New(sha512.New, []byte(hashSecret))
	h.Write([]byte(hashData))

	// Step 4: Uppercase hex encode
	secureHash := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))

	log.Printf("Generated Hash: %s", secureHash)
	log.Println("========================")

	// Step 5: Build final URL
	if !strings.HasSuffix(baseURL, "/vpcpay.html") {
		baseURL += "/vpcpay.html"
	}

	finalURL := fmt.Sprintf("%s?%s&vnp_SecureHash=%s", baseURL, queryString, secureHash)
	log.Printf("Final URL: %s", finalURL)

	return finalURL
}

// phpURLEncode encodes string like PHP's urlencode()
// PHP urlencode: spaces become '+', special chars become %XX
// Go url.QueryEscape: spaces become '%20'
func phpURLEncode(s string) string {
	// First use Go's QueryEscape, then replace %20 with +
	encoded := url.QueryEscape(s)
	// Replace %20 with + to match PHP's urlencode behavior
	return strings.ReplaceAll(encoded, "%20", "+")
}

func createSecureHash(data, secret string) string {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write([]byte(data))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
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
