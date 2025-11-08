package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// REFUND REQUEST REPOSITORY IMPLEMENTATION
// =====================================================
type refundRepository struct {
	pool *pgxpool.Pool
}

func NewRefundRepository(pool *pgxpool.Pool) RefundRepoInterface {
	return &refundRepository{pool: pool}
}

// =====================================================
// TRANSACTION-AWARE METHODS
// =====================================================

// CreateWithTx creates refund request within provided transaction
func (r *refundRepository) CreateWithTx(
	ctx context.Context,
	tx pgx.Tx,
	refund *model.RefundRequest,
) error {
	query := `
		INSERT INTO refund_requests (
			id, payment_transaction_id, order_id, requested_by,
			requested_amount, reason, proof_images, status, requested_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING updated_at
	`

	// Serialize proof_images to JSONB array
	proofImagesJSON, err := json.Marshal(refund.ProofImages)
	if err != nil {
		return fmt.Errorf("failed to marshal proof_images: %w", err)
	}

	err = tx.QueryRow(ctx, query,
		refund.ID,
		refund.PaymentTransactionID,
		refund.OrderID,
		refund.RequestedBy,
		refund.RequestedAmount,
		refund.Reason,
		proofImagesJSON,
		refund.Status,
		refund.RequestedAt,
	).Scan(&refund.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create refund request: %w", err)
	}

	return nil
}

// UpdateStatusWithTx updates refund request status within transaction
func (r *refundRepository) UpdateStatusWithTx(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
	status string,
) error {
	query := `
		UPDATE refund_requests
		SET status = $1,
			processing_at = CASE WHEN $1 = 'processing' THEN NOW() ELSE processing_at END,
			completed_at = CASE WHEN $1 = 'completed' THEN NOW() ELSE completed_at END,
			failed_at = CASE WHEN $1 = 'failed' THEN NOW() ELSE failed_at END,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := tx.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update refund status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrRefundRequestNotFound
	}

	return nil
}

// =====================================================
// STANDALONE METHODS
// =====================================================

// Create creates refund request
func (r *refundRepository) Create(
	ctx context.Context,
	refund *model.RefundRequest,
) error {
	query := `
		INSERT INTO refund_requests (
			id, payment_transaction_id, order_id, requested_by,
			requested_amount, reason, proof_images, status, requested_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING updated_at
	`

	// Serialize proof_images to JSONB array
	proofImagesJSON, err := json.Marshal(refund.ProofImages)
	if err != nil {
		return fmt.Errorf("failed to marshal proof_images: %w", err)
	}

	err = r.pool.QueryRow(ctx, query,
		refund.ID,
		refund.PaymentTransactionID,
		refund.OrderID,
		refund.RequestedBy,
		refund.RequestedAmount,
		refund.Reason,
		proofImagesJSON,
		refund.Status,
		refund.RequestedAt,
	).Scan(&refund.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create refund request: %w", err)
	}

	return nil
}

// GetByID gets refund request by ID
func (r *refundRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.RefundRequest, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, requested_by,
			requested_amount, reason, proof_images, status,
			approved_by, approved_at, admin_notes,
			rejected_by, rejected_at, rejection_reason,
			gateway_refund_id, gateway_refund_response,
			requested_at, processing_at, completed_at, failed_at, updated_at
		FROM refund_requests
		WHERE id = $1
	`

	refund := &model.RefundRequest{}
	var proofImagesJSON, gatewayRefundResponseJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&refund.ID,
		&refund.PaymentTransactionID,
		&refund.OrderID,
		&refund.RequestedBy,
		&refund.RequestedAmount,
		&refund.Reason,
		&proofImagesJSON,
		&refund.Status,
		&refund.ApprovedBy,
		&refund.ApprovedAt,
		&refund.AdminNotes,
		&refund.RejectedBy,
		&refund.RejectedAt,
		&refund.RejectionReason,
		&refund.GatewayRefundID,
		&gatewayRefundResponseJSON,
		&refund.RequestedAt,
		&refund.ProcessingAt,
		&refund.CompletedAt,
		&refund.FailedAt,
		&refund.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrRefundRequestNotFound
		}
		return nil, fmt.Errorf("failed to get refund request: %w", err)
	}

	// Unmarshal JSONB fields
	if proofImagesJSON != nil {
		if err := json.Unmarshal(proofImagesJSON, &refund.ProofImages); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proof_images: %w", err)
		}
	}

	if gatewayRefundResponseJSON != nil {
		if err := json.Unmarshal(gatewayRefundResponseJSON, &refund.GatewayRefundResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gateway_refund_response: %w", err)
		}
	}

	return refund, nil
}

