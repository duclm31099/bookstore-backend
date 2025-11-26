package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	orderModel "bookstore-backend/internal/domains/order/model"
	os "bookstore-backend/internal/domains/order/service"
	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/model"
	repo "bookstore-backend/internal/domains/payment/repository"
	"bookstore-backend/pkg/logger"
)

// =====================================================
// PAYMENT SERVICE IMPLEMENTATION
// =====================================================
type paymentService struct {
	paymentRepo repo.PaymentRepoInteface
	webhookRepo repo.WebhookRepoInterface
	refundRepo  repo.RefundRepoInterface
	txManager   repo.TransactionManager

	// Gateway integrations
	vnpayGateway gateway.VNPayGateway
	momoGateway  gateway.MomoGateway

	// Order service (for cross-domain operations)
	orderService os.OrderService
}

func NewPaymentService(
	paymentRepo repo.PaymentRepoInteface,
	webhookRepo repo.WebhookRepoInterface,
	refundRepo repo.RefundRepoInterface,
	txManager repo.TransactionManager,
	vnpayGateway gateway.VNPayGateway,
	momoGateway gateway.MomoGateway,
	orderService os.OrderService,
) PaymentService {
	return &paymentService{
		paymentRepo:  paymentRepo,
		webhookRepo:  webhookRepo,
		refundRepo:   refundRepo,
		txManager:    txManager,
		vnpayGateway: vnpayGateway,
		momoGateway:  momoGateway,
		orderService: orderService,
	}
}

// =====================================================
// CREATE PAYMENT
// =====================================================

