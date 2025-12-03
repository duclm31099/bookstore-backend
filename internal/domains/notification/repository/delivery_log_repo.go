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

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// DELIVERY LOG REPOSITORY IMPLEMENTATION
// ================================================

type deliveryLogRepository struct {
	db *pgxpool.Pool
}

func NewDeliveryLogRepository(db *pgxpool.Pool) DeliveryLogRepository {
	return &deliveryLogRepository{db: db}
}

// Create creates a new delivery log
func (r *deliveryLogRepository) Create(ctx context.Context, log *model.DeliveryLog) error {
	query := `
		INSERT INTO notification_delivery_logs (
			id, notification_id, channel, attempt_number, status,
			recipient, provider, provider_message_id, provider_response,
			error_code, error_message,
			queued_at, processing_at, sent_at, delivered_at,
			opened_at, clicked_at, failed_at, retry_after,
			max_retries, estimated_cost
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
		RETURNING created_at
	`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		log.ID, log.NotificationID, log.Channel, log.AttemptNumber, log.Status,
		log.Recipient, log.Provider, log.ProviderMessageID, log.ProviderResponse,
		log.ErrorCode, log.ErrorMessage,
		log.QueuedAt, log.ProcessingAt, log.SentAt, log.DeliveredAt,
		log.OpenedAt, log.ClickedAt, log.FailedAt, log.RetryAfter,
		log.MaxRetries, log.EstimatedCost,
	).Scan(&log.CreatedAt)

	if err != nil {
		return fmt.Errorf("create delivery log: %w", err)
	}

	return nil
}

// CreateWithTx creates delivery log within transaction
func (r *deliveryLogRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, log *model.DeliveryLog) error {
	query := `
		INSERT INTO notification_delivery_logs (
			id, notification_id, channel, attempt_number, status,
			recipient, provider, provider_message_id, provider_response,
			error_code, error_message,
			queued_at, processing_at, sent_at, delivered_at,
			opened_at, clicked_at, failed_at, retry_after,
			max_retries, estimated_cost
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
		RETURNING created_at
	`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	err := tx.QueryRow(ctx, query,
		log.ID, log.NotificationID, log.Channel, log.AttemptNumber, log.Status,
		log.Recipient, log.Provider, log.ProviderMessageID, log.ProviderResponse,
		log.ErrorCode, log.ErrorMessage,
		log.QueuedAt, log.ProcessingAt, log.SentAt, log.DeliveredAt,
		log.OpenedAt, log.ClickedAt, log.FailedAt, log.RetryAfter,
		log.MaxRetries, log.EstimatedCost,
	).Scan(&log.CreatedAt)

	if err != nil {
		return fmt.Errorf("create delivery log with tx: %w", err)
	}

	return nil
}

// GetByID retrieves delivery log by ID
func (r *deliveryLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DeliveryLog, error) {
	query := `
		SELECT 
			id, notification_id, channel, attempt_number, status,
			recipient, provider, provider_message_id, provider_response,
			error_code, error_message,
			queued_at, processing_at, sent_at, delivered_at,
			opened_at, clicked_at, failed_at, retry_after,
			max_retries, estimated_cost, created_at
		FROM notification_delivery_logs
		WHERE id = $1
	`
	var responseBytes []byte
	var log model.DeliveryLog
	err := r.db.QueryRow(ctx, query, id).Scan(
		&log.ID, &log.NotificationID, &log.Channel, &log.AttemptNumber, &log.Status,
		&log.Recipient, &log.Provider, &log.ProviderMessageID, &responseBytes,
		&log.ErrorCode, &log.ErrorMessage,
		&log.QueuedAt, &log.ProcessingAt, &log.SentAt, &log.DeliveredAt,
		&log.OpenedAt, &log.ClickedAt, &log.FailedAt, &log.RetryAfter,
		&log.MaxRetries, &log.EstimatedCost, &log.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("delivery log not found")
		}
		return nil, fmt.Errorf("get delivery log by id: %w", err)
	}

	if len(responseBytes) > 0 {
		if err := json.Unmarshal(responseBytes, &log.ProviderResponse); err != nil {
			return nil, fmt.Errorf("unmarshal provider response: %w", err)
		}
	}

	return &log, nil
}

// Update updates delivery log
func (r *deliveryLogRepository) Update(ctx context.Context, log *model.DeliveryLog) error {
	query := `
		UPDATE notification_delivery_logs
		SET 
			status = $2,
			provider_message_id = $3,
			provider_response = $4,
			error_code = $5,
			error_message = $6,
			processing_at = $7,
			sent_at = $8,
			delivered_at = $9,
			opened_at = $10,
			clicked_at = $11,
			failed_at = $12,
			retry_after = $13
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		log.ID, log.Status, log.ProviderMessageID, log.ProviderResponse,
		log.ErrorCode, log.ErrorMessage,
		log.ProcessingAt, log.SentAt, log.DeliveredAt,
		log.OpenedAt, log.ClickedAt, log.FailedAt, log.RetryAfter,
	)

	if err != nil {
		return fmt.Errorf("update delivery log: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery log not found")
	}

	return nil
}

// ListByNotificationID retrieves all delivery logs for a notification
func (r *deliveryLogRepository) ListByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]model.DeliveryLog, error) {
	query := `
		SELECT 
			id, notification_id, channel, attempt_number, status,
			recipient, provider, provider_message_id, provider_response,
			error_code, error_message,
			queued_at, processing_at, sent_at, delivered_at,
			opened_at, clicked_at, failed_at, retry_after,
			max_retries, estimated_cost, created_at
		FROM notification_delivery_logs
		WHERE notification_id = $1
		ORDER BY attempt_number DESC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, notificationID)
	if err != nil {
		return nil, fmt.Errorf("list delivery logs by notification id: %w", err)
	}
	defer rows.Close()

	return scanDeliveryLogs(rows)
}

