package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/payment/model"
	"bookstore-backend/internal/domains/payment/service"
	"bookstore-backend/internal/shared/response"
	res "bookstore-backend/internal/shared/response"
)

type PaymentHandler struct {
	paymentService service.PaymentService
	refundService  service.RefundInterface
}

// NewPaymentHandler creates new payment handler
func NewPaymentHandler(
	paymentService service.PaymentService,
	refundService service.RefundInterface,
) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		refundService:  refundService,
	}
}

// =====================================================
// USER REFUND ENDPOINTS
// =====================================================

// RequestRefund creates refund request
// POST /api/v1/payments/:payment_id/refund-request
func (h *PaymentHandler) RequestRefund(c *gin.Context) {
	// Step 1: Get user ID
	userIDStr, err := getUserID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	// Step 2: Get payment ID from URL
	paymentIDStr := c.Param("payment_id")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_PAYMENT_ID", "Invalid payment ID")
		return
	}

	// Step 3: Bind request body
	var req model.CreateRefundRequestDTO
	if err := bindJSON(c, &req); err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 4: Validate request
	if err := req.Validate(); err != nil {
		res.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 5: Call service
	response, err := h.refundService.RequestRefund(c.Request.Context(), userID, paymentID, req)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 6: Return response
	res.Success(c, http.StatusCreated, "Success", response)
}

// GetRefundStatus gets refund request status
// GET /api/v1/payments/:payment_id/refund-request
func (h *PaymentHandler) GetRefundStatus(c *gin.Context) {
	// Step 1: Get user ID
	userIDStr, err := getUserID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	// Step 2: Get payment ID from URL
	paymentIDStr := c.Param("payment_id")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_PAYMENT_ID", "Invalid payment ID")
		return
	}

	// Step 3: Call service
	response, err := h.refundService.GetRefundStatus(c.Request.Context(), userID, paymentID)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 4: Return response
	res.Success(c, http.StatusOK, "Success", response)
}

// =====================================================
// USER PAYMENT ENDPOINTS
// =====================================================

// CreatePayment creates new payment transaction
// POST /api/v1/payments/create
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	// Step 1: Get user ID from context
	userIDStr, err := getUserID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	// Step 2: Bind request body
	var req model.CreatePaymentRequest
	if err := bindJSON(c, &req); err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 3: Validate request
	if err := req.Validate(); err != nil {
		res.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 4: Call service
	response, err := h.paymentService.CreatePayment(c.Request.Context(), userID, req)
	if err != nil {
		// Map service errors to HTTP status codes
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return success response
	res.Success(c, http.StatusCreated, "OK", response)
}

// GetPaymentStatus gets payment transaction status
// GET /api/v1/payments/:payment_id
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	// Step 1: Get user ID
	userIDStr, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	// Step 2: Get payment ID from URL
	paymentIDStr := c.Param("payment_id")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_PAYMENT_ID", "Invalid payment ID")
		return
	}

	// Step 3: Call service
	response, err := h.paymentService.GetPaymentStatus(c.Request.Context(), userID, paymentID)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 4: Return response
	res.Success(c, http.StatusOK, "OK", response)
}

// ListUserPayments lists payments for current user
// GET /api/v1/payments
func (h *PaymentHandler) ListUserPayments(c *gin.Context) {
	// Step 1: Get user ID
	userIDStr, err := getUserID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	// Step 2: Bind query parameters
	var req model.ListPaymentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// Step 3: Set defaults
	if err := req.Validate(); err != nil {
		res.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 4: Call service
	response, err := h.paymentService.ListUserPayments(c.Request.Context(), userID, req)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return response
	res.Success(c, http.StatusOK, "OK", response)
}

// =====================================================
// ERROR MAPPING HELPER
// =====================================================

func mapPaymentError(err error) (statusCode int, errorCode string) {
	// Default
	statusCode = http.StatusInternalServerError
	errorCode = "INTERNAL_ERROR"

	// Check if it's a PaymentError
	if paymentErr, ok := err.(*model.PaymentError); ok {
		errorCode = paymentErr.Code

		// Map error codes to HTTP status codes
		switch paymentErr.Code {
		case model.ErrCodePaymentNotFound:
			statusCode = http.StatusNotFound
		case model.ErrCodeOrderAlreadyPaid:
			statusCode = http.StatusConflict
		case model.ErrCodeRetryLimitExceeded:
			statusCode = http.StatusTooManyRequests
		case model.ErrCodeOrderNotPending:
			statusCode = http.StatusBadRequest
		case model.ErrCodeInvalidGateway:
			statusCode = http.StatusBadRequest
		case model.ErrCodeUnauthorized:
			statusCode = http.StatusUnauthorized
		case model.ErrCodeGatewayTimeout, model.ErrCodeGatewayUnavailable:
			statusCode = http.StatusServiceUnavailable
		default:
			statusCode = http.StatusInternalServerError
		}
	}

	return statusCode, errorCode
}

// =====================================================
// PAYMENT HANDLER STRUCT
// =====================================================

// =====================================================
// HELPER FUNCTIONS
// =====================================================

// getUserID extracts user ID from JWT claims in context
func getUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", fmt.Errorf("user_id not found in context")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("invalid user_id type")
	}

	return userIDStr, nil
}