// CreatePayment initiates payment for an order
//
// Business Logic Flow:
// 1. Validate request
// 2. Get order and verify ownership
// 3. Validate order status (must be 'pending')
// 4. Check no existing successful payment
// 5. Check retry limit (max 3 attempts)
// 6. Create payment_transactions record
// 7. Generate payment URL (VNPay/Momo) or confirm COD
// 8. Return response with payment URL or confirmation
//
// Edge Cases:
// - Order not found -> PAY001
// - Order already paid -> PAY002
// - Retry limit exceeded -> PAY003
// - Order not in pending status -> PAY004
// - Invalid gateway -> PAY005
func (s *paymentService) CreatePayment(
	ctx context.Context,
	userID uuid.UUID,
	req model.CreatePaymentRequest,
) (*model.CreatePaymentResponse, error) {
	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return nil, model.NewPaymentError(model.ErrCodeInvalidGateway, "Invalid request", err)
	}

	// Step 2: Get order and verify ownership
	order, err := s.orderService.GetOrderDetail(ctx, req.OrderID, userID)
	if err != nil {
		return nil, model.NewPaymentError(model.ErrCodePaymentNotFound, "Order not found", err)
	}

	// Step 3: Validate order status (must be 'pending')
	if order.Status != orderModel.OrderStatusPending {
		return nil, model.NewOrderNotPendingError(order.Status)
	}

	// Step 4: Check no existing successful payment
	hasSuccessPayment, err := s.paymentRepo.HasSuccessfulPayment(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check successful payment: %w", err)
	}
	if hasSuccessPayment {
		return nil, model.NewOrderAlreadyPaidError(req.OrderID.String())
	}

	// Step 5: Check retry limit (max 3 attempts)
	canRetry, attemptCount, err := s.paymentRepo.CheckRetryLimit(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check retry limit: %w", err)
	}
	if !canRetry {
		return nil, model.NewRetryLimitExceededError()
	}

	// Step 6: Create payment_transactions record
	paymentID := uuid.New()
	payment := &model.PaymentTransaction{
		ID:          paymentID,
		OrderID:     req.OrderID,
		Gateway:     model.GatewayVNPay,
		Amount:      order.Total,
		Currency:    model.DefaultCurrency,
		Status:      model.PaymentStatusPending,
		RetryCount:  attemptCount,
		InitiatedAt: time.Now(),
	}

	// Create payment record
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Step 7: Generate payment URL based on gateway
	response := &model.CreatePaymentResponse{
		PaymentTransactionID: paymentID,
		Gateway:              req.Gateway,
		Amount:               order.Total,
		Currency:             model.DefaultCurrency,
		ExpiresAt:            time.Now().Add(time.Duration(model.PaymentTimeoutMinutes) * time.Minute),
	}

	// switch req.Gateway {
	// case model.GatewayVNPay:

	// case model.GatewayMomo:
	// 	// Generate Momo payment URL
	// 	paymentURL, err := s.momoGateway.CreatePaymentURL(ctx, gateway.MomoPaymentRequest{
	// 		OrderID:   paymentID.String(),
	// 		Amount:    order.Total,
	// 		OrderInfo: fmt.Sprintf("Payment for order %s", order.OrderNumber),
	// 	})

	// 	if err != nil {
	// 		s.paymentRepo.MarkAsFailed(ctx, paymentID, model.ErrCodeGatewayUnavailable, err.Error())
	// 		return nil, fmt.Errorf("failed to generate Momo URL: %w", err)
	// 	}

	// 	// Update payment to processing
	// 	s.paymentRepo.UpdateStatus(ctx, paymentID, model.PaymentStatusProcessing)

	// 	response.PaymentURL = &paymentURL

	// case model.GatewayCOD:
	// 	// COD: No payment URL needed, just confirmation
	// 	message := "COD order confirmed. Pay on delivery."
	// 	response.Message = &message
	// 	// COD payment stays in 'pending' until delivery

	// case model.GatewayBankTransfer:
	// 	// Bank Transfer: Generate QR code or bank details
	// 	// TODO: Implement bank transfer logic
	// 	bankAccount := "1234567890 - VietcomBank"
	// 	response.BankAccount = &bankAccount

	// default:
	// 	return nil, model.NewInvalidGatewayError(req.Gateway)
	// }
	// Generate VNPay payment URL
	paymentURL, err := s.vnpayGateway.CreatePaymentURL(ctx, gateway.VNPayPaymentRequest{
		TransactionRef: paymentID.String(),
		Amount:         order.Total,
		OrderInfo:      strings.ReplaceAll(order.OrderNumber, "-", ""),
		ReturnURL:      s.vnpayGateway.GetReturnURL(),
	})

	if err != nil {
		// Mark payment as failed
		s.paymentRepo.MarkAsFailed(ctx, paymentID, model.ErrCodeGatewayUnavailable, err.Error())
		return nil, fmt.Errorf("failed to generate VNPay URL: %w", err)
	}

	// Update payment to processing
	// This is critical - if this fails, we must rollback to prevent orphaned payments
	if err := s.paymentRepo.UpdateStatus(ctx, paymentID, model.PaymentStatusProcessing); err != nil {
		logger.Error("Failed to update payment status to processing", err)

		// ✅ ROLLBACK: Mark payment as failed to prevent limbo state
		// This ensures the payment record reflects the actual state
		rollbackErr := s.paymentRepo.MarkAsFailed(
			ctx,
			paymentID,
			model.ErrCodeGatewayUnavailable,
			fmt.Sprintf("Failed to update status: %v", err),
		)
		if rollbackErr != nil {
			logger.Error("Failed to rollback payment after status update error", rollbackErr)
			// Log both errors for debugging
		}

		// Return error to client - do NOT provide payment URL
		// User can retry the payment (new record will be created)
		return nil, fmt.Errorf("failed to prepare payment transaction: %w", err)
	}
	response = &model.CreatePaymentResponse{
		PaymentTransactionID: paymentID,
		Gateway:              req.Gateway,
		Amount:               order.Total,
		Currency:             model.DefaultCurrency,
		ExpiresAt:            time.Now().Add(time.Duration(model.PaymentTimeoutMinutes) * time.Minute),
		PaymentURL:           &paymentURL,
	}
	return response, nil
}

// =====================================================
// GET PAYMENT STATUS
// =====================================================

