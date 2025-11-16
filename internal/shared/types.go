package shared

import "time"

// SecurityAlertType defines types of security alerts
type SecurityAlertType string

const (
	AlertNewDeviceLogin     SecurityAlertType = "new_device_login"
	AlertPasswordChanged    SecurityAlertType = "password_changed"
	AlertEmailChanged       SecurityAlertType = "email_changed"
	AlertSuspiciousActivity SecurityAlertType = "suspicious_activity"
	AlertAccountLocked      SecurityAlertType = "account_locked"

	TypeCleanupExpiredToken   = "auth:cleanup_expired_tokens"
	TypeProcessFailedLogin    = "auth:process_failed_login"
	TypeSendSecurityAlert     = "auth:send_security_alert"
	TypeSendVerificationEmail = "email:verification"
	TypeSendResetEmail        = "email:reset_password"
	TypeProcessBookImage      = "book:process_image"
	TypeDeleteBookImages      = "book:delete_images"
)

// SecurityAlertPayload represents data for security alert
type SecurityAlertPayload struct {
	UserID     string            `json:"userId"`
	Email      string            `json:"email"`
	AlertType  SecurityAlertType `json:"alertType"`
	DeviceInfo map[string]string `json:"deviceInfo"`
	IPAddress  string            `json:"ipAddress"`
}

// FailedLoginPayload represents data for failed login tracking
type FailedLoginPayload struct {
	UserID    string    `json:"userId"`
	IPAddress string    `json:"ipAddress"`
	Timestamp time.Time `json:"timestamp"`
}

// User basic info (để tránh import cycle với user domain)
type UserBasicInfo struct {
	ID       string
	Email    string
	FullName string
}
