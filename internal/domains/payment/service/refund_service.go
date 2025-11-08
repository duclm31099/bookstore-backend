package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	os "bookstore-backend/internal/domains/order/service"
	"bookstore-backend/internal/domains/payment/gateway"
	"bookstore-backend/internal/domains/payment/model"
	repo "bookstore-backend/internal/domains/payment/repository"
)

// =====================================================
// REFUND SERVICE INTERFACE
// =====================================================

// =====================================================
// REFUND SERVICE IMPLEMENTATION
// =====================================================
type refundService struct {
	paymentRepo repo.PaymentRepoInteface
	refundRepo  repo.RefundRepoInterface
	txManager   repo.TransactionManager

	vnpayGateway gateway.VNPayGateway
	momoGateway  gateway.MomoGateway

	orderService os.OrderService
}

func NewRefundService(
	paymentRepo repo.PaymentRepoInteface,
	refundRepo repo.RefundRepoInterface,
	txManager repo.TransactionManager,
	vnpayGateway gateway.VNPayGateway,
	momoGateway gateway.MomoGateway,
	orderService os.OrderService,
) RefundInterface {
	return &refundService{
		paymentRepo:  paymentRepo,
		refundRepo:   refundRepo,
		txManager:    txManager,
		vnpayGateway: vnpayGateway,
		momoGateway:  momoGateway,
		orderService: orderService,
	}
}

// =====================================================
// USER: REQUEST REFUND
// =====================================================

