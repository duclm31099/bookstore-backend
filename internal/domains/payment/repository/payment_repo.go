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
	"github.com/shopspring/decimal"

	"bookstore-backend/internal/domains/payment/model"
)

// =====================================================
// PAYMENT REPOSITORY IMPLEMENTATION
// =====================================================
type ppRepository struct {
	pool *pgxpool.Pool
}

func NewppRepository(pool *pgxpool.Pool) PaymentRepoInteface {
	return &ppRepository{pool: pool}
}

// =====================================================
// TRANSACTION-AWARE METHODS
// =====================================================

// CreateWithTx creates payment transaction within provided transaction
func (r *ppRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, payment *model.PaymentTransaction) error {
	query := `
		INSERT INTO payment_transactions (
			id, order_id, gateway, amount, currency, status, 
			payment_details, retry_count, initiated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING created_at, updated_at
	`

	// Serialize payment_details to JSONB
	paymentDetailsJSON, err := json.Marshal(payment.PaymentDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal payment_details: %w", err)
	}

	err = tx.QueryRow(ctx, query,
		payment.ID,
		payment.OrderID,
		payment.Gateway,
		payment.Amount,
		payment.Currency,
		payment.Status,
		paymentDetailsJSON,
		payment.RetryCount,
		payment.InitiatedAt,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create payment transaction: %w", err)
	}

	return nil
}

