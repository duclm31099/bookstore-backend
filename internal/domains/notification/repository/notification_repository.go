package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// NOTIFICATION REPOSITORY IMPLEMENTATION
// ================================================

type notificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{db: db}
}

// Create creates a new notification
func (r *notificationRepository) Create(ctx context.Context, n *model.Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, type, title, message, data,
			is_read, is_sent, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		RETURNING created_at, updated_at
	`

	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		n.ID, n.UserID, n.Type, n.Title, n.Message, n.Data,
		n.IsRead, n.IsSent, n.Channels, n.DeliveryStatus,
		n.ReferenceType, n.ReferenceID, n.IdempotencyKey,
		n.Priority, n.ExpiresAt, n.TemplateCode, n.TemplateVersion, n.TemplateData,
	).Scan(&n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotificationNotFound
		}
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}

// CreateWithTx creates notification within transaction
func (r *notificationRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, n *model.Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, type, title, message, data,
			is_read, is_sent, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		RETURNING created_at, updated_at
	`

	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}

	err := tx.QueryRow(ctx, query,
		n.ID, n.UserID, n.Type, n.Title, n.Message, n.Data,
		n.IsRead, n.IsSent, n.Channels, n.DeliveryStatus,
		n.ReferenceType, n.ReferenceID, n.IdempotencyKey,
		n.Priority, n.ExpiresAt, n.TemplateCode, n.TemplateVersion, n.TemplateData,
	).Scan(&n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create notification with tx: %w", err)
	}

	return nil
}

// GetByID retrieves notification by ID
func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	query := `
		SELECT 
			id, user_id, type, title, message, data,
			is_read, read_at, is_sent, sent_at, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data,
			created_at, updated_at
		FROM notifications
		WHERE id = $1
	`

	var n model.Notification
	err := r.db.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Data,
		&n.IsRead, &n.ReadAt, &n.IsSent, &n.SentAt, &n.Channels, &n.DeliveryStatus,
		&n.ReferenceType, &n.ReferenceID, &n.IdempotencyKey,
		&n.Priority, &n.ExpiresAt, &n.TemplateCode, &n.TemplateVersion, &n.TemplateData,
		&n.CreatedAt, &n.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("get notification by id: %w", err)
	}

	return &n, nil
}

// GetByIdempotencyKey retrieves notification by idempotency key
func (r *notificationRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.Notification, error) {
	query := `
		SELECT 
			id, user_id, type, title, message, data,
			is_read, read_at, is_sent, sent_at, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data,
			created_at, updated_at
		FROM notifications
		WHERE idempotency_key = $1
	`

	var n model.Notification
	err := r.db.QueryRow(ctx, query, key).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Data,
		&n.IsRead, &n.ReadAt, &n.IsSent, &n.SentAt, &n.Channels, &n.DeliveryStatus,
		&n.ReferenceType, &n.ReferenceID, &n.IdempotencyKey,
		&n.Priority, &n.ExpiresAt, &n.TemplateCode, &n.TemplateVersion, &n.TemplateData,
		&n.CreatedAt, &n.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("get notification by idempotency key: %w", err)
	}

	return &n, nil
}

// Update updates notification
func (r *notificationRepository) Update(ctx context.Context, n *model.Notification) error {
	query := `
		UPDATE notifications
		SET 
			title = $2,
			message = $3,
			data = $4,
			is_read = $5,
			read_at = $6,
			is_sent = $7,
			sent_at = $8,
			delivery_status = $9,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		n.ID, n.Title, n.Message, n.Data,
		n.IsRead, n.ReadAt, n.IsSent, n.SentAt, n.DeliveryStatus,
	)

	if err != nil {
		return fmt.Errorf("update notification: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrNotificationNotFound
	}

	return nil
}

// Delete soft deletes notification
func (r *notificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrNotificationNotFound
	}

	return nil
}

// List retrieves notifications with filters and pagination
func (r *notificationRepository) List(ctx context.Context, filters model.ListNotificationsRequest) ([]model.Notification, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE user_id = $1"
	args := []interface{}{filters.UserID}
	argCount := 1

	if filters.Type != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filters.Type)
	}

	if filters.IsRead != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND is_read = $%d", argCount)
		args = append(args, *filters.IsRead)
	}

	if filters.Channel != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND $%d = ANY(channels)", argCount)
		args = append(args, *filters.Channel)
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM notifications " + whereClause
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filters.SortBy != "" {
		orderBy = fmt.Sprintf("%s %s", filters.SortBy, filters.SortOrder)
	}

	// Calculate pagination
	limit := filters.PageSize
	offset := (filters.Page - 1) * filters.PageSize

	// Query notifications
	query := fmt.Sprintf(`
		SELECT 
			id, user_id, type, title, message, data,
			is_read, read_at, is_sent, sent_at, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data,
			created_at, updated_at
		FROM notifications
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argCount+1, argCount+2)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Data,
			&n.IsRead, &n.ReadAt, &n.IsSent, &n.SentAt, &n.Channels, &n.DeliveryStatus,
			&n.ReferenceType, &n.ReferenceID, &n.IdempotencyKey,
			&n.Priority, &n.ExpiresAt, &n.TemplateCode, &n.TemplateVersion, &n.TemplateData,
			&n.CreatedAt, &n.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, total, nil
}

