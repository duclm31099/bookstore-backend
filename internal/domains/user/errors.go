package user

import "errors"

// Repository-level errors
var (
	// Not Found
	ErrUserNotFound = errors.New("user not found")

	// Conflict
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrPhoneAlreadyExists = errors.New("phone number already exists")

	// Invalid State
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInsufficientPoints = errors.New("insufficient loyalty points")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrUserDeleted        = errors.New("user account has been deleted")
)

// Service-level (Business logic) errors
var (
	// Authentication
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotVerified    = errors.New("email address not verified")
	ErrAccountLocked      = errors.New("account has been locked")

	// Password
	ErrPasswordTooWeak  = errors.New("password does not meet security requirements")
	ErrSamePassword     = errors.New("new password cannot be same as current password")
	ErrPasswordMismatch = errors.New("passwords do not match")

	// Sessions (FR-AUTH-006: max 5 sessions per user)
	ErrMaxSessionsReached = errors.New("maximum number of active sessions reached")

	// Rate Limiting
	ErrTooManyAttempts = errors.New("too many login attempts, please try again later")
	ErrTooManyRequests = errors.New("too many requests, please slow down")

	// Authorization
	ErrUnauthorized = errors.New("unauthorized access")
	ErrForbidden    = errors.New("forbidden: insufficient permissions")
	ErrInvalidRole  = errors.New("invalid user role")
)

// Validation errors
var (
	ErrInvalidEmail = errors.New("invalid email format")
	ErrInvalidPhone = errors.New("invalid phone number format")
	ErrInvalidUUID  = errors.New("invalid UUID format")
	ErrEmptyField   = errors.New("required field is empty")
	ErrFieldTooLong = errors.New("field exceeds maximum length")
)