// getAdminID extracts admin ID from JWT claims
func getAdminID(c *gin.Context) (string, error) {
	// Same as getUserID for now
	// In future, you might have separate admin claims
	return getUserID(c)
}

// bindJSON binds JSON request body and validates
func bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

// VNPayWebhook handles VNPay IPN callback
// GET/POST /api/v1/webhooks/vnpay
func (h *PaymentHandler) VNPayWebhook(c *gin.Context) {
	// Step 1: Parse webhook data from query parameters
	var webhookData model.VNPayWebhookRequest

	// VNPay sends data as query parameters
	if err := c.ShouldBindQuery(&webhookData); err != nil {
		// Log error but still return 200 to VNPay
		c.JSON(http.StatusOK, gin.H{
			"RspCode": "99",
			"Message": "Invalid request format",
		})
		return
	}

	// Step 2: Process webhook (async processing handled in service)
	err := h.paymentService.ProcessVNPayWebhook(c.Request.Context(), webhookData)

	// Step 3: Return response to VNPay (MUST be fast, < 3 seconds)
	if err != nil {
		// Even if processing fails, acknowledge webhook
		// Failed webhooks will be retried by background job
		c.JSON(http.StatusOK, gin.H{
			"RspCode": "99",
			"Message": fmt.Sprintf("Processing error: %v", err),
		})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"RspCode": "00",
		"Message": "Success",
	})
}

// MomoWebhook handles Momo IPN callback
// POST /api/v1/webhooks/momo
func (h *PaymentHandler) MomoWebhook(c *gin.Context) {
	// Step 1: Parse webhook data from JSON body
	var webhookData model.MomoWebhookRequest

	if err := c.ShouldBindJSON(&webhookData); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"resultCode": 99,
			"message":    "Invalid request format",
		})
		return
	}

	// Step 2: Process webhook
	err := h.paymentService.ProcessMomoWebhook(c.Request.Context(), webhookData)

	// Step 3: Return response to Momo
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"resultCode": 99,
			"message":    fmt.Sprintf("Processing error: %v", err),
		})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"resultCode": 0,
		"message":    "Success",
	})
}
func (h *PaymentHandler) AdminListPayments(c *gin.Context) {
	// Step 1: Verify admin access (done by middleware)

	// Step 2: Bind query parameters
	var req model.AdminListPaymentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// Step 3: Validate request
	if err := req.Validate(); err != nil {
		res.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 4: Call service
	response, err := h.paymentService.AdminListPayments(c.Request.Context(), req)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return response
	res.Success(c, http.StatusOK, "OK", response)
}

// AdminGetPaymentDetail gets detailed payment info
// GET /api/v1/admin/payments/:payment_id
func (h *PaymentHandler) AdminGetPaymentDetail(c *gin.Context) {
	// Step 1: Get payment ID from URL
	paymentIDStr := c.Param("payment_id")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_PAYMENT_ID", "Invalid payment ID")
		return
	}

	// Step 2: Call service
	response, err := h.paymentService.AdminGetPaymentDetail(c.Request.Context(), paymentID)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 3: Return response
	res.Success(c, http.StatusOK, "OK", response)
}

