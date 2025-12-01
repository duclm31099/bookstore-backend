package model

import (
	"errors"
	"fmt"
)

// ================================================
// DOMAIN-SPECIFIC ERRORS
// ================================================

// Notification errors
var (
	ErrNotificationNotFound    = errors.New("notification not found")
	ErrNotificationExpired     = errors.New("notification has expired")
	ErrInvalidNotificationType = errors.New("invalid notification type")
	ErrInvalidChannel          = errors.New("invalid notification channel")
	ErrInvalidJSONB            = errors.New("invalid JSONB data")
	ErrRateLimitExceeded       = errors.New("rate limit exceeded")
	ErrDuplicateNotification   = errors.New("duplicate notification detected")
)

// Preferences errors
var (
	ErrPreferencesNotFound = errors.New("notification preferences not found")
)

// Template errors
var (
	ErrTemplateNotFound    = errors.New("notification template not found")
	ErrTemplateInactive    = errors.New("notification template is not active")
	ErrTemplateCodeExists  = errors.New("template code already exists")
	ErrMissingVariables    = errors.New("missing required template variables")
	ErrInvalidTemplateCode = errors.New("invalid template code format")
)

// Campaign errors
var (
	ErrCampaignNotFound       = errors.New("notification campaign not found")
	ErrCampaignAlreadyStarted = errors.New("campaign has already been started")
	ErrCampaignCompleted      = errors.New("campaign has already been completed")
	ErrCampaignCancelled      = errors.New("campaign has been cancelled")
	ErrInvalidTargetType      = errors.New("invalid campaign target type")
)

// Delivery errors
var (
	ErrDeliveryFailed      = errors.New("notification delivery failed")
	ErrProviderUnavailable = errors.New("notification provider unavailable")
	ErrInvalidRecipient    = errors.New("invalid recipient")
	ErrMaxRetriesExceeded  = errors.New("maximum retry attempts exceeded")
)

// ================================================
// ERROR CODES (for API responses)
// ================================================

const (
	// Notification error codes
	ErrCodeNotificationNotFound    = "NOTIFICATION_NOT_FOUND"
	ErrCodeNotificationExpired     = "NOTIFICATION_EXPIRED"
	ErrCodeInvalidNotificationType = "INVALID_NOTIFICATION_TYPE"
	ErrCodeInvalidChannel          = "INVALID_CHANNEL"
	ErrCodeRateLimitExceeded       = "RATE_LIMIT_EXCEEDED"

	// Template error codes
	ErrCodeTemplateNotFound   = "TEMPLATE_NOT_FOUND"
	ErrCodeTemplateInactive   = "TEMPLATE_INACTIVE"
	ErrCodeTemplateCodeExists = "TEMPLATE_CODE_EXISTS"
	ErrCodeMissingVariables   = "MISSING_VARIABLES"

	// Campaign error codes
	ErrCodeCampaignNotFound  = "CAMPAIGN_NOT_FOUND"
	ErrCodeInvalidTargetType = "INVALID_TARGET_TYPE"

	// Delivery error codes
	ErrCodeDeliveryFailed      = "DELIVERY_FAILED"
	ErrCodeProviderUnavailable = "PROVIDER_UNAVAILABLE"
)

// ================================================
// CUSTOM ERROR TYPE (Optional - for richer errors)
// ================================================

type NotificationError struct {
	Code    string
	Message string
	Err     error
}

func (e *NotificationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *NotificationError) Unwrap() error {
	return e.Err
}

// NewNotificationError creates a new notification error
func NewNotificationError(code, message string, err error) *NotificationError {
	return &NotificationError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
