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
// WEBHOOK LOG REPOSITORY IMPLEMENTATION
// =====================================================
type webhookRepository struct {
	pool *pgxpool.Pool
}

func NewWebhookRepository(pool *pgxpool.Pool) WebhookRepoInterface {
	return &webhookRepository{pool: pool}
}

// =====================================================
// CREATE & LOGGING METHODS
// =====================================================

// Create creates webhook log
// This is called IMMEDIATELY when webhook is received (before processing)
// Purpose: Audit trail, debugging, and idempotency checking
func (r *webhookRepository) Create(
	ctx context.Context,
	log *model.PaymentWebhookLog,
) error {
	query := `
		INSERT INTO payment_webhook_logs (
			id, payment_transaction_id, order_id, gateway, webhook_event,
			headers, body, signature, is_valid, is_processed, received_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	// Serialize headers and body to JSONB
	headersJSON, err := json.Marshal(log.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	bodyJSON, err := json.Marshal(log.Body)
	if err != nil {
		return fmt.Errorf("failed to marshal body: %w", err)
	}

	_, err = r.pool.Exec(ctx, query,
		log.ID,
		log.PaymentTransactionID,
		log.OrderID,
		log.Gateway,
		log.WebhookEvent,
		headersJSON,
		bodyJSON,
		log.Signature,
		log.IsValid,
		log.IsProcessed,
		log.ReceivedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create webhook log: %w", err)
	}

	return nil
}

// =====================================================
// IDEMPOTENCY CHECKING
// =====================================================

// CheckIdempotency checks if webhook already processed
// This is CRITICAL for preventing duplicate processing
//
// Business Logic:
// - A webhook is uniquely identified by (gateway, event, transaction_id)
// - If same webhook arrives multiple times (network retry), process only once
// - Uses unique index: idx_payment_webhook_logs_idempotency
//
// Returns:
// - true: webhook already processed (skip processing)
// - false: new webhook (proceed with processing)
func (r *webhookRepository) CheckIdempotency(
	ctx context.Context,
	gateway, event, transactionID string,
) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM payment_webhook_logs
			WHERE gateway = $1
			AND webhook_event = $2
			AND body->>'transaction_id' = $3
			AND is_processed = true
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, gateway, event, transactionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	return exists, nil
}

// =====================================================
// STATUS UPDATE METHODS
// =====================================================

// MarkAsProcessed marks webhook as successfully processed
// Called after payment/refund status updated successfully
func (r *webhookRepository) MarkAsProcessed(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE payment_webhook_logs
		SET is_processed = true
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark webhook as processed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook log not found: %s", id)
	}

	return nil
}

// MarkAsInvalid marks webhook as invalid (signature verification failed)
// This indicates potential fraud or misconfiguration
func (r *webhookRepository) MarkAsInvalid(
	ctx context.Context,
	id uuid.UUID,
	reason string,
) error {
	query := `
		UPDATE payment_webhook_logs
		SET is_valid = false,
			processing_error = $2
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("failed to mark webhook as invalid: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook log not found: %s", id)
	}

	return nil
}