// AdminReconcilePayment manually updates payment status
// POST /api/v1/admin/payments/:payment_id/reconcile
func (h *PaymentHandler) AdminReconcilePayment(c *gin.Context) {
	// Step 1: Get admin ID
	adminIDStr, err := getAdminID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_ADMIN_ID", "Invalid admin ID")
		return
	}

	// Step 2: Get payment ID
	paymentIDStr := c.Param("payment_id")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_PAYMENT_ID", "Invalid payment ID")
		return
	}

	// Step 3: Bind request body
	var req model.ManualReconciliationRequest
	if err := bindJSON(c, &req); err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 4: Validate request
	if err := req.Validate(); err != nil {
		res.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 5: Call service
	err = h.paymentService.AdminReconcilePayment(c.Request.Context(), adminID, paymentID, req)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 6: Return success
	res.Success(c, http.StatusOK, "Payment reconciled successfully", nil)
}

// =====================================================
// ADMIN ANALYTICS ENDPOINTS
// =====================================================

// GetPaymentDashboard gets payment analytics dashboard
// AdminListPendingRefunds lists pending refund requests
// GET /api/v1/admin/payments/refunds/pending
func (h *PaymentHandler) AdminListPendingRefunds(c *gin.Context) {
	// Step 1: Get pagination parameters
	page := 1
	limit := 20

	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// Step 2: Call service
	refunds, total, err := h.refundService.ListPendingRefunds(c.Request.Context(), page, limit)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 3: Build response
	totalPages := (total + limit - 1) / limit
	response := gin.H{
		"refunds": refunds,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}

	res.Success(c, http.StatusOK, "OK", response)
}

// AdminGetRefundDetail gets refund request detail
// GET /api/v1/admin/payments/refunds/:refund_id
func (h *PaymentHandler) AdminGetRefundDetail(c *gin.Context) {
	// Step 1: Get refund ID
	refundIDStr := c.Param("refund_id")
	refundID, err := uuid.Parse(refundIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REFUND_ID", "Invalid refund ID")
		return
	}

	// Step 2: Call service
	refund, details, err := h.refundService.GetRefundDetail(c.Request.Context(), refundID)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 3: Build response
	response := gin.H{
		"refund":  refund,
		"details": details,
	}

	res.Success(c, http.StatusOK, "OK", response)
}

// AdminApproveRefund approves refund request
// POST /api/v1/admin/payments/refunds/:refund_id/approve
func (h *PaymentHandler) AdminApproveRefund(c *gin.Context) {
	// Step 1: Get admin ID
	adminIDStr, err := getAdminID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_ADMIN_ID", "Invalid admin ID")
		return
	}

	// Step 2: Get refund ID
	refundIDStr := c.Param("refund_id")
	refundID, err := uuid.Parse(refundIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REFUND_ID", "Invalid refund ID")
		return
	}

	// Step 3: Bind request body
	var req model.ApproveRefundRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		// Admin notes is optional, so empty body is OK
		req = model.ApproveRefundRequestDTO{}
	}

	// Step 4: Call service
	response, err := h.refundService.ApproveRefund(c.Request.Context(), adminID, refundID, req)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return response
	res.Success(c, http.StatusOK, "OK", response)
}

// AdminRejectRefund rejects refund request
// POST /api/v1/admin/payments/refunds/:refund_id/reject
func (h *PaymentHandler) AdminRejectRefund(c *gin.Context) {
	// Step 1: Get admin ID
	adminIDStr, err := getAdminID(c)
	if err != nil {
		res.Error(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_ADMIN_ID", "Invalid admin ID")
		return
	}

	// Step 2: Get refund ID
	refundIDStr := c.Param("refund_id")
	refundID, err := uuid.Parse(refundIDStr)
	if err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REFUND_ID", "Invalid refund ID")
		return
	}

	// Step 3: Bind request body
	var req model.RejectRefundRequestDTO
	if err := bindJSON(c, &req); err != nil {
		res.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 4: Validate request
	if err := req.Validate(); err != nil {
		res.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 5: Call service
	err = h.refundService.RejectRefund(c.Request.Context(), adminID, refundID, req)
	if err != nil {
		statusCode, errCode := mapPaymentError(err)
		res.Error(c, statusCode, errCode, err.Error())
		return
	}

	// Step 6: Return success
	res.Success(c, http.StatusOK, "Refund request rejected successfully", nil)
}