// RequestRefund creates refund request for a payment
//
// Business Logic:
// 1. Validate request
// 2. Get payment and verify user ownership
// 3. Validate payment can be refunded:
//   - Payment status = 'success'
//   - Gateway != 'cod' (COD doesn't need refund)
//   - No pending refund request exists
//
// 4. Get order and validate refund eligibility:
//   - Order status in ['cancelled', 'returned', 'delivered']
//   - Within refund window (7 days after delivery)
//
// 5. Create refund_requests record
// 6. Notify admin
func (s *refundService) RequestRefund(
	ctx context.Context,
	userID uuid.UUID,
	paymentID uuid.UUID,
	req model.CreateRefundRequestDTO,
) (*model.RefundRequestResponse, error) {
	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Get payment and verify ownership
	payment, err := s.paymentRepo.GetByIDAndVerifyUser(ctx, paymentID, userID)
	if err != nil {
		return nil, model.NewPaymentError(model.ErrCodePaymentNotFound, "Payment not found", err)
	}

	// Step 3: Validate payment can be refunded
	if !payment.CanBeRefunded() {
		if payment.Gateway == model.GatewayCOD {
			return nil, model.NewRefundNotAllowedError("COD orders cannot be refunded (no payment made)")
		}
		if payment.Status != model.PaymentStatusSuccess {
			return nil, model.NewPaymentError(
				model.ErrCodePaymentNotSuccessful,
				fmt.Sprintf("Payment status is '%s', must be 'success' to refund", payment.Status),
				model.ErrPaymentNotSuccessful,
			)
		}
		return nil, model.NewRefundNotAllowedError("Payment cannot be refunded")
	}

	// Step 4: Check no pending refund exists
	hasPending, err := s.refundRepo.HasPendingRefund(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending refund: %w", err)
	}
	if hasPending {
		return nil, model.NewPaymentError(
			model.ErrCodeRefundAlreadyExists,
			"A refund request already exists for this payment",
			model.ErrRefundAlreadyExists,
		)
	}

	// Step 5: Get order and validate refund eligibility
	order, err := s.orderService.GetOrderDetail(ctx, payment.OrderID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Check order status allows refund
	if order.Status != "cancelled" && order.Status != "returned" && order.Status != "delivered" {
		return nil, model.NewPaymentError(
			model.ErrCodeOrderCannotRefund,
			fmt.Sprintf("Order status '%s' does not allow refund", order.Status),
			model.ErrOrderCannotRefund,
		)
	}

	// Check refund window (7 days after delivery)
	if order.Status == "delivered" && order.DeliveredAt != nil {
		daysSinceDelivery := time.Since(*order.DeliveredAt).Hours() / 24
		if daysSinceDelivery > float64(model.RefundWindowDays) {
			return nil, model.NewPaymentError(
				model.ErrCodeRefundWindowExpired,
				fmt.Sprintf("Refund window expired (max %d days after delivery)", model.RefundWindowDays),
				model.ErrRefundWindowExpired,
			)
		}
	}

	// Step 6: Create refund request
	refundID := uuid.New()
	refund := &model.RefundRequest{
		ID:                   refundID,
		PaymentTransactionID: paymentID,
		OrderID:              payment.OrderID,
		RequestedBy:          userID,
		RequestedAmount:      payment.Amount, // Full refund only
		Reason:               req.Reason,
		ProofImages:          req.ProofImages,
		Status:               model.RefundStatusPending,
		RequestedAt:          time.Now(),
	}

	if err := s.refundRepo.Create(ctx, refund); err != nil {
		return nil, fmt.Errorf("failed to create refund request: %w", err)
	}

	// Step 7: Build response
	response := &model.RefundRequestResponse{
		RefundRequestID:         refundID,
		PaymentTransactionID:    paymentID,
		Status:                  model.RefundStatusPending,
		RequestedAmount:         payment.Amount,
		Reason:                  req.Reason,
		RequestedAt:             time.Now(),
		EstimatedProcessingTime: "3-5 business days",
	}

	// TODO: Send notification to admin
	fmt.Printf("New refund request: %s for payment %s\n", refundID, paymentID)

	return response, nil
}

// =====================================================
// USER: GET REFUND STATUS
// =====================================================

// GetRefundStatus gets refund request status
func (s *refundService) GetRefundStatus(
	ctx context.Context,
	userID uuid.UUID,
	paymentID uuid.UUID,
) (*model.RefundRequestResponse, error) {
	// Verify payment ownership
	payment, err := s.paymentRepo.GetByIDAndVerifyUser(ctx, paymentID, userID)
	if err != nil {
		return nil, err
	}

	// Get refund request
	refund, err := s.refundRepo.GetByPaymentID(ctx, payment.ID)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &model.RefundRequestResponse{
		RefundRequestID:      refund.ID,
		PaymentTransactionID: refund.PaymentTransactionID,
		Status:               refund.Status,
		RequestedAmount:      refund.RequestedAmount,
		Reason:               refund.Reason,
		RequestedAt:          refund.RequestedAt,
		ApprovedAt:           refund.ApprovedAt,
		RejectedAt:           refund.RejectedAt,
		RejectionReason:      refund.RejectionReason,
		GatewayRefundID:      refund.GatewayRefundID,
		CompletedAt:          refund.CompletedAt,
	}

	return response, nil
}

// =====================================================
// ADMIN: LIST PENDING REFUNDS
// =====================================================

// ListPendingRefunds lists pending refund requests
func (s *refundService) ListPendingRefunds(
	ctx context.Context,
	page, limit int,
) ([]*model.RefundRequest, int, error) {
	return s.refundRepo.ListPendingRefunds(ctx, page, limit)
}

// =====================================================
// ADMIN: GET REFUND DETAIL
// =====================================================

// GetRefundDetail gets refund request with full details
func (s *refundService) GetRefundDetail(
	ctx context.Context,
	refundID uuid.UUID,
) (*model.RefundRequest, map[string]interface{}, error) {
	result, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return nil, nil, err
	}
	return result, nil, nil
}

// =====================================================
// ADMIN: APPROVE REFUND
// =====================================================