// ListFailedRetries retrieves failed logs that need retry
func (r *deliveryLogRepository) ListFailedRetries(ctx context.Context, limit int) ([]model.DeliveryLog, error) {
	query := `
		SELECT 
			id, notification_id, channel, attempt_number, status,
			recipient, provider, provider_message_id, provider_response,
			error_code, error_message,
			queued_at, processing_at, sent_at, delivered_at,
			opened_at, clicked_at, failed_at, retry_after,
			max_retries, estimated_cost, created_at
		FROM notification_delivery_logs
		WHERE status = $1
		AND retry_after IS NOT NULL
		AND retry_after <= NOW()
		AND attempt_number < max_retries
		ORDER BY retry_after ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, model.DeliveryStatusFailed, limit)
	if err != nil {
		return nil, fmt.Errorf("list failed retries: %w", err)
	}
	defer rows.Close()

	return scanDeliveryLogs(rows)
}

// UpdateStatus updates delivery log status
func (r *deliveryLogRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorCode, errorMessage *string) error {
	query := `
		UPDATE notification_delivery_logs
		SET 
			status = $2,
			error_code = $3,
			error_message = $4,
			failed_at = CASE WHEN $2 = 'failed' THEN NOW() ELSE failed_at END
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, status, errorCode, errorMessage)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery log not found")
	}

	return nil
}

// MarkAsDelivered marks delivery log as delivered
func (r *deliveryLogRepository) MarkAsDelivered(ctx context.Context, id uuid.UUID, providerMessageID string) error {
	query := `
		UPDATE notification_delivery_logs
		SET 
			status = $2,
			provider_message_id = $3,
			delivered_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, model.DeliveryStatusDelivered, providerMessageID)
	if err != nil {
		return fmt.Errorf("mark as delivered: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery log not found")
	}

	return nil
}

// MarkAsFailed marks delivery log as failed
func (r *deliveryLogRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) error {
	query := `
		UPDATE notification_delivery_logs
		SET 
			status = $2,
			error_code = $3,
			error_message = $4,
			failed_at = NOW(),
			retry_after = CASE 
				WHEN attempt_number < max_retries 
				THEN NOW() + INTERVAL '5 minutes' * POWER(2, attempt_number)
				ELSE NULL
			END
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, model.DeliveryStatusFailed, errorCode, errorMessage)
	if err != nil {
		return fmt.Errorf("mark as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delivery log not found")
	}

	return nil
}

// GetDeliveryRate calculates delivery success rate
func (r *deliveryLogRepository) GetDeliveryRate(ctx context.Context, from, to time.Time, channel *string) (float64, error) {
	whereClause := "WHERE created_at BETWEEN $1 AND $2"
	args := []interface{}{from, to}

	if channel != nil {
		whereClause += " AND channel = $3"
		args = append(args, *channel)
	}

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) FILTER (WHERE status IN ('delivered', 'sent'))::FLOAT / 
			NULLIF(COUNT(*)::FLOAT, 0) * 100 as delivery_rate
		FROM notification_delivery_logs
		%s
	`, whereClause)

	var rate float64
	err := r.db.QueryRow(ctx, query, args...).Scan(&rate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get delivery rate: %w", err)
	}

	return rate, nil
}

// CountByStatus counts delivery logs by status
func (r *deliveryLogRepository) CountByStatus(ctx context.Context, status string, from, to time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notification_delivery_logs
		WHERE status = $1
		AND created_at BETWEEN $2 AND $3
	`

	var count int
	err := r.db.QueryRow(ctx, query, status, from, to).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by status: %w", err)
	}

	return count, nil
}

// Helper function to scan delivery logs
func scanDeliveryLogs(rows pgx.Rows) ([]model.DeliveryLog, error) {
	var logs []model.DeliveryLog
	var providerResponseBytes []byte
	for rows.Next() {
		var log model.DeliveryLog
		err := rows.Scan(
			&log.ID, &log.NotificationID, &log.Channel, &log.AttemptNumber, &log.Status,
			&log.Recipient, &log.Provider, &log.ProviderMessageID, &providerResponseBytes,
			&log.ErrorCode, &log.ErrorMessage,
			&log.QueuedAt, &log.ProcessingAt, &log.SentAt, &log.DeliveredAt,
			&log.OpenedAt, &log.ClickedAt, &log.FailedAt, &log.RetryAfter,
			&log.MaxRetries, &log.EstimatedCost, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan delivery log: %w", err)
		}
		if len(providerResponseBytes) > 0 {
			if err := json.Unmarshal(providerResponseBytes, &log.ProviderResponse); err != nil {
				return nil, fmt.Errorf("unmarshal provider response: %w", err)
			}
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return logs, nil
}
