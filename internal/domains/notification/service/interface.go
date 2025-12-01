package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// NOTIFICATION SERVICE INTERFACE
// ================================================

type NotificationService interface {
	// Core notification operations
	SendNotification(ctx context.Context, req model.SendNotificationRequest) (*model.Notification, error)
	CreateNotification(ctx context.Context, req model.CreateNotificationRequest) (*model.Notification, error)

	// Retrieval operations
	GetNotificationByID(ctx context.Context, userID, notificationID uuid.UUID) (*model.NotificationResponse, error)
	ListNotifications(ctx context.Context, req model.ListNotificationsRequest) (*model.NotificationListResponse, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)

	// Status operations
	MarkAsRead(ctx context.Context, userID uuid.UUID, req model.MarkAsReadRequest) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error)
	DeleteNotification(ctx context.Context, userID, notificationID uuid.UUID) error

	// Background job operations
	ProcessUnsentNotifications(ctx context.Context, limit int) error
	CleanupExpiredNotifications(ctx context.Context) (int, error)
	CleanupOldReadNotifications(ctx context.Context, olderThan time.Duration) (int, error)
}

// ================================================
// PREFERENCES SERVICE INTERFACE
// ================================================

type PreferencesService interface {
	// User preference management
	GetUserPreferences(ctx context.Context, userID uuid.UUID) (*model.PreferencesResponse, error)
	UpdateUserPreferences(ctx context.Context, userID uuid.UUID, req model.UpdatePreferencesRequest) (*model.PreferencesResponse, error)

	// Preference checks (for internal use)
	CanSendNotification(ctx context.Context, userID uuid.UUID, notificationType, channel string) (bool, string, error)
	IsInQuietHours(ctx context.Context, userID uuid.UUID) (bool, error)
}

// ================================================
// TEMPLATE SERVICE INTERFACE (Admin)
// ================================================

type TemplateService interface {
	// Template CRUD
	CreateTemplate(ctx context.Context, adminID uuid.UUID, req model.CreateTemplateRequest) (*model.TemplateResponse, error)
	GetTemplateByID(ctx context.Context, templateID uuid.UUID) (*model.TemplateResponse, error)
	GetTemplateByCode(ctx context.Context, code string) (*model.TemplateResponse, error)
	UpdateTemplate(ctx context.Context, adminID, templateID uuid.UUID, req model.UpdateTemplateRequest) (*model.TemplateResponse, error)
	DeleteTemplate(ctx context.Context, templateID uuid.UUID) error
	ListTemplates(ctx context.Context, category *string, isActive *bool, page, pageSize int) ([]model.TemplateResponse, int64, error)

	// Template operations
	RenderTemplate(ctx context.Context, templateCode, channel string, data map[string]interface{}) (string, string, error)
	ValidateTemplateVariables(ctx context.Context, templateCode string, data map[string]interface{}) error
}

// ================================================
// CAMPAIGN SERVICE INTERFACE (Admin)
// ================================================

type CampaignService interface {
	// Campaign CRUD
	CreateCampaign(ctx context.Context, adminID uuid.UUID, req model.CreateCampaignRequest) (*model.CampaignResponse, error)
	GetCampaignByID(ctx context.Context, campaignID uuid.UUID) (*model.CampaignResponse, error)
	ListCampaigns(ctx context.Context, status *string, createdBy *uuid.UUID, page, pageSize int) ([]model.CampaignResponse, int64, error)

	// Campaign execution
	StartCampaign(ctx context.Context, campaignID uuid.UUID) error
	CancelCampaign(ctx context.Context, campaignID uuid.UUID) error
	ProcessScheduledCampaigns(ctx context.Context) error

	// Campaign batch processing (called by worker)
	ProcessCampaignBatch(ctx context.Context, campaignID uuid.UUID, batchNumber int) error
}

// ================================================
// DELIVERY SERVICE INTERFACE (Internal)
// ================================================

type DeliveryService interface {
	// Send via specific channel
	SendEmail(ctx context.Context, notification *model.Notification, recipient string) error
	SendSMS(ctx context.Context, notification *model.Notification, recipient string) error
	SendPush(ctx context.Context, notification *model.Notification, recipient string) error

	// Delivery tracking
	LogDeliveryAttempt(ctx context.Context, notificationID uuid.UUID, channel, recipient, status string) error
	UpdateDeliveryStatus(ctx context.Context, notificationID uuid.UUID, channel, status string) error

	// Retry failed deliveries
	RetryFailedDeliveries(ctx context.Context, limit int) error
}
