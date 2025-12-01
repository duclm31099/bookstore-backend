package model

import (
	"time"

	"github.com/google/uuid"
)

// ================================================
// NOTIFICATION DTOs
// ================================================

// CreateNotificationRequest - Request to create a notification
type CreateNotificationRequest struct {
	UserID        uuid.UUID              `json:"user_id" validate:"required"`
	Type          string                 `json:"type" validate:"required,oneof=promotion_removed order_status payment new_promotion review_response system_alert"`
	Title         string                 `json:"title" validate:"required,max=255"`
	Message       string                 `json:"message" validate:"required"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Channels      []string               `json:"channels" validate:"required,dive,oneof=in_app email push sms"`
	ReferenceType *string                `json:"reference_type,omitempty"`
	ReferenceID   *uuid.UUID             `json:"reference_id,omitempty"`
	Priority      *int                   `json:"priority,omitempty" validate:"omitempty,min=1,max=3"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	TemplateCode  *string                `json:"template_code,omitempty"`
	TemplateData  map[string]interface{} `json:"template_data,omitempty"`
}

// SendNotificationRequest - Request to send notification using template
type SendNotificationRequest struct {
	UserID        uuid.UUID              `json:"user_id" validate:"required"`
	TemplateCode  string                 `json:"template_code" validate:"required"`
	Channels      []string               `json:"channels,omitempty"`
	Data          map[string]interface{} `json:"data" validate:"required"`
	ReferenceType *string                `json:"reference_type,omitempty"`
	ReferenceID   *uuid.UUID             `json:"reference_id,omitempty"`
	Priority      *int                   `json:"priority,omitempty"`
}

// ListNotificationsRequest - Query filters for listing notifications
type ListNotificationsRequest struct {
	UserID    uuid.UUID `query:"user_id" validate:"required"`
	Type      *string   `query:"type,omitempty"`
	IsRead    *bool     `query:"is_read,omitempty"`
	Channel   *string   `query:"channel,omitempty"`
	Page      int       `query:"page" validate:"min=1"`
	PageSize  int       `query:"page_size" validate:"min=1,max=100"`
	SortBy    string    `query:"sort_by" validate:"oneof=created_at read_at priority"`
	SortOrder string    `query:"sort_order" validate:"oneof=asc desc"`
}

// NotificationResponse - Response DTO
type NotificationResponse struct {
	ID             uuid.UUID              `json:"id"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Data           map[string]interface{} `json:"data,omitempty"`
	IsRead         bool                   `json:"is_read"`
	ReadAt         *time.Time             `json:"read_at,omitempty"`
	IsSent         bool                   `json:"is_sent"`
	SentAt         *time.Time             `json:"sent_at,omitempty"`
	Channels       []string               `json:"channels"`
	DeliveryStatus map[string]interface{} `json:"delivery_status,omitempty"`
	ReferenceType  *string                `json:"reference_type,omitempty"`
	ReferenceID    *uuid.UUID             `json:"reference_id,omitempty"`
	Priority       int                    `json:"priority"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// NotificationListResponse - Paginated list
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Pagination    PaginationMeta         `json:"pagination"`
	UnreadCount   int                    `json:"unread_count"`
}

// MarkAsReadRequest - Mark notification(s) as read
type MarkAsReadRequest struct {
	NotificationIDs []uuid.UUID `json:"notification_ids" validate:"required,min=1"`
}

// ================================================
// PREFERENCES DTOs
// ================================================

// UpdatePreferencesRequest - Update user notification preferences
type UpdatePreferencesRequest struct {
	Preferences     map[string]PreferenceChannels `json:"preferences" validate:"required"`
	DoNotDisturb    *bool                         `json:"do_not_disturb,omitempty"`
	QuietHoursStart *string                       `json:"quiet_hours_start,omitempty" validate:"omitempty,time_format=15:04"`
	QuietHoursEnd   *string                       `json:"quiet_hours_end,omitempty" validate:"omitempty,time_format=15:04"`
}

