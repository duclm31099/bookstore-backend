package mock

import (
	"context"
	"fmt"
	"time"

	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// MOCK VNPAY GATEWAY FOR TESTING
// =====================================================

type MockVNPayGateway struct {
	returnURL         string
	shouldFailPayment bool
	shouldFailRefund  bool
}

func NewMockVNPayGateway(returnURL string) gateway.VNPayGateway {
	return &MockVNPayGateway{
		returnURL: returnURL,
	}
}

func (m *MockVNPayGateway) CreatePaymentURL(
	ctx context.Context,
	req gateway.VNPayPaymentRequest,
) (string, error) {
	if m.shouldFailPayment {
		return "", fmt.Errorf("mock payment creation failed")
	}

	// Generate mock payment URL
	mockURL := fmt.Sprintf(
		"https://mock-vnpay.com/payment?txnRef=%s&amount=%s&returnUrl=%s",
		req.TransactionRef,
		req.Amount.StringFixed(0),
		req.ReturnURL,
	)

	return mockURL, nil
}

func (m *MockVNPayGateway) VerifySignature(webhookData model.VNPayWebhookRequest) bool {
	// Mock always returns true for testing
	// In real tests, you can control this behavior
	return true
}

func (m *MockVNPayGateway) InitiateRefund(
	ctx context.Context,
	req gateway.VNPayRefundRequest,
) (*gateway.VNPayRefundResponse, error) {
	if m.shouldFailRefund {
		return nil, fmt.Errorf("mock refund failed")
	}

	// Generate mock refund response
	refundTxnID := fmt.Sprintf("MOCK_REFUND_%d", time.Now().Unix())

	return &gateway.VNPayRefundResponse{
		RefundTransactionID: refundTxnID,
		ResponseCode:        "00",
		Message:             "Mock refund success",
		RawResponse: map[string]interface{}{
			"vnp_ResponseCode": "00",
			"vnp_Message":      "Success",
			"vnp_TxnRef":       refundTxnID,
		},
	}, nil
}

func (m *MockVNPayGateway) GetReturnURL() string {
	return m.returnURL
}

// SetFailPayment sets whether payment creation should fail
func (m *MockVNPayGateway) SetFailPayment(shouldFail bool) {
	m.shouldFailPayment = shouldFail
}

// SetFailRefund sets whether refund should fail
func (m *MockVNPayGateway) SetFailRefund(shouldFail bool) {
	m.shouldFailRefund = shouldFail
}
