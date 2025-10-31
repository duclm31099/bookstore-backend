package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID `json:"id" db:"id"`
	Email    string    `json:"email" db:"email"`
	Password string    `json:"-" db:"password_hash"` // Never expose in JSON
	FullName string    `json:"full_name" db:"full_name"`
	Phone    *string   `json:"phone" db:"phone"`
	Role     string    `json:"role" db:"role"`
	IsActive bool      `json:"is_active" db:"is_active"`
	Points   int       `json:"points" db:"points"`

	// Email verification
	IsVerified         bool       `json:"is_verified" db:"is_verified"`
	VerificationToken  *string    `json:"-" db:"verification_token"`
	VerificationSentAt *time.Time `json:"-" db:"verification_sent_at"`

	// Password reset
	ResetToken          *string    `json:"-" db:"reset_token"`
	ResetTokenExpiresAt *time.Time `json:"-" db:"reset_token_expires_at"`

	LastLoginAt *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// DTOs (Data Transfer Objects)
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required"`
	Phone    string `json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}