// ListByUserID retrieves user's notifications
func (r *notificationRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, error) {
	query := `
		SELECT 
			id, user_id, type, title, message, data,
			is_read, read_at, is_sent, sent_at, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data,
			created_at, updated_at
		FROM notifications
		WHERE user_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY priority DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list by user id: %w", err)
	}
	defer rows.Close()

	return scanNotifications(rows)
}

// ListUnread retrieves unread notifications
func (r *notificationRepository) ListUnread(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, error) {
	query := `
		SELECT 
			id, user_id, type, title, message, data,
			is_read, read_at, is_sent, sent_at, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data,
			created_at, updated_at
		FROM notifications
		WHERE user_id = $1 AND is_read = FALSE
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY priority DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list unread: %w", err)
	}
	defer rows.Close()

	return scanNotifications(rows)
}

// MarkAsRead marks notifications as read
func (r *notificationRepository) MarkAsRead(ctx context.Context, notificationIDs []uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
		WHERE id = ANY($1) AND user_id = $2 AND is_read = FALSE
	`

	result, err := r.db.Exec(ctx, query, notificationIDs, userID)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrNotificationNotFound
	}

	return nil
}

// MarkAllAsRead marks all user's notifications as read
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("mark all as read: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// BulkCreate creates multiple notifications
func (r *notificationRepository) BulkCreate(ctx context.Context, notifications []model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Use batch insert for better performance
	batch := &pgx.Batch{}

	query := `
		INSERT INTO notifications (
			id, user_id, type, title, message, data,
			is_read, is_sent, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`

	for _, n := range notifications {
		if n.ID == uuid.Nil {
			n.ID = uuid.New()
		}
		batch.Queue(query,
			n.ID, n.UserID, n.Type, n.Title, n.Message, n.Data,
			n.IsRead, n.IsSent, n.Channels, n.DeliveryStatus,
			n.ReferenceType, n.ReferenceID, n.IdempotencyKey,
			n.Priority, n.ExpiresAt, n.TemplateCode, n.TemplateVersion, n.TemplateData,
		)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for range notifications {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("bulk create: %w", err)
		}
	}

	return nil
}

// UpdateDeliveryStatus updates delivery status for a channel
func (r *notificationRepository) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, channel string, status string) error {
	query := `
		UPDATE notifications
		SET 
			delivery_status = jsonb_set(
				COALESCE(delivery_status, '{}'::jsonb),
				$2::text[],
				$3::jsonb
			),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, fmt.Sprintf("{%s}", channel), fmt.Sprintf(`"%s"`, status))
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrNotificationNotFound
	}

	return nil
}

// MarkAsSent marks notification as sent
func (r *notificationRepository) MarkAsSent(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_sent = TRUE, sent_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark as sent: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrNotificationNotFound
	}

	return nil
}

// DeleteExpired deletes expired notifications
func (r *notificationRepository) DeleteExpired(ctx context.Context, before time.Time) (int, error) {
	query := `DELETE FROM notifications WHERE expires_at IS NOT NULL AND expires_at < $1`

	result, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("delete expired: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// DeleteOldRead deletes old read notifications
func (r *notificationRepository) DeleteOldRead(ctx context.Context, before time.Time) (int, error) {
	query := `DELETE FROM notifications WHERE is_read = TRUE AND read_at < $1`

	result, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("delete old read: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// GetUnreadCount gets unread notification count
func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT get_unread_notification_count($1)`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}

	return count, nil
}

// GetUnsentNotifications retrieves unsent notifications
func (r *notificationRepository) GetUnsentNotifications(ctx context.Context, limit int) ([]model.Notification, error) {
	query := `
		SELECT 
			id, user_id, type, title, message, data,
			is_read, read_at, is_sent, sent_at, channels, delivery_status,
			reference_type, reference_id, idempotency_key,
			priority, expires_at, template_code, template_version, template_data,
			created_at, updated_at
		FROM notifications
		WHERE is_sent = FALSE
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get unsent notifications: %w", err)
	}
	defer rows.Close()

	return scanNotifications(rows)
}

// CountByType counts notifications by type
func (r *notificationRepository) CountByType(ctx context.Context, notificationType string, from, to time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE type = $1
		AND created_at BETWEEN $2 AND $3
	`

	var count int
	err := r.db.QueryRow(ctx, query, notificationType, from, to).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by type: %w", err)
	}

	return count, nil
}

// Helper function to scan notifications
func scanNotifications(rows pgx.Rows) ([]model.Notification, error) {
	var notifications []model.Notification

	for rows.Next() {
		var n model.Notification
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Data,
			&n.IsRead, &n.ReadAt, &n.IsSent, &n.SentAt, &n.Channels, &n.DeliveryStatus,
			&n.ReferenceType, &n.ReferenceID, &n.IdempotencyKey,
			&n.Priority, &n.ExpiresAt, &n.TemplateCode, &n.TemplateVersion, &n.TemplateData,
			&n.CreatedAt, &n.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return notifications, nil
}