// MarkProcessingError marks webhook processing error
// Used when webhook is valid but processing failed (e.g., database error)
func (r *webhookRepository) MarkProcessingError(
	ctx context.Context,
	id uuid.UUID,
	errorMsg string,
) error {
	query := `
		UPDATE payment_webhook_logs
		SET processing_error = $2
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to mark processing error: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook log not found: %s", id)
	}

	return nil
}

// =====================================================
// RETRY MECHANISM
// =====================================================

// GetFailedWebhooks gets webhooks that failed processing (for retry)
//
// Business Logic:
// - Get webhooks where is_processed = false
// - AND is_valid = true (signature OK, but processing failed)
// - Within last 24 hours (old webhooks are abandoned)
// - Limit to batch size (e.g., 100) to avoid overwhelming system
//
// Used by background job to retry failed webhook processing
func (r *webhookRepository) GetFailedWebhooks(
	ctx context.Context,
	limit int,
) ([]*model.PaymentWebhookLog, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, gateway, webhook_event,
			headers, body, signature, is_valid, is_processed,
			processing_error, received_at
		FROM payment_webhook_logs
		WHERE is_processed = false
		AND is_valid = true
		AND received_at > NOW() - INTERVAL '24 hours'
		ORDER BY received_at ASC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.PaymentWebhookLog
	for rows.Next() {
		webhook := &model.PaymentWebhookLog{}
		var headersJSON, bodyJSON []byte

		err := rows.Scan(
			&webhook.ID,
			&webhook.PaymentTransactionID,
			&webhook.OrderID,
			&webhook.Gateway,
			&webhook.WebhookEvent,
			&headersJSON,
			&bodyJSON,
			&webhook.Signature,
			&webhook.IsValid,
			&webhook.IsProcessed,
			&webhook.ProcessingError,
			&webhook.ReceivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		// Unmarshal JSONB
		if headersJSON != nil {
			json.Unmarshal(headersJSON, &webhook.Headers)
		}
		if bodyJSON != nil {
			json.Unmarshal(bodyJSON, &webhook.Body)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// =====================================================
// ADMIN DEBUGGING METHODS
// =====================================================

// ListByPaymentID lists webhook logs for a payment (admin)
// Used for debugging payment issues
func (r *webhookRepository) ListByPaymentID(
	ctx context.Context,
	paymentID uuid.UUID,
) ([]*model.PaymentWebhookLog, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, gateway, webhook_event,
			headers, body, signature, is_valid, is_processed,
			processing_error, received_at
		FROM payment_webhook_logs
		WHERE payment_transaction_id = $1
		ORDER BY received_at ASC
	`

	rows, err := r.pool.Query(ctx, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks by payment: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.PaymentWebhookLog
	for rows.Next() {
		webhook := &model.PaymentWebhookLog{}
		var headersJSON, bodyJSON []byte

		err := rows.Scan(
			&webhook.ID,
			&webhook.PaymentTransactionID,
			&webhook.OrderID,
			&webhook.Gateway,
			&webhook.WebhookEvent,
			&headersJSON,
			&bodyJSON,
			&webhook.Signature,
			&webhook.IsValid,
			&webhook.IsProcessed,
			&webhook.ProcessingError,
			&webhook.ReceivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		// Unmarshal JSONB
		if headersJSON != nil {
			json.Unmarshal(headersJSON, &webhook.Headers)
		}
		if bodyJSON != nil {
			json.Unmarshal(bodyJSON, &webhook.Body)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// GetByID gets webhook log by ID
// Used for admin webhook detail page
func (r *webhookRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.PaymentWebhookLog, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, gateway, webhook_event,
			headers, body, signature, is_valid, is_processed,
			processing_error, received_at
		FROM payment_webhook_logs
		WHERE id = $1
	`

	webhook := &model.PaymentWebhookLog{}
	var headersJSON, bodyJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID,
		&webhook.PaymentTransactionID,
		&webhook.OrderID,
		&webhook.Gateway,
		&webhook.WebhookEvent,
		&headersJSON,
		&bodyJSON,
		&webhook.Signature,
		&webhook.IsValid,
		&webhook.IsProcessed,
		&webhook.ProcessingError,
		&webhook.ReceivedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("webhook log not found")
		}
		return nil, fmt.Errorf("failed to get webhook log: %w", err)
	}

	// Unmarshal JSONB
	if headersJSON != nil {
		json.Unmarshal(headersJSON, &webhook.Headers)
	}
	if bodyJSON != nil {
		json.Unmarshal(bodyJSON, &webhook.Body)
	}

	return webhook, nil
}

// =====================================================
// STATISTICS & MONITORING
// =====================================================

// GetWebhookStats gets webhook statistics for monitoring
// Used by admin dashboard to monitor webhook health
func (r *webhookRepository) GetWebhookStats(
	ctx context.Context,
	gateway string,
	fromDate, toDate time.Time,
) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_webhooks,
			COUNT(*) FILTER (WHERE is_valid = true) as valid_webhooks,
			COUNT(*) FILTER (WHERE is_valid = false) as invalid_webhooks,
			COUNT(*) FILTER (WHERE is_processed = true) as processed_webhooks,
			COUNT(*) FILTER (WHERE is_processed = false AND is_valid = true) as failed_webhooks,
			COUNT(*) FILTER (WHERE processing_error IS NOT NULL) as error_webhooks
		FROM payment_webhook_logs
		WHERE gateway = $1
		AND received_at BETWEEN $2 AND $3
	`

	var stats struct {
		TotalWebhooks     int
		ValidWebhooks     int
		InvalidWebhooks   int
		ProcessedWebhooks int
		FailedWebhooks    int
		ErrorWebhooks     int
	}

	err := r.pool.QueryRow(ctx, query, gateway, fromDate, toDate).Scan(
		&stats.TotalWebhooks,
		&stats.ValidWebhooks,
		&stats.InvalidWebhooks,
		&stats.ProcessedWebhooks,
		&stats.FailedWebhooks,
		&stats.ErrorWebhooks,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get webhook stats: %w", err)
	}

	result := map[string]interface{}{
		"total_webhooks":     stats.TotalWebhooks,
		"valid_webhooks":     stats.ValidWebhooks,
		"invalid_webhooks":   stats.InvalidWebhooks,
		"processed_webhooks": stats.ProcessedWebhooks,
		"failed_webhooks":    stats.FailedWebhooks,
		"error_webhooks":     stats.ErrorWebhooks,
		"success_rate":       0.0,
	}

	// Calculate success rate
	if stats.TotalWebhooks > 0 {
		result["success_rate"] = float64(stats.ProcessedWebhooks) / float64(stats.TotalWebhooks) * 100
	}

	return result, nil
}

// ListRecentWebhooks lists recent webhooks for monitoring
// Used by admin dashboard to see latest webhook activity
func (r *webhookRepository) ListRecentWebhooks(
	ctx context.Context,
	limit int,
) ([]*model.PaymentWebhookLog, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, gateway, webhook_event,
			is_valid, is_processed, processing_error, received_at
		FROM payment_webhook_logs
		ORDER BY received_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.PaymentWebhookLog
	for rows.Next() {
		webhook := &model.PaymentWebhookLog{}

		err := rows.Scan(
			&webhook.ID,
			&webhook.PaymentTransactionID,
			&webhook.OrderID,
			&webhook.Gateway,
			&webhook.WebhookEvent,
			&webhook.IsValid,
			&webhook.IsProcessed,
			&webhook.ProcessingError,
			&webhook.ReceivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// =====================================================
// CLEANUP METHODS
// =====================================================

// DeleteOldWebhooks deletes webhook logs older than retention period
// Used by cleanup job to prevent table bloat
// Default retention: 90 days
func (r *webhookRepository) DeleteOldWebhooks(
	ctx context.Context,
	retentionDays int,
) (int, error) {
	query := `
		DELETE FROM payment_webhook_logs
		WHERE received_at < NOW() - INTERVAL '$1 days'
		RETURNING id
	`

	rows, err := r.pool.Query(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old webhooks: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	return count, nil
}

// =====================================================
// ADVANCED DEBUGGING METHODS
// =====================================================

// FindDuplicateWebhooks finds potential duplicate webhooks
// Used for debugging idempotency issues
func (r *webhookRepository) FindDuplicateWebhooks(
	ctx context.Context,
	gateway, transactionID string,
) ([]*model.PaymentWebhookLog, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, gateway, webhook_event,
			is_valid, is_processed, processing_error, received_at
		FROM payment_webhook_logs
		WHERE gateway = $1
		AND body->>'transaction_id' = $2
		ORDER BY received_at ASC
	`

	rows, err := r.pool.Query(ctx, query, gateway, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find duplicate webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.PaymentWebhookLog
	for rows.Next() {
		webhook := &model.PaymentWebhookLog{}

		err := rows.Scan(
			&webhook.ID,
			&webhook.PaymentTransactionID,
			&webhook.OrderID,
			&webhook.Gateway,
			&webhook.WebhookEvent,
			&webhook.IsValid,
			&webhook.IsProcessed,
			&webhook.ProcessingError,
			&webhook.ReceivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// GetUnprocessedWebhooksByPayment gets unprocessed webhooks for a payment
// Used when payment status is inconsistent with expected webhook
func (r *webhookRepository) GetUnprocessedWebhooksByPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) ([]*model.PaymentWebhookLog, error) {
	query := `
		SELECT 
			id, payment_transaction_id, order_id, gateway, webhook_event,
			headers, body, signature, is_valid, is_processed,
			processing_error, received_at
		FROM payment_webhook_logs
		WHERE payment_transaction_id = $1
		AND is_processed = false
		ORDER BY received_at ASC
	`

	rows, err := r.pool.Query(ctx, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.PaymentWebhookLog
	for rows.Next() {
		webhook := &model.PaymentWebhookLog{}
		var headersJSON, bodyJSON []byte

		err := rows.Scan(
			&webhook.ID,
			&webhook.PaymentTransactionID,
			&webhook.OrderID,
			&webhook.Gateway,
			&webhook.WebhookEvent,
			&headersJSON,
			&bodyJSON,
			&webhook.Signature,
			&webhook.IsValid,
			&webhook.IsProcessed,
			&webhook.ProcessingError,
			&webhook.ReceivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		// Unmarshal JSONB
		if headersJSON != nil {
			json.Unmarshal(headersJSON, &webhook.Headers)
		}
		if bodyJSON != nil {
			json.Unmarshal(bodyJSON, &webhook.Body)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}