// PreferencesResponse - User preferences response
type PreferencesResponse struct {
	UserID          uuid.UUID                     `json:"user_id"`
	Preferences     map[string]PreferenceChannels `json:"preferences"`
	DoNotDisturb    bool                          `json:"do_not_disturb"`
	QuietHoursStart *string                       `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd   *string                       `json:"quiet_hours_end,omitempty"`
	UpdatedAt       time.Time                     `json:"updated_at"`
}

// ================================================
// TEMPLATE DTOs (Admin)
// ================================================

// CreateTemplateRequest - Create notification template
type CreateTemplateRequest struct {
	Code              string   `json:"code" validate:"required,max=100"`
	Name              string   `json:"name" validate:"required,max=255"`
	Description       *string  `json:"description,omitempty"`
	Category          string   `json:"category" validate:"required,oneof=transactional marketing system"`
	EmailSubject      *string  `json:"email_subject,omitempty"`
	EmailBodyHTML     *string  `json:"email_body_html,omitempty"`
	EmailBodyText     *string  `json:"email_body_text,omitempty"`
	SMSBody           *string  `json:"sms_body,omitempty"`
	PushTitle         *string  `json:"push_title,omitempty"`
	PushBody          *string  `json:"push_body,omitempty"`
	InAppTitle        *string  `json:"in_app_title,omitempty"`
	InAppBody         *string  `json:"in_app_body,omitempty"`
	InAppActionURL    *string  `json:"in_app_action_url,omitempty"`
	RequiredVariables []string `json:"required_variables,omitempty"`
	Language          string   `json:"language" validate:"required,max=5"`
	DefaultChannels   []string `json:"default_channels" validate:"required,dive,oneof=in_app email push sms"`
	DefaultPriority   int      `json:"default_priority" validate:"required,min=1,max=3"`
	ExpiresAfterHours *int     `json:"expires_after_hours,omitempty"`
}

// UpdateTemplateRequest - Update template
type UpdateTemplateRequest struct {
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	EmailSubject      *string  `json:"email_subject,omitempty"`
	EmailBodyHTML     *string  `json:"email_body_html,omitempty"`
	EmailBodyText     *string  `json:"email_body_text,omitempty"`
	SMSBody           *string  `json:"sms_body,omitempty"`
	PushTitle         *string  `json:"push_title,omitempty"`
	PushBody          *string  `json:"push_body,omitempty"`
	InAppTitle        *string  `json:"in_app_title,omitempty"`
	InAppBody         *string  `json:"in_app_body,omitempty"`
	InAppActionURL    *string  `json:"in_app_action_url,omitempty"`
	RequiredVariables []string `json:"required_variables,omitempty"`
	DefaultChannels   []string `json:"default_channels,omitempty"`
	DefaultPriority   *int     `json:"default_priority,omitempty"`
	ExpiresAfterHours *int     `json:"expires_after_hours,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
}

// TemplateResponse - Template response
type TemplateResponse struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	Category          string     `json:"category"`
	RequiredVariables []string   `json:"required_variables"`
	Language          string     `json:"language"`
	DefaultChannels   []string   `json:"default_channels"`
	DefaultPriority   int        `json:"default_priority"`
	ExpiresAfterHours *int       `json:"expires_after_hours,omitempty"`
	Version           int        `json:"version"`
	IsActive          bool       `json:"is_active"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy         *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ================================================
// CAMPAIGN DTOs (Admin)
// ================================================

// CreateCampaignRequest - Create notification campaign
type CreateCampaignRequest struct {
	Name              string                 `json:"name" validate:"required,max=255"`
	Description       *string                `json:"description,omitempty"`
	TemplateCode      string                 `json:"template_code" validate:"required"`
	TargetType        string                 `json:"target_type" validate:"required,oneof=all_users segment specific_users"`
	TargetSegment     *string                `json:"target_segment,omitempty"`
	TargetUserIDs     []uuid.UUID            `json:"target_user_ids,omitempty"`
	TargetFilters     map[string]interface{} `json:"target_filters,omitempty"`
	ScheduledAt       *time.Time             `json:"scheduled_at,omitempty"`
	BatchSize         *int                   `json:"batch_size,omitempty" validate:"omitempty,min=100,max=5000"`
	BatchDelaySeconds *int                   `json:"batch_delay_seconds,omitempty" validate:"omitempty,min=1,max=60"`
	TemplateData      map[string]interface{} `json:"template_data" validate:"required"`
	Channels          []string               `json:"channels" validate:"required,dive,oneof=in_app email push sms"`
}

// CampaignResponse - Campaign response
type CampaignResponse struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	TemplateCode    *string    `json:"template_code,omitempty"`
	Status          string     `json:"status"`
	TotalRecipients *int       `json:"total_recipients,omitempty"`
	ProcessedCount  int        `json:"processed_count"`
	SentCount       int        `json:"sent_count"`
	DeliveredCount  int        `json:"delivered_count"`
	FailedCount     int        `json:"failed_count"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ================================================
// SHARED DTOs
// ================================================

// PaginationMeta - Standard pagination metadata
type PaginationMeta struct {
	CurrentPage  int   `json:"current_page"`
	PageSize     int   `json:"page_size"`
	TotalPages   int   `json:"total_pages"`
	TotalRecords int64 `json:"total_records"`
}

// UnreadCountResponse - Unread notification count
type UnreadCountResponse struct {
	Count int `json:"count"`
}