// UpdateStatusWithTx updates payment status within transaction
func (r *ppRepository) UpdateStatusWithTx(
	ctx context.Context,
	tx pgx.Tx,
	paymentID uuid.UUID,
	status string,
	details map[string]interface{},
) error {
	query := `
		UPDATE payment_transactions
		SET status = $1,
			processing_at = CASE WHEN $1 = 'processing' THEN NOW() ELSE processing_at END,
			completed_at = CASE WHEN $1 = 'success' THEN NOW() ELSE completed_at END,
			failed_at = CASE WHEN $1 = 'failed' THEN NOW() ELSE failed_at END,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := tx.Exec(ctx, query, status, paymentID)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPaymentNotFound
	}

	return nil
}

// =====================================================
// STANDALONE METHODS
// =====================================================

// Create creates payment transaction
func (r *ppRepository) Create(ctx context.Context, payment *model.PaymentTransaction) error {
	query := `
		INSERT INTO payment_transactions (
			id, order_id, gateway, amount, currency, status, 
			payment_details, retry_count, initiated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING created_at, updated_at
	`

	paymentDetailsJSON, err := json.Marshal(payment.PaymentDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal payment_details: %w", err)
	}

	err = r.pool.QueryRow(ctx, query,
		payment.ID,
		payment.OrderID,
		payment.Gateway,
		payment.Amount,
		payment.Currency,
		payment.Status,
		paymentDetailsJSON,
		payment.RetryCount,
		payment.InitiatedAt,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create payment transaction: %w", err)
	}

	return nil
}

// GetByID gets payment transaction by ID
func (r *ppRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PaymentTransaction, error) {
	query := `
		SELECT 
			id, order_id, gateway, transaction_id, amount, currency, status,
			error_code, error_message, gateway_response, gateway_signature,
			payment_details, refund_amount, refund_reason, refunded_at,
			retry_count, initiated_at, processing_at, completed_at, failed_at,
			created_at, updated_at
		FROM payment_transactions
		WHERE id = $1
	`

	payment := &model.PaymentTransaction{}
	var gatewayResponseJSON, paymentDetailsJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Gateway,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.ErrorCode,
		&payment.ErrorMessage,
		&gatewayResponseJSON,
		&payment.GatewaySignature,
		&paymentDetailsJSON,
		&payment.RefundAmount,
		&payment.RefundReason,
		&payment.RefundedAt,
		&payment.RetryCount,
		&payment.InitiatedAt,
		&payment.ProcessingAt,
		&payment.CompletedAt,
		&payment.FailedAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Unmarshal JSONB fields
	if gatewayResponseJSON != nil {
		if err := json.Unmarshal(gatewayResponseJSON, &payment.GatewayResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gateway_response: %w", err)
		}
	}

	if paymentDetailsJSON != nil {
		if err := json.Unmarshal(paymentDetailsJSON, &payment.PaymentDetails); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payment_details: %w", err)
		}
	}

	return payment, nil
}

// GetByOrderID gets latest payment transaction for an order
func (r *ppRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*model.PaymentTransaction, error) {
	query := `
		SELECT 
			id, order_id, gateway, transaction_id, amount, currency, status,
			error_code, error_message, gateway_response, gateway_signature,
			payment_details, refund_amount, refund_reason, refunded_at,
			retry_count, initiated_at, processing_at, completed_at, failed_at,
			created_at, updated_at
		FROM payment_transactions
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	payment := &model.PaymentTransaction{}
	var gatewayResponseJSON, paymentDetailsJSON []byte

	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Gateway,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.ErrorCode,
		&payment.ErrorMessage,
		&gatewayResponseJSON,
		&payment.GatewaySignature,
		&paymentDetailsJSON,
		&payment.RefundAmount,
		&payment.RefundReason,
		&payment.RefundedAt,
		&payment.RetryCount,
		&payment.InitiatedAt,
		&payment.ProcessingAt,
		&payment.CompletedAt,
		&payment.FailedAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Unmarshal JSONB
	if gatewayResponseJSON != nil {
		json.Unmarshal(gatewayResponseJSON, &payment.GatewayResponse)
	}
	if paymentDetailsJSON != nil {
		json.Unmarshal(paymentDetailsJSON, &payment.PaymentDetails)
	}

	return payment, nil
}

// GetByTransactionID gets payment by gateway transaction ID
func (r *ppRepository) GetByTransactionID(
	ctx context.Context,
	gateway, transactionID string,
) (*model.PaymentTransaction, error) {
	query := `
		SELECT 
			id, order_id, gateway, transaction_id, amount, currency, status,
			error_code, error_message, gateway_response, gateway_signature,
			payment_details, refund_amount, refund_reason, refunded_at,
			retry_count, initiated_at, processing_at, completed_at, failed_at,
			created_at, updated_at
		FROM payment_transactions
		WHERE gateway = $1 AND transaction_id = $2
	`

	payment := &model.PaymentTransaction{}
	var gatewayResponseJSON, paymentDetailsJSON []byte

	err := r.pool.QueryRow(ctx, query, gateway, transactionID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Gateway,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.ErrorCode,
		&payment.ErrorMessage,
		&gatewayResponseJSON,
		&payment.GatewaySignature,
		&paymentDetailsJSON,
		&payment.RefundAmount,
		&payment.RefundReason,
		&payment.RefundedAt,
		&payment.RetryCount,
		&payment.InitiatedAt,
		&payment.ProcessingAt,
		&payment.CompletedAt,
		&payment.FailedAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Unmarshal JSONB
	if gatewayResponseJSON != nil {
		json.Unmarshal(gatewayResponseJSON, &payment.GatewayResponse)
	}
	if paymentDetailsJSON != nil {
		json.Unmarshal(paymentDetailsJSON, &payment.PaymentDetails)
	}

	return payment, nil
}

// MarkAsSuccess marks payment as successful
// This method encapsulates the complex update logic for successful payments
func (r *ppRepository) MarkAsSuccess(
	ctx context.Context,
	id uuid.UUID,
	transactionID string,
	gatewayResponse map[string]interface{},
	paymentDetails map[string]interface{},
) error {
	query := `
		UPDATE payment_transactions
		SET status = 'success',
			transaction_id = $2,
			gateway_response = $3,
			payment_details = $4,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	gatewayResponseJSON, _ := json.Marshal(gatewayResponse)
	paymentDetailsJSON, _ := json.Marshal(paymentDetails)

	result, err := r.pool.Exec(ctx, query, id, transactionID, gatewayResponseJSON, paymentDetailsJSON)
	if err != nil {
		return fmt.Errorf("failed to mark payment as success: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPaymentNotFound
	}

	return nil
}

// UpdateRefundAmount updates refund amount after refund processing
func (r *ppRepository) UpdateRefundAmount(
	ctx context.Context,
	id uuid.UUID,
	refundAmount decimal.Decimal,
	reason string,
) error {
	query := `
        UPDATE payment_transactions
        SET refund_amount = refund_amount + $2,
            refund_reason = $3,
            status = CASE 
                WHEN (refund_amount + $2) >= amount THEN 'refunded'
                ELSE status
            END,
            refunded_at = CASE
                WHEN refunded_at IS NULL THEN NOW()
                ELSE refunded_at
            END,
            updated_at = NOW()
        WHERE id = $1
    `

	result, err := r.pool.Exec(ctx, query, id, refundAmount, reason)
	if err != nil {
		return fmt.Errorf("failed to update refund amount: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPaymentNotFound
	}

	return nil
}

// MarkAsFailed marks payment as failed with error details
func (r *ppRepository) MarkAsFailed(
	ctx context.Context,
	id uuid.UUID,
	errorCode, errorMessage string,
) error {
	query := `
		UPDATE payment_transactions
		SET status = 'failed',
			error_code = $2,
			error_message = $3,
			failed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, errorCode, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to mark payment as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPaymentNotFound
	}

	return nil
}

// MarkAsCancelled marks payment as cancelled
func (r *ppRepository) MarkAsCancelled(
	ctx context.Context,
	id uuid.UUID,
	reason string,
) error {
	query := `
		UPDATE payment_transactions
		SET status = 'cancelled',
			error_code = 'PAY_TIMEOUT',
			error_message = $2,
			failed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("failed to mark payment as cancelled: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPaymentNotFound
	}

	return nil
}

// CheckRetryLimit checks if order can retry payment
// Returns: (canRetry bool, attemptCount int, error)
func (r *ppRepository) CheckRetryLimit(
	ctx context.Context,
	orderID uuid.UUID,
) (bool, int, error) {
	// Use database function for accurate check
	query := `SELECT can_retry_payment($1)`

	var canRetry bool
	err := r.pool.QueryRow(ctx, query, orderID).Scan(&canRetry)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check retry limit: %w", err)
	}

	// Get current attempt count
	countQuery := `
		SELECT COUNT(*)
		FROM payment_transactions
		WHERE order_id = $1
		AND status IN ('failed', 'cancelled')
	`

	var attemptCount int
	err = r.pool.QueryRow(ctx, countQuery, orderID).Scan(&attemptCount)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get attempt count: %w", err)
	}

	return canRetry, attemptCount, nil
}

// GetExpiredPayments gets payments that have exceeded timeout (15 minutes)
// Used by background job to auto-cancel expired payments
func (r *ppRepository) GetExpiredPayments(
	ctx context.Context,
	limit int,
) ([]*model.PaymentTransaction, error) {
	query := `
		SELECT 
			id, order_id, gateway, transaction_id, amount, currency, status,
			retry_count, initiated_at, created_at, updated_at
		FROM payment_transactions
		WHERE status IN ('pending', 'processing')
		AND gateway != 'cod'
		AND initiated_at < NOW() - INTERVAL '15 minutes'
		ORDER BY initiated_at ASC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.PaymentTransaction
	for rows.Next() {
		payment := &model.PaymentTransaction{}
		err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.Gateway,
			&payment.TransactionID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.RetryCount,
			&payment.InitiatedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	return payments, nil
}

// HasSuccessfulPayment checks if order has successful payment
func (r *ppRepository) HasSuccessfulPayment(
	ctx context.Context,
	orderID uuid.UUID,
) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM payment_transactions
			WHERE order_id = $1
			AND status = 'success'
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, orderID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check successful payment: %w", err)
	}

	return exists, nil
}

