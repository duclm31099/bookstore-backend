package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ================================================
// NOTIFICATION ENTITY
// ================================================

type Notification struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	Type            string     `json:"type"`
	Title           string     `json:"title"`
	Message         string     `json:"message"`
	Data            JSONB      `json:"data,omitempty"`
	IsRead          bool       `json:"is_read"`
	ReadAt          *time.Time `json:"read_at,omitempty"`
	IsSent          bool       `json:"is_sent"`
	SentAt          *time.Time `json:"sent_at,omitempty"`
	Channels        []string   `json:"channels"`
	DeliveryStatus  JSONB      `json:"delivery_status,omitempty"`
	ReferenceType   *string    `json:"reference_type,omitempty"`
	ReferenceID     *uuid.UUID `json:"reference_id,omitempty"`
	IdempotencyKey  *string    `json:"idempotency_key,omitempty"`
	Priority        int        `json:"priority"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	TemplateCode    *string    `json:"template_code,omitempty"`
	TemplateVersion *int       `json:"template_version,omitempty"`
	TemplateData    JSONB      `json:"template_data,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Notification types constants
const (
	NotificationTypePromotionRemoved = "promotion_removed"
	NotificationTypeOrderStatus      = "order_status"
	NotificationTypePayment          = "payment"
	NotificationTypeNewPromotion     = "new_promotion"
	NotificationTypeReviewResponse   = "review_response"
	NotificationTypeSystemAlert      = "system_alert"
)

// Notification channels
const (
	ChannelInApp = "in_app"
	ChannelEmail = "email"
	ChannelPush  = "push"
	ChannelSMS   = "sms"
)

// Priority levels
const (
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
)

// ================================================
// NOTIFICATION PREFERENCES
// ================================================

type NotificationPreferences struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	Preferences     JSONB      `json:"preferences"`
	DoNotDisturb    bool       `json:"do_not_disturb"`
	QuietHoursStart *time.Time `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd   *time.Time `json:"quiet_hours_end,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Default preference structure
type PreferenceChannels struct {
	InApp bool `json:"in_app"`
	Email bool `json:"email"`
	Push  bool `json:"push"`
}

type PreferencesMap map[string]PreferenceChannels

// ================================================
// NOTIFICATION TEMPLATE
// ================================================

type NotificationTemplate struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	Category          string     `json:"category"`
	EmailSubject      *string    `json:"email_subject,omitempty"`
	EmailBodyHTML     *string    `json:"email_body_html,omitempty"`
	EmailBodyText     *string    `json:"email_body_text,omitempty"`
	SMSBody           *string    `json:"sms_body,omitempty"`
	PushTitle         *string    `json:"push_title,omitempty"`
	PushBody          *string    `json:"push_body,omitempty"`
	InAppTitle        *string    `json:"in_app_title,omitempty"`
	InAppBody         *string    `json:"in_app_body,omitempty"`
	InAppActionURL    *string    `json:"in_app_action_url,omitempty"`
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

// Template categories
const (
	CategoryTransactional = "transactional"
	CategoryMarketing     = "marketing"
	CategorySystem        = "system"
)

// ================================================
// DELIVERY LOG
// ================================================