// GetByPaymentID gets active refund request for a payment
// Active = pending, approved, or processing (not rejected/completed/failed)
func (r *refundRepository) GetByPaymentID(
	ctx context.Context,
	paymentID uuid.UUID,
) (*model.RefundRequest, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, requested_by,
			requested_amount, reason, proof_images, status,
			approved_by, approved_at, admin_notes,
			rejected_by, rejected_at, rejection_reason,
			gateway_refund_id, gateway_refund_response,
			requested_at, processing_at, completed_at, failed_at, updated_at
		FROM refund_requests
		WHERE payment_transaction_id = $1
		AND status IN ('pending', 'approved', 'processing')
		ORDER BY requested_at DESC
		LIMIT 1
	`

	refund := &model.RefundRequest{}
	var proofImagesJSON, gatewayRefundResponseJSON []byte

	err := r.pool.QueryRow(ctx, query, paymentID).Scan(
		&refund.ID,
		&refund.PaymentTransactionID,
		&refund.OrderID,
		&refund.RequestedBy,
		&refund.RequestedAmount,
		&refund.Reason,
		&proofImagesJSON,
		&refund.Status,
		&refund.ApprovedBy,
		&refund.ApprovedAt,
		&refund.AdminNotes,
		&refund.RejectedBy,
		&refund.RejectedAt,
		&refund.RejectionReason,
		&refund.GatewayRefundID,
		&gatewayRefundResponseJSON,
		&refund.RequestedAt,
		&refund.ProcessingAt,
		&refund.CompletedAt,
		&refund.FailedAt,
		&refund.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrRefundRequestNotFound
		}
		return nil, fmt.Errorf("failed to get refund request: %w", err)
	}

	// Unmarshal JSONB
	if proofImagesJSON != nil {
		json.Unmarshal(proofImagesJSON, &refund.ProofImages)
	}
	if gatewayRefundResponseJSON != nil {
		json.Unmarshal(gatewayRefundResponseJSON, &refund.GatewayRefundResponse)
	}

	return refund, nil
}

// Approve approves refund request
// Business Logic:
// 1. Update status to 'approved'
// 2. Set approved_by and approved_at
// 3. Set admin_notes if provided
// 4. Trigger will update payment_transactions.refund_amount
func (r *refundRepository) Approve(
	ctx context.Context,
	id uuid.UUID,
	approvedBy uuid.UUID,
	notes *string,
) error {
	query := `
		UPDATE refund_requests
		SET status = 'approved',
			approved_by = $2,
			approved_at = NOW(),
			admin_notes = $3,
			updated_at = NOW()
		WHERE id = $1
		AND status = 'pending'
	`

	result, err := r.pool.Exec(ctx, query, id, approvedBy, notes)
	if err != nil {
		return fmt.Errorf("failed to approve refund request: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Either not found or not in pending status
		// Check which case
		var currentStatus string
		checkQuery := `SELECT status FROM refund_requests WHERE id = $1`
		err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&currentStatus)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.ErrRefundRequestNotFound
			}
			return fmt.Errorf("failed to check refund status: %w", err)
		}

		// Refund exists but not in pending status
		return model.ErrCannotApproveRefund
	}

	return nil
}

// Reject rejects refund request
func (r *refundRepository) Reject(
	ctx context.Context,
	id uuid.UUID,
	rejectedBy uuid.UUID,
	reason string,
) error {
	query := `
		UPDATE refund_requests
		SET status = 'rejected',
			rejected_by = $2,
			rejected_at = NOW(),
			rejection_reason = $3,
			updated_at = NOW()
		WHERE id = $1
		AND status = 'pending'
	`

	result, err := r.pool.Exec(ctx, query, id, rejectedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to reject refund request: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if refund exists
		var currentStatus string
		checkQuery := `SELECT status FROM refund_requests WHERE id = $1`
		err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&currentStatus)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.ErrRefundRequestNotFound
			}
			return fmt.Errorf("failed to check refund status: %w", err)
		}

		// Refund exists but not in pending status
		return model.ErrCannotRejectRefund
	}

	return nil
}

// UpdateGatewayRefund updates gateway refund details after initiating refund
// Called after calling VNPay/Momo refund API
func (r *refundRepository) UpdateGatewayRefund(
	ctx context.Context,
	id uuid.UUID,
	gatewayRefundID string,
	response map[string]interface{},
) error {
	query := `
		UPDATE refund_requests
		SET gateway_refund_id = $2,
			gateway_refund_response = $3,
			status = 'processing',
			processing_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal gateway response: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, id, gatewayRefundID, responseJSON)
	if err != nil {
		return fmt.Errorf("failed to update gateway refund: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrRefundRequestNotFound
	}

	return nil
}