// UpdateStatus updates payment status (simple version)
func (r *ppRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
) error {
	query := `
		UPDATE payment_transactions
		SET status = $1,
			processing_at = CASE WHEN $1 = 'processing' THEN NOW() ELSE processing_at END,
			completed_at = CASE WHEN $1 = 'success' THEN NOW() ELSE completed_at END,
			failed_at = CASE WHEN $1 IN ('failed', 'cancelled') THEN NOW() ELSE failed_at END,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPaymentNotFound
	}

	return nil
}

// ListByUserID lists payments for a user with filters
func (r *ppRepository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filters map[string]interface{},
	page, limit int,
) ([]*model.PaymentTransaction, int, error) {
	// Build dynamic query with filters
	query := `
		SELECT 
			pt.id, pt.order_id, pt.gateway, pt.transaction_id, 
			pt.amount, pt.currency, pt.status,
			pt.completed_at, pt.created_at
		FROM payment_transactions pt
		INNER JOIN orders o ON pt.order_id = o.id
		WHERE o.user_id = $1
	`

	args := []interface{}{userID}
	argIndex := 2

	// Apply filters
	if orderID, ok := filters["order_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND pt.order_id = $%d", argIndex)
		args = append(args, orderID)
		argIndex++
	}

	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND pt.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if gateway, ok := filters["gateway"].(string); ok {
		query += fmt.Sprintf(" AND pt.gateway = $%d", argIndex)
		args = append(args, gateway)
		argIndex++
	}

	// Count total
	countQuery := query
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM ("+countQuery+") as count_query", args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	// Add pagination
	query += " ORDER BY pt.created_at DESC"
	offset := (page - 1) * limit
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.PaymentTransaction
	for rows.Next() {
		payment := &model.PaymentTransaction{}
		err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.Gateway,
			&payment.TransactionID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.CompletedAt,
			&payment.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	return payments, total, nil
}

func (r *ppRepository) AdminListPayments(
	ctx context.Context,
	filters map[string]interface{},
	page, limit int,
) ([]*model.PaymentTransaction, int, error) {
	// Base query with JOIN to get order and user info
	query := `
        SELECT 
            pt.id, pt.order_id, pt.gateway, pt.transaction_id,
            pt.amount, pt.currency, pt.status,
            pt.retry_count, pt.completed_at, pt.created_at,
            o.order_number,
            u.email as user_email
        FROM payment_transactions pt
        INNER JOIN orders o ON pt.order_id = o.id
        INNER JOIN users u ON o.user_id = u.id
        WHERE 1=1
    `

	args := []interface{}{}
	argIndex := 1

	// Filter: status
	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND pt.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Filter: gateway
	if gateway, ok := filters["gateway"].(string); ok {
		query += fmt.Sprintf(" AND pt.gateway = $%d", argIndex)
		args = append(args, gateway)
		argIndex++
	}

	// Filter: date range
	if fromDate, ok := filters["from_date"].(time.Time); ok {
		query += fmt.Sprintf(" AND pt.created_at >= $%d", argIndex)
		args = append(args, fromDate)
		argIndex++
	}

	if toDate, ok := filters["to_date"].(time.Time); ok {
		query += fmt.Sprintf(" AND pt.created_at <= $%d", argIndex)
		args = append(args, toDate)
		argIndex++
	}

	// Filter: amount range
	if minAmount, ok := filters["min_amount"].(float64); ok {
		query += fmt.Sprintf(" AND pt.amount >= $%d", argIndex)
		args = append(args, minAmount)
		argIndex++
	}

	if maxAmount, ok := filters["max_amount"].(float64); ok {
		query += fmt.Sprintf(" AND pt.amount <= $%d", argIndex)
		args = append(args, maxAmount)
		argIndex++
	}

	// Filter: search by order_number or transaction_id
	if search, ok := filters["search"].(string); ok {
		query += fmt.Sprintf(" AND (o.order_number ILIKE $%d OR pt.transaction_id ILIKE $%d)", argIndex, argIndex)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		argIndex++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	// Add pagination
	query += " ORDER BY pt.created_at DESC"
	offset := (page - 1) * limit
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.PaymentTransaction
	for rows.Next() {
		payment := &model.PaymentTransaction{}
		var orderNumber, userEmail string

		err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.Gateway,
			&payment.TransactionID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.RetryCount,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&orderNumber,
			&userEmail,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan payment: %w", err)
		}

		// Attach order info (need to add to entity or use separate DTO)
		// payment.OrderNumber = orderNumber
		// payment.UserEmail = userEmail

		payments = append(payments, payment)
	}

	return payments, total, nil
}

// AdminGetStatistics gets payment statistics
func (r *ppRepository) AdminGetStatistics(
	ctx context.Context,
	filters map[string]interface{},
) (*model.PaymentStatistics, error) {
	query := `
		SELECT 
			COALESCE(SUM(amount) FILTER (WHERE status = 'success'), 0) as total_amount,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count
		FROM payment_transactions
		WHERE 1=1
	`

	args := []interface{}{}
	// Apply date filters if provided
	// TODO: Add date range filters

	stats := &model.PaymentStatistics{}
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&stats.TotalAmount,
		&stats.SuccessCount,
		&stats.PendingCount,
		&stats.FailedCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}

// GetByOrderIDAndStatus gets payment by order and status
// Useful for checking if order has pending payment
func (r *ppRepository) GetByOrderIDAndStatus(
	ctx context.Context,
	orderID uuid.UUID,
	status string,
) (*model.PaymentTransaction, error) {
	query := `
        SELECT 
            id, order_id, gateway, transaction_id, amount, currency, status,
            retry_count, initiated_at, created_at, updated_at
        FROM payment_transactions
        WHERE order_id = $1 AND status = $2
        ORDER BY created_at DESC
        LIMIT 1
    `

	payment := &model.PaymentTransaction{}
	err := r.pool.QueryRow(ctx, query, orderID, status).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Gateway,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.RetryCount,
		&payment.InitiatedAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}
func (m *ppRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (m *ppRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (m *ppRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// MarkExpiredPaymentsAsCancelled bulk update expired payments
// Returns count of cancelled payments
func (r *ppRepository) MarkExpiredPaymentsAsCancelled(
	ctx context.Context,
	reason string,
) (int, error) {
	query := `
        UPDATE payment_transactions
        SET status = 'cancelled',
            error_code = 'PAY_TIMEOUT',
            error_message = $1,
            failed_at = NOW(),
            updated_at = NOW()
        WHERE status IN ('pending', 'processing')
        AND gateway != 'cod'
        AND initiated_at < NOW() - INTERVAL '15 minutes'
        RETURNING id
    `

	rows, err := r.pool.Query(ctx, query, reason)
	if err != nil {
		return 0, fmt.Errorf("failed to mark expired payments as cancelled: %w", err)
	}
	defer rows.Close()

	count := 0
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("failed to scan id: %w", err)
		}
		ids = append(ids, id)
		count++
	}

	return count, nil
}
func (r *ppRepository) GetByIDAndVerifyUser(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (*model.PaymentTransaction, error) {
	query := `
        SELECT 
            pt.id, pt.order_id, pt.gateway, pt.transaction_id, 
            pt.amount, pt.currency, pt.status,
            pt.error_code, pt.error_message, pt.gateway_response, 
            pt.gateway_signature, pt.payment_details, 
            pt.refund_amount, pt.refund_reason, pt.refunded_at,
            pt.retry_count, pt.initiated_at, pt.processing_at, 
            pt.completed_at, pt.failed_at, pt.created_at, pt.updated_at
        FROM payment_transactions pt
        INNER JOIN orders o ON pt.order_id = o.id
        WHERE pt.id = $1 AND o.user_id = $2
    `

	payment := &model.PaymentTransaction{}
	var gatewayResponseJSON, paymentDetailsJSON []byte

	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&payment.ID, &payment.OrderID, &payment.Gateway, &payment.TransactionID,
		&payment.Amount, &payment.Currency, &payment.Status,
		&payment.ErrorCode, &payment.ErrorMessage, &gatewayResponseJSON,
		&payment.GatewaySignature, &paymentDetailsJSON,
		&payment.RefundAmount, &payment.RefundReason, &payment.RefundedAt,
		&payment.RetryCount, &payment.InitiatedAt, &payment.ProcessingAt,
		&payment.CompletedAt, &payment.FailedAt, &payment.CreatedAt, &payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPaymentNotFound // Or ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Unmarshal JSONB
	if gatewayResponseJSON != nil {
		json.Unmarshal(gatewayResponseJSON, &payment.GatewayResponse)
	}
	if paymentDetailsJSON != nil {
		json.Unmarshal(paymentDetailsJSON, &payment.PaymentDetails)
	}

	return payment, nil
}