type DeliveryLog struct {
	ID                uuid.UUID  `json:"id"`
	NotificationID    uuid.UUID  `json:"notification_id"`
	Channel           string     `json:"channel"`
	AttemptNumber     int        `json:"attempt_number"`
	Status            string     `json:"status"`
	Recipient         string     `json:"recipient"`
	Provider          *string    `json:"provider,omitempty"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	ProviderResponse  JSONB      `json:"provider_response,omitempty"`
	ErrorCode         *string    `json:"error_code,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	QueuedAt          *time.Time `json:"queued_at,omitempty"`
	ProcessingAt      *time.Time `json:"processing_at,omitempty"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	OpenedAt          *time.Time `json:"opened_at,omitempty"`
	ClickedAt         *time.Time `json:"clicked_at,omitempty"`
	FailedAt          *time.Time `json:"failed_at,omitempty"`
	RetryAfter        *time.Time `json:"retry_after,omitempty"`
	MaxRetries        int        `json:"max_retries"`
	EstimatedCost     *float64   `json:"estimated_cost,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Delivery statuses
const (
	DeliveryStatusQueued     = "queued"
	DeliveryStatusProcessing = "processing"
	DeliveryStatusSent       = "sent"
	DeliveryStatusDelivered  = "delivered"
	DeliveryStatusFailed     = "failed"
	DeliveryStatusBounced    = "bounced"
	DeliveryStatusOpened     = "opened"
	DeliveryStatusClicked    = "clicked"
)

// ================================================
// CAMPAIGN
// ================================================

type Campaign struct {
	ID                uuid.UUID   `json:"id"`
	Name              string      `json:"name"`
	Description       *string     `json:"description,omitempty"`
	TemplateCode      *string     `json:"template_code,omitempty"`
	TargetType        string      `json:"target_type"`
	TargetSegment     *string     `json:"target_segment,omitempty"`
	TargetUserIDs     []uuid.UUID `json:"target_user_ids,omitempty"`
	TargetFilters     JSONB       `json:"target_filters,omitempty"`
	ScheduledAt       *time.Time  `json:"scheduled_at,omitempty"`
	StartedAt         *time.Time  `json:"started_at,omitempty"`
	CompletedAt       *time.Time  `json:"completed_at,omitempty"`
	CancelledAt       *time.Time  `json:"cancelled_at,omitempty"`
	Status            string      `json:"status"`
	BatchSize         int         `json:"batch_size"`
	BatchDelaySeconds int         `json:"batch_delay_seconds"`
	TotalRecipients   *int        `json:"total_recipients,omitempty"`
	ProcessedCount    int         `json:"processed_count"`
	SentCount         int         `json:"sent_count"`
	DeliveredCount    int         `json:"delivered_count"`
	FailedCount       int         `json:"failed_count"`
	TemplateData      JSONB       `json:"template_data,omitempty"`
	Channels          []string    `json:"channels"`
	CreatedBy         *uuid.UUID  `json:"created_by,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// Campaign statuses
const (
	CampaignStatusDraft     = "draft"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusRunning   = "running"
	CampaignStatusPaused    = "paused"
	CampaignStatusCompleted = "completed"
	CampaignStatusCancelled = "cancelled"
)

// Campaign target types
const (
	TargetTypeAllUsers      = "all_users"
	TargetTypeSegment       = "segment"
	TargetTypeSpecificUsers = "specific_users"
)

// ================================================
// RATE LIMIT
// ================================================

type RateLimit struct {
	ID               uuid.UUID `json:"id"`
	Scope            string    `json:"scope"`
	ScopeID          *string   `json:"scope_id,omitempty"`
	MaxNotifications int       `json:"max_notifications"`
	WindowMinutes    int       `json:"window_minutes"`
	CurrentCount     int       `json:"current_count"`
	WindowStart      time.Time `json:"window_start"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Rate limit scopes
const (
	RateLimitScopeGlobal           = "global"
	RateLimitScopeUser             = "user"
	RateLimitScopeNotificationType = "notification_type"
)

// ================================================
// JSONB TYPE (PostgreSQL JSONB support)
// ================================================

type JSONB map[string]interface{}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return ErrInvalidJSONB
	}

	result := make(JSONB)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// MarshalJSON implements json.Marshaler
func (j JSONB) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	// Marshal underlying map để tránh recursion
	return json.Marshal(map[string]interface{}(j))
}

// UnmarshalJSON implements json.Unmarshaler
func (j *JSONB) UnmarshalJSON(data []byte) error {
	// Unmarshal vào map thay vì JSONB để tránh recursion
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*j = JSONB(m)
	return nil
}