// MarkAsCompleted marks refund as completed
// Called when gateway webhook confirms refund success
func (r *refundRepository) MarkAsCompleted(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE refund_requests
		SET status = 'completed',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		AND status = 'processing'
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark refund as completed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrRefundRequestNotFound
	}

	return nil
}

// MarkAsFailed marks refund as failed
// Called when gateway refund API fails
func (r *refundRepository) MarkAsFailed(
	ctx context.Context,
	id uuid.UUID,
	reason string,
) error {
	query := `
		UPDATE refund_requests
		SET status = 'failed',
			admin_notes = CASE 
				WHEN admin_notes IS NULL THEN $2
				ELSE admin_notes || E'\n\nFailed: ' || $2
			END,
			failed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("failed to mark refund as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrRefundRequestNotFound
	}

	return nil
}

// ListPendingRefunds lists pending refund requests for admin dashboard
// Used to show refunds waiting for approval
func (r *refundRepository) ListPendingRefunds(
	ctx context.Context,
	page, limit int,
) ([]*model.RefundRequest, int, error) {
	// Count total pending
	countQuery := `SELECT COUNT(*) FROM refund_requests WHERE status = 'pending'`
	var total int
	err := r.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pending refunds: %w", err)
	}

	// Get paginated list
	query := `
		SELECT 
			id, payment_transaction_id, order_id, requested_by,
			requested_amount, reason, proof_images, status,
			requested_at, updated_at
		FROM refund_requests
		WHERE status = 'pending'
		ORDER BY requested_at ASC
		LIMIT $1 OFFSET $2
	`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pending refunds: %w", err)
	}
	defer rows.Close()

	var refunds []*model.RefundRequest
	for rows.Next() {
		refund := &model.RefundRequest{}
		var proofImagesJSON []byte

		err := rows.Scan(
			&refund.ID,
			&refund.PaymentTransactionID,
			&refund.OrderID,
			&refund.RequestedBy,
			&refund.RequestedAmount,
			&refund.Reason,
			&proofImagesJSON,
			&refund.Status,
			&refund.RequestedAt,
			&refund.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan refund: %w", err)
		}

		// Unmarshal proof_images
		if proofImagesJSON != nil {
			json.Unmarshal(proofImagesJSON, &refund.ProofImages)
		}

		refunds = append(refunds, refund)
	}

	return refunds, total, nil
}

// HasPendingRefund checks if payment has pending refund request
// Used to prevent duplicate refund requests
func (r *refundRepository) HasPendingRefund(
	ctx context.Context,
	paymentID uuid.UUID,
) (bool, error) {
	// Use database function for efficiency
	query := `SELECT has_pending_refund_request($1)`

	var hasPending bool
	err := r.pool.QueryRow(ctx, query, paymentID).Scan(&hasPending)
	if err != nil {
		return false, fmt.Errorf("failed to check pending refund: %w", err)
	}

	return hasPending, nil
}

// =====================================================
// ADDITIONAL ADMIN METHODS
// =====================================================

// GetRefundWithDetails gets refund with payment and order details
// Used for admin refund detail page
func (r *refundRepository) GetRefundWithDetails(
	ctx context.Context,
	id uuid.UUID,
) (*model.RefundRequest, map[string]interface{}, error) {
	query := `
		SELECT 
			rr.id, rr.payment_transaction_id, rr.order_id, rr.requested_by,
			rr.requested_amount, rr.reason, rr.proof_images, rr.status,
			rr.approved_by, rr.approved_at, rr.admin_notes,
			rr.rejected_by, rr.rejected_at, rr.rejection_reason,
			rr.gateway_refund_id, rr.gateway_refund_response,
			rr.requested_at, rr.processing_at, rr.completed_at, rr.failed_at, rr.updated_at,
			-- Payment details
			pt.gateway, pt.amount as payment_amount, pt.transaction_id,
			-- Order details
			o.order_number,
			-- User details
			u.email as user_email, u.name as user_name
		FROM refund_requests rr
		INNER JOIN payment_transactions pt ON rr.payment_transaction_id = pt.id
		INNER JOIN orders o ON rr.order_id = o.id
		INNER JOIN users u ON rr.requested_by = u.id
		WHERE rr.id = $1
	`

	refund := &model.RefundRequest{}
	var proofImagesJSON, gatewayRefundResponseJSON []byte
	var gateway, transactionID, orderNumber, userEmail, userName string
	var paymentAmount float64

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&refund.ID,
		&refund.PaymentTransactionID,
		&refund.OrderID,
		&refund.RequestedBy,
		&refund.RequestedAmount,
		&refund.Reason,
		&proofImagesJSON,
		&refund.Status,
		&refund.ApprovedBy,
		&refund.ApprovedAt,
		&refund.AdminNotes,
		&refund.RejectedBy,
		&refund.RejectedAt,
		&refund.RejectionReason,
		&refund.GatewayRefundID,
		&gatewayRefundResponseJSON,
		&refund.RequestedAt,
		&refund.ProcessingAt,
		&refund.CompletedAt,
		&refund.FailedAt,
		&refund.UpdatedAt,
		// Additional details
		&gateway,
		&paymentAmount,
		&transactionID,
		&orderNumber,
		&userEmail,
		&userName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, model.ErrRefundRequestNotFound
		}
		return nil, nil, fmt.Errorf("failed to get refund with details: %w", err)
	}

	// Unmarshal JSONB
	if proofImagesJSON != nil {
		json.Unmarshal(proofImagesJSON, &refund.ProofImages)
	}
	if gatewayRefundResponseJSON != nil {
		json.Unmarshal(gatewayRefundResponseJSON, &refund.GatewayRefundResponse)
	}

	// Build details map
	details := map[string]interface{}{
		"payment": map[string]interface{}{
			"gateway":        gateway,
			"amount":         paymentAmount,
			"transaction_id": transactionID,
		},
		"order": map[string]interface{}{
			"order_number": orderNumber,
		},
		"user": map[string]interface{}{
			"email": userEmail,
			"name":  userName,
		},
	}

	return refund, details, nil
}

// ListAllRefunds lists all refund requests with filters (admin)
func (r *refundRepository) ListAllRefunds(
	ctx context.Context,
	filters map[string]interface{},
	page, limit int,
) ([]*model.RefundRequest, int, error) {
	query := `
		SELECT 
			rr.id, rr.payment_transaction_id, rr.order_id, rr.requested_by,
			rr.requested_amount, rr.reason, rr.status,
			rr.requested_at, rr.approved_at, rr.rejected_at, rr.completed_at
		FROM refund_requests rr
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Filter by status
	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND rr.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Filter by date range
	if fromDate, ok := filters["from_date"].(time.Time); ok {
		query += fmt.Sprintf(" AND rr.requested_at >= $%d", argIndex)
		args = append(args, fromDate)
		argIndex++
	}

	if toDate, ok := filters["to_date"].(time.Time); ok {
		query += fmt.Sprintf(" AND rr.requested_at <= $%d", argIndex)
		args = append(args, toDate)
		argIndex++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count refunds: %w", err)
	}

	// Add pagination
	query += " ORDER BY rr.requested_at DESC"
	offset := (page - 1) * limit
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list refunds: %w", err)
	}
	defer rows.Close()

	var refunds []*model.RefundRequest
	for rows.Next() {
		refund := &model.RefundRequest{}
		err := rows.Scan(
			&refund.ID,
			&refund.PaymentTransactionID,
			&refund.OrderID,
			&refund.RequestedBy,
			&refund.RequestedAmount,
			&refund.Reason,
			&refund.Status,
			&refund.RequestedAt,
			&refund.ApprovedAt,
			&refund.RejectedAt,
			&refund.CompletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan refund: %w", err)
		}
		refunds = append(refunds, refund)
	}

	return refunds, total, nil
}
