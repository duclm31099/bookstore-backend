package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// NOTIFICATION REPOSITORY INTERFACE
// ================================================

type NotificationRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, notification *model.Notification) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, notification *model.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.Notification, error)
	Update(ctx context.Context, notification *model.Notification) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List and filter operations
	List(ctx context.Context, filters model.ListNotificationsRequest) ([]model.Notification, int64, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, error)
	ListUnread(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, error)

	// Bulk operations
	MarkAsRead(ctx context.Context, notificationIDs []uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error)
	BulkCreate(ctx context.Context, notifications []model.Notification) error

	// Status operations
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, channel string, status string) error
	MarkAsSent(ctx context.Context, id uuid.UUID) error

	// Cleanup operations
	DeleteExpired(ctx context.Context, before time.Time) (int, error)
	DeleteOldRead(ctx context.Context, before time.Time) (int, error)

	// Statistics
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetUnsentNotifications(ctx context.Context, limit int) ([]model.Notification, error)
	CountByType(ctx context.Context, notificationType string, from, to time.Time) (int, error)
}

// ================================================
// PREFERENCES REPOSITORY INTERFACE
// ================================================

type PreferencesRepository interface {
	// Core operations
	Create(ctx context.Context, prefs *model.NotificationPreferences) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error)
	Update(ctx context.Context, prefs *model.NotificationPreferences) error
	Upsert(ctx context.Context, prefs *model.NotificationPreferences) error

	// Check operations
	IsChannelEnabled(ctx context.Context, userID uuid.UUID, notificationType, channel string) (bool, error)
	IsInQuietHours(ctx context.Context, userID uuid.UUID, checkTime time.Time) (bool, error)
	IsDoNotDisturb(ctx context.Context, userID uuid.UUID) (bool, error)
}

// ================================================
// TEMPLATE REPOSITORY INTERFACE
// ================================================

type TemplateRepository interface {
	// Core CRUD
	Create(ctx context.Context, template *model.NotificationTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationTemplate, error)
	GetByCode(ctx context.Context, code string) (*model.NotificationTemplate, error)
	Update(ctx context.Context, template *model.NotificationTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List operations
	List(ctx context.Context, category *string, isActive *bool, limit, offset int) ([]model.NotificationTemplate, int64, error)
	ListActive(ctx context.Context) ([]model.NotificationTemplate, error)

	// Version operations
	IncrementVersion(ctx context.Context, id uuid.UUID) error
	GetLatestVersion(ctx context.Context, code string) (*model.NotificationTemplate, error)
}

// ================================================
// DELIVERY LOG REPOSITORY INTERFACE
// ================================================

type DeliveryLogRepository interface {
	// Core operations
	Create(ctx context.Context, log *model.DeliveryLog) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, log *model.DeliveryLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.DeliveryLog, error)
	Update(ctx context.Context, log *model.DeliveryLog) error

	// List operations
	ListByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]model.DeliveryLog, error)
	ListFailedRetries(ctx context.Context, limit int) ([]model.DeliveryLog, error)

	// Status updates
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorCode, errorMessage *string) error
	MarkAsDelivered(ctx context.Context, id uuid.UUID, providerMessageID string) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) error

	// Statistics
	GetDeliveryRate(ctx context.Context, from, to time.Time, channel *string) (float64, error)
	CountByStatus(ctx context.Context, status string, from, to time.Time) (int, error)
}

// ================================================
// CAMPAIGN REPOSITORY INTERFACE
// ================================================

type CampaignRepository interface {
	// Core CRUD
	Create(ctx context.Context, campaign *model.Campaign) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Campaign, error)
	Update(ctx context.Context, campaign *model.Campaign) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List operations
	List(ctx context.Context, status *string, createdBy *uuid.UUID, limit, offset int) ([]model.Campaign, int64, error)
	ListScheduled(ctx context.Context, before time.Time) ([]model.Campaign, error)

	// Status operations
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateProgress(ctx context.Context, id uuid.UUID, sent, delivered, failed int) error

	// Execution operations
	MarkAsStarted(ctx context.Context, id uuid.UUID) error
	MarkAsCompleted(ctx context.Context, id uuid.UUID) error
	MarkAsCancelled(ctx context.Context, id uuid.UUID) error
}

// ================================================
// RATE LIMIT REPOSITORY INTERFACE
// ================================================

type RateLimitRepository interface {
	// Core operations
	Create(ctx context.Context, rateLimit *model.RateLimit) error
	GetByScope(ctx context.Context, scope, scopeID string, windowMinutes int) (*model.RateLimit, error)
	Update(ctx context.Context, rateLimit *model.RateLimit) error
	Upsert(ctx context.Context, rateLimit *model.RateLimit) error

	// Check and increment
	CheckLimit(ctx context.Context, scope, scopeID string, maxNotifications, windowMinutes int) (bool, error)
	IncrementCount(ctx context.Context, scope, scopeID string, windowMinutes int) error

	// Reset operations
	ResetExpiredWindows(ctx context.Context) (int, error)
	ResetByScope(ctx context.Context, scope, scopeID string) error
}