// ApproveRefund approves refund request and initiates gateway refund
//
// Business Logic:
// 1. Get refund request and validate can be approved
// 2. Get payment transaction
// 3. Start transaction
// 4. Update refund status to 'approved'
// 5. Initiate gateway refund API call
// 6. Update refund with gateway refund ID
// 7. Commit transaction
// 8. Gateway webhook will mark as 'completed' later
func (s *refundService) ApproveRefund(
	ctx context.Context,
	adminID uuid.UUID,
	refundID uuid.UUID,
	req model.ApproveRefundRequestDTO,
) (*model.RefundRequestResponse, error) {
	// Step 1: Get refund request
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return nil, err
	}

	// Validate can be approved
	if !refund.CanBeApproved() {
		return nil, model.NewPaymentError(
			model.ErrCodeRefundNotAllowed,
			fmt.Sprintf("Refund status is '%s', cannot approve", refund.Status),
			model.ErrCannotApproveRefund,
		)
	}

	// Step 2: Get payment transaction
	payment, err := s.paymentRepo.GetByID(ctx, refund.PaymentTransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Step 3: Start transaction
	tx, err := s.txManager.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.txManager.RollbackTx(ctx, tx)

	// Step 4: Approve refund request
	if err := s.refundRepo.Approve(ctx, refundID, adminID, req.AdminNotes); err != nil {
		return nil, fmt.Errorf("failed to approve refund: %w", err)
	}

	// Step 5: Initiate gateway refund
	var gatewayRefundID string
	var gatewayResponse map[string]interface{}

	switch payment.Gateway {
	case model.GatewayVNPay:
		// Call VNPay refund API
		refundResp, err := s.vnpayGateway.InitiateRefund(ctx, gateway.VNPayRefundRequest{
			TransactionID:   *payment.TransactionID,
			Amount:          refund.RequestedAmount,
			RefundAmount:    refund.RequestedAmount,
			TransactionDate: payment.CompletedAt.Format("20060102150405"),
			Reason:          refund.Reason,
		})

		if err != nil {
			// Mark refund as failed
			s.refundRepo.MarkAsFailed(ctx, refundID, err.Error())
			return nil, fmt.Errorf("VNPay refund API failed: %w", err)
		}

		gatewayRefundID = refundResp.RefundTransactionID
		gatewayResponse = refundResp.RawResponse

	case model.GatewayMomo:
		// Call Momo refund API
		// TODO: Implement Momo refund
		return nil, fmt.Errorf("Momo refund not implemented")

	case model.GatewayCOD:
		// COD doesn't need refund (already handled in validation)
		return nil, model.NewPaymentError(
			model.ErrCodeCODNoRefund,
			"COD orders cannot be refunded",
			model.ErrCODNoRefund,
		)
	}

	// Step 6: Update refund with gateway refund ID
	if err := s.refundRepo.UpdateGatewayRefund(ctx, refundID, gatewayRefundID, gatewayResponse); err != nil {
		return nil, fmt.Errorf("failed to update gateway refund: %w", err)
	}

	// Step 7: Commit transaction
	if err := s.txManager.CommitTx(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Step 8: Build response
	now := time.Now()
	response := &model.RefundRequestResponse{
		RefundRequestID:      refundID,
		PaymentTransactionID: refund.PaymentTransactionID,
		Status:               model.RefundStatusProcessing,
		RequestedAmount:      refund.RequestedAmount,
		Reason:               refund.Reason,
		RequestedAt:          refund.RequestedAt,
		ApprovedAt:           &now,
		GatewayRefundID:      &gatewayRefundID,
	}

	// TODO: Notify user
	fmt.Printf("Refund approved: %s, gateway refund ID: %s\n", refundID, gatewayRefundID)

	return response, nil
}

// =====================================================
// ADMIN: REJECT REFUND
// =====================================================

// RejectRefund rejects refund request
func (s *refundService) RejectRefund(
	ctx context.Context,
	adminID uuid.UUID,
	refundID uuid.UUID,
	req model.RejectRefundRequestDTO,
) error {
	// Validate request
	if err := req.Validate(); err != nil {
		return err
	}

	// Get refund request
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return err
	}

	// Validate can be rejected
	if !refund.CanBeRejected() {
		return model.NewPaymentError(
			model.ErrCodeRefundNotAllowed,
			fmt.Sprintf("Refund status is '%s', cannot reject", refund.Status),
			model.ErrCannotRejectRefund,
		)
	}

	// Reject refund
	if err := s.refundRepo.Reject(ctx, refundID, adminID, req.RejectionReason); err != nil {
		return fmt.Errorf("failed to reject refund: %w", err)
	}

	// TODO: Notify user
	fmt.Printf("Refund rejected: %s, reason: %s\n", refundID, req.RejectionReason)

	return nil
}
