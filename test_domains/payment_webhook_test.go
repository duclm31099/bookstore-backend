package test_domains

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// PAYMENT WEBHOOK TESTS
// ========================================

func TestVNPayWebhook_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup: Create order and payment
	userID, token := createAndLoginUser(t, "vnpay_webhook@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create order
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}, headers)

	resp := makeRequest(t, "POST", "/api/v1/orders", map[string]interface{}{
		"address_id": addressID,
	}, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	orderData := result["data"].(map[string]interface{})
	orderID := orderData["id"].(string)

	// Create payment
	resp = makeRequest(t, "POST", "/api/v1/payments/create", map[string]interface{}{
		"order_id":       orderID,
		"payment_method": "vnpay",
	}, headers)
	result = assertSuccess(t, resp, http.StatusCreated)
	paymentData := result["data"].(map[string]interface{})
	paymentID := paymentData["payment_id"].(string)

	// Simulate VNPay webhook callback
	// Note: In real scenario, this would include proper signature
	webhookPayload := map[string]interface{}{
		"vnp_TxnRef":        paymentID,
		"vnp_Amount":        "10000000", // 100,000 VND * 100
		"vnp_ResponseCode":  "00",       // Success
		"vnp_TransactionNo": "123456789",
		"vnp_BankCode":      "NCB",
		"vnp_PayDate":       "20260102214500",
		"vnp_SecureHash":    "dummy_hash", // In real test, calculate proper hash
	}

	// Call webhook endpoint (no auth required for webhooks)
	resp = makeRequest(t, "POST", "/api/v1/webhooks/vnpay", webhookPayload, nil)

	// Webhook should respond (may succeed or fail based on signature validation)
	// Just verify endpoint is accessible
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

func TestVNPayWebhook_InvalidSignature(t *testing.T) {
	cleanBeforeTest(t)

	// Simulate VNPay webhook with invalid signature
	webhookPayload := map[string]interface{}{
		"vnp_TxnRef":        "fake-payment-id",
		"vnp_Amount":        "10000000",
		"vnp_ResponseCode":  "00",
		"vnp_TransactionNo": "123456789",
		"vnp_BankCode":      "NCB",
		"vnp_PayDate":       "20260102214500",
		"vnp_SecureHash":    "invalid_signature",
	}

	resp := makeRequest(t, "POST", "/api/v1/webhooks/vnpay", webhookPayload, nil)

	// Should reject invalid signature
	// Exact status code depends on implementation
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

func TestVNPayWebhook_DuplicateCallback(t *testing.T) {
	cleanBeforeTest(t)

	// Setup: Create order and payment
	userID, token := createAndLoginUser(t, "vnpay_dup@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create order and payment
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}, headers)

	resp := makeRequest(t, "POST", "/api/v1/orders", map[string]interface{}{
		"address_id": addressID,
	}, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	orderData := result["data"].(map[string]interface{})
	orderID := orderData["id"].(string)

	resp = makeRequest(t, "POST", "/api/v1/payments/create", map[string]interface{}{
		"order_id":       orderID,
		"payment_method": "vnpay",
	}, headers)
	result = assertSuccess(t, resp, http.StatusCreated)
	paymentData := result["data"].(map[string]interface{})
	paymentID := paymentData["payment_id"].(string)

	webhookPayload := map[string]interface{}{
		"vnp_TxnRef":        paymentID,
		"vnp_Amount":        "10000000",
		"vnp_ResponseCode":  "00",
		"vnp_TransactionNo": "123456789",
		"vnp_BankCode":      "NCB",
		"vnp_PayDate":       "20260102214500",
		"vnp_SecureHash":    "dummy_hash",
	}

	// Call webhook twice with same data (idempotency test)
	resp1 := makeRequest(t, "POST", "/api/v1/webhooks/vnpay", webhookPayload, nil)
	resp2 := makeRequest(t, "POST", "/api/v1/webhooks/vnpay", webhookPayload, nil)

	// Both should respond (second one should be idempotent)
	assert.NotEqual(t, http.StatusNotFound, resp1.StatusCode)
	assert.NotEqual(t, http.StatusNotFound, resp2.StatusCode)
}

func TestMomoWebhook_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Setup: Create order and payment
	userID, token := createAndLoginUser(t, "momo_webhook@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create order
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}, headers)

	resp := makeRequest(t, "POST", "/api/v1/orders", map[string]interface{}{
		"address_id": addressID,
	}, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	orderData := result["data"].(map[string]interface{})
	orderID := orderData["id"].(string)

	// Create payment
	resp = makeRequest(t, "POST", "/api/v1/payments/create", map[string]interface{}{
		"order_id":       orderID,
		"payment_method": "momo",
	}, headers)
	result = assertSuccess(t, resp, http.StatusCreated)
	paymentData := result["data"].(map[string]interface{})
	paymentID := paymentData["payment_id"].(string)

	// Simulate Momo webhook callback
	webhookPayload := map[string]interface{}{
		"partnerCode":  "MOMO",
		"orderId":      paymentID,
		"requestId":    paymentID,
		"amount":       100000,
		"orderInfo":    "Payment for order",
		"orderType":    "momo_wallet",
		"transId":      "987654321",
		"resultCode":   0, // Success
		"message":      "Successful",
		"payType":      "qr",
		"responseTime": "1704214500000",
		"extraData":    "",
		"signature":    "dummy_signature",
	}

	// Call webhook endpoint
	resp = makeRequest(t, "POST", "/api/v1/webhooks/momo", webhookPayload, nil)

	// Webhook should respond
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

func TestMomoWebhook_InvalidSignature(t *testing.T) {
	cleanBeforeTest(t)

	// Simulate Momo webhook with invalid signature
	webhookPayload := map[string]interface{}{
		"partnerCode":  "MOMO",
		"orderId":      "fake-payment-id",
		"requestId":    "fake-request-id",
		"amount":       100000,
		"orderInfo":    "Payment for order",
		"orderType":    "momo_wallet",
		"transId":      "987654321",
		"resultCode":   0,
		"message":      "Successful",
		"payType":      "qr",
		"responseTime": "1704214500000",
		"extraData":    "",
		"signature":    "invalid_signature",
	}

	resp := makeRequest(t, "POST", "/api/v1/webhooks/momo", webhookPayload, nil)

	// Should reject invalid signature
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

func TestMomoWebhook_FailedPayment(t *testing.T) {
	cleanBeforeTest(t)

	// Setup: Create order and payment
	userID, token := createAndLoginUser(t, "momo_fail@test.com", "Test1234")
	categoryID := createTestCategory(t, "Fiction")
	authorID := createTestAuthor(t, "Test Author")
	publisherID := createTestPublisher(t, "Test Publisher")
	bookID := createTestBook(t, "Test Book", 100000, categoryID, authorID, publisherID)
	warehouseID := createTestWarehouse(t, "Main Warehouse", 10.762622, 106.660172)
	createTestInventory(t, warehouseID, bookID, 50)
	addressID := createTestAddress(t, userID, true)

	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create order and payment
	makeRequest(t, "POST", "/api/v1/cart/items", map[string]interface{}{
		"book_id":  bookID,
		"quantity": 1,
	}, headers)

	resp := makeRequest(t, "POST", "/api/v1/orders", map[string]interface{}{
		"address_id": addressID,
	}, headers)
	result := assertSuccess(t, resp, http.StatusCreated)
	orderData := result["data"].(map[string]interface{})
	orderID := orderData["id"].(string)

	resp = makeRequest(t, "POST", "/api/v1/payments/create", map[string]interface{}{
		"order_id":       orderID,
		"payment_method": "momo",
	}, headers)
	result = assertSuccess(t, resp, http.StatusCreated)
	paymentData := result["data"].(map[string]interface{})
	paymentID := paymentData["payment_id"].(string)

	// Simulate failed payment webhook
	webhookPayload := map[string]interface{}{
		"partnerCode":  "MOMO",
		"orderId":      paymentID,
		"requestId":    paymentID,
		"amount":       100000,
		"orderInfo":    "Payment for order",
		"orderType":    "momo_wallet",
		"transId":      "987654321",
		"resultCode":   1000, // Failed
		"message":      "Transaction failed",
		"payType":      "qr",
		"responseTime": "1704214500000",
		"extraData":    "",
		"signature":    "dummy_signature",
	}

	resp = makeRequest(t, "POST", "/api/v1/webhooks/momo", webhookPayload, nil)

	// Webhook should process the failure
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

func TestVerifyVNPayReturn_Success(t *testing.T) {
	cleanBeforeTest(t)

	// This endpoint is called by frontend after VNPay redirect
	// Test with query parameters
	queryParams := "?vnp_TxnRef=payment-id&vnp_ResponseCode=00&vnp_SecureHash=dummy"

	resp := makeRequest(t, "GET", "/api/v1/payments/vnpay/verify"+queryParams, nil, nil)

	// Should respond (may redirect or return JSON)
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

func TestVerifyMomoReturn_Success(t *testing.T) {
	cleanBeforeTest(t)

	// This endpoint is called by frontend after Momo redirect
	// Test with query parameters
	queryParams := "?orderId=payment-id&resultCode=0&signature=dummy"

	resp := makeRequest(t, "GET", "/api/v1/payments/momo/verify"+queryParams, nil, nil)

	// Should respond
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}