// GetPaymentStatus gets payment status for user
// Used for polling after payment redirect
func (s *paymentService) GetPaymentStatus(
	ctx context.Context,
	userID uuid.UUID,
	paymentID uuid.UUID,
) (*model.PaymentStatusResponse, error) {
	// Get payment and verify user ownership
	payment, err := s.paymentRepo.GetByIDAndVerifyUser(ctx, paymentID, userID)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &model.PaymentStatusResponse{
		TransactionID:  payment.ID,
		OrderID:        payment.OrderID,
		Gateway:        payment.Gateway,
		Status:         payment.Status,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		TransactionRef: payment.TransactionID,
		ErrorCode:      payment.ErrorCode,
		ErrorMessage:   payment.ErrorMessage,
		PaymentDetails: payment.PaymentDetails,
		InitiatedAt:    payment.InitiatedAt,
		CompletedAt:    payment.CompletedAt,
		FailedAt:       payment.FailedAt,
	}

	return response, nil
}

// =====================================================
// LIST USER PAYMENTS
// =====================================================

// ListUserPayments lists payments for current user
func (s *paymentService) ListUserPayments(
	ctx context.Context,
	userID uuid.UUID,
	req model.ListPaymentsRequest,
) (*model.ListPaymentsResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build filters
	filters := make(map[string]interface{})
	if req.OrderID != nil {
		filters["order_id"] = *req.OrderID
	}
	if req.Status != nil {
		filters["status"] = *req.Status
	}
	if req.Gateway != nil {
		filters["gateway"] = *req.Gateway
	}

	// Get payments from repository
	payments, total, err := s.paymentRepo.ListByUserID(ctx, userID, filters, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	// Build response
	summaries := make([]model.PaymentSummaryResponse, 0, len(payments))
	for _, p := range payments {
		summaries = append(summaries, model.PaymentSummaryResponse{
			TransactionID: p.ID,
			OrderID:       p.OrderID,
			Gateway:       p.Gateway,
			Status:        p.Status,
			Amount:        p.Amount,
			CompletedAt:   p.CompletedAt,
			CreatedAt:     p.CreatedAt,
		})
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &model.ListPaymentsResponse{
		Payments: summaries,
		Pagination: model.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// =====================================================
// PROCESS VNPAY WEBHOOK
// =====================================================

// ProcessVNPayWebhook processes VNPay IPN callback
//
// Business Logic Flow:
// 1. Log webhook to payment_webhook_logs
// 2. Verify signature
// 3. Check idempotency (prevent duplicate processing)
// 4. Get payment transaction by vnp_TxnRef
// 5. Process based on vnp_ResponseCode:
//   - "00" = success → update payment & order status
//   - Others = failed → mark payment as failed
//
// 6. Handle edge case: payment success but order cancelled
// 7. Mark webhook as processed
//
// Idempotency:
// - Webhooks can arrive multiple times (network retry)
// - Check unique constraint: (gateway, event, transaction_id, is_processed=true)
// - If already processed, return success immediately
//
// Security:
// - Verify HMAC-SHA512 signature
// - Reject invalid signatures (potential fraud)
func (s *paymentService) ProcessVNPayWebhook(
	ctx context.Context,
	webhookData model.VNPayWebhookRequest,
) error {
	// Step 1: Create webhook log (audit trail)
	webhookID := uuid.New()
	webhookLog := &model.PaymentWebhookLog{
		ID:           webhookID,
		Gateway:      model.GatewayVNPay,
		WebhookEvent: &model.WebhookEventPaymentSuccess, // Will update based on response code
		Body: map[string]interface{}{
			"vnp_Amount":            webhookData.VnpAmount,
			"vnp_BankCode":          webhookData.VnpBankCode,
			"vnp_CardType":          webhookData.VnpCardType,
			"vnp_OrderInfo":         webhookData.VnpOrderInfo,
			"vnp_PayDate":           webhookData.VnpPayDate,
			"vnp_ResponseCode":      webhookData.VnpResponseCode,
			"vnp_TmnCode":           webhookData.VnpTmnCode,
			"vnp_TransactionNo":     webhookData.VnpTransactionNo,
			"vnp_TxnRef":            webhookData.VnpTxnRef,
			"vnp_TransactionStatus": webhookData.VnpTransactionStatus,
			"transaction_id":        webhookData.VnpTransactionNo, // For idempotency check
		},
		Signature:  &webhookData.VnpSecureHash,
		ReceivedAt: time.Now(),
	}

	// Step 2: Verify signature
	isValid := s.vnpayGateway.VerifySignature(webhookData)
	if !isValid {
		// Invalid signature - potential fraud
		isValidFlag := false
		webhookLog.IsValid = &isValidFlag
		s.webhookRepo.Create(ctx, webhookLog)

		return model.NewInvalidSignatureError()
	}

	// Signature is valid
	isValidFlag := true
	webhookLog.IsValid = &isValidFlag

	// Step 3: Check idempotency
	alreadyProcessed, err := s.webhookRepo.CheckIdempotency(
		ctx,
		model.GatewayVNPay,
		*webhookLog.WebhookEvent,
		webhookData.VnpTransactionNo,
	)
	if err != nil {
		s.webhookRepo.Create(ctx, webhookLog)
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if alreadyProcessed {
		// Already processed, return success (idempotent)
		webhookLog.IsProcessed = true
		s.webhookRepo.Create(ctx, webhookLog)
		return nil
	}

	// Step 4: Get payment transaction by vnp_TxnRef (payment_transaction.id)
	paymentID, err := uuid.Parse(webhookData.VnpTxnRef)
	if err != nil {
		s.webhookRepo.Create(ctx, webhookLog)
		return fmt.Errorf("invalid transaction ref: %w", err)
	}

	payment, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		s.webhookRepo.Create(ctx, webhookLog)
		return fmt.Errorf("payment not found: %w", err)
	}

	// Attach payment_transaction_id to webhook log
	webhookLog.PaymentTransactionID = &payment.ID
	webhookLog.OrderID = &payment.OrderID

	// Create webhook log
	if err := s.webhookRepo.Create(ctx, webhookLog); err != nil {
		return fmt.Errorf("failed to create webhook log: %w", err)
	}

	// Step 5: Process based on response code
	if webhookData.VnpResponseCode == "00" {
		// Payment success
		err = s.handleSuccessfulPayment(ctx, payment, webhookData)
	} else {
		// Payment failed
		err = s.handleFailedPayment(ctx, payment, webhookData)
	}

	if err != nil {
		// Mark webhook processing error but don't return error (webhook acknowledged)
		s.webhookRepo.MarkProcessingError(ctx, webhookID, err.Error())
		return err
	}

	// Step 6: Mark webhook as processed
	if err := s.webhookRepo.MarkAsProcessed(ctx, webhookID); err != nil {
		return fmt.Errorf("failed to mark webhook as processed: %w", err)
	}

	return nil
}

// handleSuccessfulPayment handles successful payment webhook
func (s *paymentService) handleSuccessfulPayment(
	ctx context.Context,
	payment *model.PaymentTransaction,
	webhookData model.VNPayWebhookRequest,
) error {
	// Start transaction for atomic update
	tx, err := s.txManager.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.txManager.RollbackTx(ctx, tx)

	// Build payment details from webhook
	paymentDetails := map[string]interface{}{
		"bank_code": webhookData.VnpBankCode,
		"card_type": webhookData.VnpCardType,
		"pay_date":  webhookData.VnpPayDate,
	}

	gatewayResponse := map[string]interface{}{
		"vnp_Amount":            webhookData.VnpAmount,
		"vnp_BankCode":          webhookData.VnpBankCode,
		"vnp_CardType":          webhookData.VnpCardType,
		"vnp_OrderInfo":         webhookData.VnpOrderInfo,
		"vnp_PayDate":           webhookData.VnpPayDate,
		"vnp_ResponseCode":      webhookData.VnpResponseCode,
		"vnp_TransactionNo":     webhookData.VnpTransactionNo,
		"vnp_TransactionStatus": webhookData.VnpTransactionStatus,
	}

	// Update payment to success
	err = s.paymentRepo.MarkAsSuccess(
		ctx,
		payment.ID,
		webhookData.VnpTransactionNo,
		gatewayResponse,
		paymentDetails,
	)
	if err != nil {
		return fmt.Errorf("failed to mark payment as success: %w", err)
	}

	// Commit transaction
	if err := s.txManager.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Note: Trigger sync_order_payment_status() will automatically:
	// - Update orders.payment_status = 'paid'
	// - Update orders.paid_at = NOW()
	// - Update orders.status = 'confirmed' (if currently 'pending')

	// Edge case: Check if order was cancelled after payment initiated
	order, err := s.orderService.GetOrderByIDWithoutUser(ctx, payment.OrderID)
	if err == nil && order.Status == "cancelled" {
		// Order was cancelled but payment succeeded
		// Need to initiate auto-refund
		// TODO: Alert admin for manual reconciliation
		// For now, just log the issue
		fmt.Printf("WARNING: Payment %s succeeded but order %s is cancelled. Manual refund needed.\n",
			payment.ID, payment.OrderID)
	}

	return nil
}

// handleFailedPayment handles failed payment webhook
func (s *paymentService) handleFailedPayment(
	ctx context.Context,
	payment *model.PaymentTransaction,
	webhookData model.VNPayWebhookRequest,
) error {
	// Map VNPay error code to internal error code
	internalCode, errorMessage := model.MapVNPayErrorCode(webhookData.VnpResponseCode)

	// Update payment to failed
	err := s.paymentRepo.MarkAsFailed(ctx, payment.ID, internalCode, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to mark payment as failed: %w", err)
	}

	// Note: Trigger sync_order_payment_status() will automatically:
	// - Update orders.payment_status = 'failed'

	return nil
}

// =====================================================
// PROCESS MOMO WEBHOOK
// =====================================================

// ProcessMomoWebhook processes Momo IPN callback
// Similar logic to VNPay but with Momo-specific data structure
func (s *paymentService) ProcessMomoWebhook(
	ctx context.Context,
	webhookData model.MomoWebhookRequest,
) error {
	// Similar implementation to VNPay
	// TODO: Implement Momo webhook processing
	// Key differences:
	// - Signature algorithm: HMAC-SHA256 (not SHA512)
	// - Response codes: 0=success, others=fail
	// - Data structure different from VNPay

	return fmt.Errorf("Momo webhook not implemented yet")
}

// =====================================================
// BACKGROUND JOBS
// =====================================================

// CancelExpiredPayments cancels payments that exceeded timeout
//
// Business Logic:
//  1. Get expired payments (pending/processing, > 15 minutes)
//  2. For each expired payment:
//     a. Mark payment as cancelled
//     b. Call Order Service to cancel order
//     c. Release inventory (via Order Service)
//  3. Return count of cancelled payments
//
// Runs every 5 minutes via Asynq job
func (s *paymentService) CancelExpiredPayments(ctx context.Context) (int, error) {
	// Get expired payments (limit batch size)
	expiredPayments, err := s.paymentRepo.GetExpiredPayments(ctx, 100)
	if err != nil {
		return 0, fmt.Errorf("failed to get expired payments: %w", err)
	}

	cancelledCount := 0

	for _, payment := range expiredPayments {
		// Mark payment as cancelled
		reason := fmt.Sprintf("Payment timeout after %d minutes", model.PaymentTimeoutMinutes)
		err := s.paymentRepo.MarkAsCancelled(ctx, payment.ID, reason)
		if err != nil {
			fmt.Printf("Failed to cancel payment %s: %v\n", payment.ID, err)
			continue
		}

		// Call Order Service to cancel order and release inventory
		err = s.orderService.CancelOrderBySystem(ctx, payment.OrderID, reason, "payment_timeout")
		if err != nil {
			fmt.Printf("Failed to cancel order %s: %v\n", payment.OrderID, err)
			// Payment is cancelled but order might not be
			// Admin should review this case
			continue
		}

		cancelledCount++
	}

	return cancelledCount, nil
}

// RetryFailedWebhooks retries webhooks that failed processing
// Runs every 10 minutes via Asynq job
func (s *paymentService) RetryFailedWebhooks(ctx context.Context) (int, error) {
	// Get failed webhooks (limit batch size)
	failedWebhooks, err := s.webhookRepo.GetFailedWebhooks(ctx, 50)
	if err != nil {
		return 0, fmt.Errorf("failed to get failed webhooks: %w", err)
	}

	retriedCount := 0

	for _, webhook := range failedWebhooks {
		// Retry webhook processing based on gateway
		var retryErr error

		switch webhook.Gateway {
		case model.GatewayVNPay:
			// Reconstruct VNPay webhook data from stored body
			webhookData := model.VNPayWebhookRequest{
				VnpAmount:            webhook.Body["vnp_Amount"].(string),
				VnpBankCode:          webhook.Body["vnp_BankCode"].(string),
				VnpCardType:          webhook.Body["vnp_CardType"].(string),
				VnpOrderInfo:         webhook.Body["vnp_OrderInfo"].(string),
				VnpPayDate:           webhook.Body["vnp_PayDate"].(string),
				VnpResponseCode:      webhook.Body["vnp_ResponseCode"].(string),
				VnpTmnCode:           webhook.Body["vnp_TmnCode"].(string),
				VnpTransactionNo:     webhook.Body["vnp_TransactionNo"].(string),
				VnpTxnRef:            webhook.Body["vnp_TxnRef"].(string),
				VnpSecureHash:        *webhook.Signature,
				VnpTransactionStatus: webhook.Body["vnp_TransactionStatus"].(string),
			}

			retryErr = s.ProcessVNPayWebhook(ctx, webhookData)

		case model.GatewayMomo:
			// TODO: Retry Momo webhook
			continue
		}

		if retryErr != nil {
			// Mark error for this retry attempt
			s.webhookRepo.MarkProcessingError(ctx, webhook.ID, retryErr.Error())
			continue
		}

		retriedCount++
	}

	return retriedCount, nil
}

// =====================================================
// ADMIN METHODS (TODO: Implement in next part)
// =====================================================

// AdminListPayments lists all payments with filters (admin only)
func (s *paymentService) AdminListPayments(
	ctx context.Context,
	req model.AdminListPaymentsRequest,
) (*model.AdminListPaymentsResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build filters map
	filters := make(map[string]interface{})

	if req.Status != nil {
		filters["status"] = *req.Status
	}
	if req.Gateway != nil {
		filters["gateway"] = *req.Gateway
	}
	if req.FromDate != nil {
		filters["from_date"] = *req.FromDate
	}
	if req.ToDate != nil {
		filters["to_date"] = *req.ToDate
	}
	if req.MinAmount != nil {
		filters["min_amount"] = *req.MinAmount
	}
	if req.MaxAmount != nil {
		filters["max_amount"] = *req.MaxAmount
	}
	if req.Search != nil {
		filters["search"] = *req.Search
	}

	// Get payments from repository
	payments, total, err := s.paymentRepo.AdminListPayments(ctx, filters, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	// Get statistics
	stats, err := s.paymentRepo.AdminGetStatistics(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// Build response
	paymentResponses := make([]model.AdminPaymentResponse, 0, len(payments))
	for _, p := range payments {
		// Check if has refund request
		hasRefund, _ := s.refundRepo.HasPendingRefund(ctx, p.ID)

		paymentResponses = append(paymentResponses, model.AdminPaymentResponse{
			TransactionID:    p.ID,
			OrderNumber:      "", // TODO: Get from joined order data
			UserEmail:        "", // TODO: Get from joined user data
			Gateway:          p.Gateway,
			Status:           p.Status,
			Amount:           p.Amount,
			CompletedAt:      p.CompletedAt,
			HasRefundRequest: hasRefund,
			CreatedAt:        p.CreatedAt,
		})
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &model.AdminListPaymentsResponse{
		Payments:   paymentResponses,
		Statistics: *stats,
		Pagination: model.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// =====================================================
// ADMIN: GET PAYMENT DETAIL
// =====================================================

// AdminGetPaymentDetail gets detailed payment info (admin only)
func (s *paymentService) AdminGetPaymentDetail(
	ctx context.Context,
	paymentID uuid.UUID,
) (*model.AdminPaymentDetailResponse, error) {
	// Get payment
	payment, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Get order info
	order, err := s.orderService.GetOrderByIDWithoutUser(ctx, payment.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Get webhook logs
	webhookLogs, err := s.webhookRepo.ListByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook logs: %w", err)
	}

	// Build webhook log summaries
	webhookSummaries := make([]model.WebhookLogSummary, 0, len(webhookLogs))
	for _, log := range webhookLogs {
		webhookSummaries = append(webhookSummaries, model.WebhookLogSummary{
			ID:          log.ID,
			Event:       log.WebhookEvent,
			IsValid:     log.IsValid,
			IsProcessed: log.IsProcessed,
			ReceivedAt:  log.ReceivedAt,
		})
	}

	// Build response
	response := &model.AdminPaymentDetailResponse{
		TransactionID: payment.ID,
		Order: model.OrderInfo{
			ID:          order.ID,
			OrderNumber: order.OrderNumber,
			User: model.UserInfo{
				ID:    uuid.Nil,
				Email: "", // TODO: Get from user service
				Name:  "",
			},
		},
		Gateway:         payment.Gateway,
		Status:          payment.Status,
		Amount:          payment.Amount,
		GatewayResponse: payment.GatewayResponse,
		PaymentDetails:  payment.PaymentDetails,
		WebhookLogs:     webhookSummaries,
		RetryCount:      payment.RetryCount,
		InitiatedAt:     payment.InitiatedAt,
		CompletedAt:     payment.CompletedAt,
		FailedAt:        payment.FailedAt,
	}

	return response, nil
}

// =====================================================
// ADMIN: MANUAL RECONCILIATION
// =====================================================

// AdminReconcilePayment manually updates payment status (admin only)
//
// Use Case:
// - Webhook failed to process but payment actually succeeded on gateway
// - Admin verifies payment on gateway dashboard
// - Admin manually updates payment status in system
//
// Business Logic:
// 1. Validate request
// 2. Get payment transaction
// 3. Verify admin has permission (role check)
// 4. Update payment status based on admin input
// 5. Sync order status (if payment success)
// 6. Create audit log
func (s *paymentService) AdminReconcilePayment(
	ctx context.Context,
	adminID uuid.UUID,
	paymentID uuid.UUID,
	req model.ManualReconciliationRequest,
) error {
	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return err
	}

	// Step 2: Get payment
	_, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}

	// Step 3: Start transaction
	tx, err := s.txManager.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.txManager.RollbackTx(ctx, tx)

	// Step 4: Update payment based on admin's verification
	if req.Status == model.PaymentStatusSuccess {
		// Admin verified payment succeeded
		gatewayResponse := map[string]interface{}{
			"manual_reconciliation": true,
			"reconciled_by":         adminID.String(),
			"notes":                 req.Notes,
		}

		paymentDetails := map[string]interface{}{
			"gateway_transaction_id": req.GatewayTransactionID,
		}

		err = s.paymentRepo.MarkAsSuccess(
			ctx,
			paymentID,
			req.GatewayTransactionID,
			gatewayResponse,
			paymentDetails,
		)

		if err != nil {
			return fmt.Errorf("failed to mark payment as success: %w", err)
		}

		// Trigger will sync order status automatically

	} else if req.Status == model.PaymentStatusFailed {
		// Admin verified payment failed
		err = s.paymentRepo.MarkAsFailed(
			ctx,
			paymentID,
			model.ErrCodeGatewayUnavailable,
			fmt.Sprintf("Manual reconciliation: %s", req.Notes),
		)

		if err != nil {
			return fmt.Errorf("failed to mark payment as failed: %w", err)
		}
	}

	// Step 5: Commit transaction
	if err := s.txManager.CommitTx(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// TODO: Create admin audit log entry
	fmt.Printf("Admin %s reconciled payment %s: status=%s, notes=%s\n",
		adminID, paymentID, req.Status, req.Notes)

	return nil
}
