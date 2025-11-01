package user

import (
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
)

// ========================================
// AUTH DTOs
// ========================================

// RegisterRequest - FR-AUTH-001: User Registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone,omitempty"`
}

func (r RegisterRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email,
			validation.Required.Error("email is required"),
			is.Email.Error("invalid email format"),
			validation.Length(5, 255),
		),
		validation.Field(&r.Password,
			validation.Required.Error("password is required"),
			validation.Length(8, 128).Error("password must be 8-128 characters"),
			validation.Match(regexp.MustCompile(`[A-Z]`)).Error("password must contain at least one uppercase letter"),
			validation.Match(regexp.MustCompile(`[a-z]`)).Error("password must contain at least one lowercase letter"),
			validation.Match(regexp.MustCompile(`[0-9]`)).Error("password must contain at least one number"),
		),
		validation.Field(&r.FullName,
			validation.Required.Error("full name is required"),
			validation.Length(2, 100),
		),
		validation.Field(&r.Phone,
			validation.When(r.Phone != "",
				is.E164.Error("phone must be in E.164 format (e.g., +84912345678)"),
			),
		),
	)
}

// LoginRequest - FR-AUTH-002: User Login
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (r LoginRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required, is.Email),
		validation.Field(&r.Password, validation.Required),
	)
}

// LoginResponse - JWT tokens
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserDTO   `json:"user"`
}

// RefreshTokenRequest - FR-AUTH-004: Token Refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// VerifyEmailRequest - FR-AUTH-001: Email Verification
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// ForgotPasswordRequest - FR-AUTH-003: Password Reset Request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

func (r ForgotPasswordRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required, is.Email),
	)
}

// ResetPasswordRequest - FR-AUTH-003: Password Reset
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (r ResetPasswordRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Token, validation.Required),
		validation.Field(&r.NewPassword,
			validation.Required,
			validation.Length(8, 128),
			validation.Match(regexp.MustCompile(`[A-Z]`)).Error("must contain uppercase letter"),
			validation.Match(regexp.MustCompile(`[a-z]`)).Error("must contain lowercase letter"),
			validation.Match(regexp.MustCompile(`[0-9]`)).Error("must contain number"),
		),
	)
}

// ChangePasswordRequest - User changes own password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (r ChangePasswordRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.CurrentPassword, validation.Required),
		validation.Field(&r.NewPassword,
			validation.Required,
			validation.Length(8, 128),
			validation.Match(regexp.MustCompile(`[A-Z]`)).Error("must contain uppercase"),
			validation.Match(regexp.MustCompile(`[a-z]`)).Error("must contain lowercase"),
			validation.Match(regexp.MustCompile(`[0-9]`)).Error("must contain number"),
		),
	)
}

// ========================================
// USER PROFILE DTOs
// ========================================

// UserDTO - Public user representation (safe to expose)
type UserDTO struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Phone       *string    `json:"phone,omitempty"`
	Role        Role       `json:"role"`
	IsActive    bool       `json:"is_active"`
	Points      int        `json:"points"`
	IsVerified  bool       `json:"is_verified"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ToDTO converts User entity to UserDTO
func (u *User) ToDTO() UserDTO {
	return UserDTO{
		ID:          u.ID,
		Email:       u.Email,
		FullName:    u.FullName,
		Phone:       u.Phone,
		Role:        u.Role,
		IsActive:    u.IsActive,
		Points:      u.Points,
		IsVerified:  u.IsVerified,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

// UpdateProfileRequest - User updates own profile
type UpdateProfileRequest struct {
	FullName string  `json:"full_name,omitempty"`
	Phone    *string `json:"phone,omitempty"`
}

func (r UpdateProfileRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.FullName,
			validation.When(r.FullName != "", validation.Length(2, 100)),
		),
		validation.Field(&r.Phone,
			validation.When(r.Phone != nil && *r.Phone != "",
				is.E164.Error("phone must be in E.164 format"),
			),
		),
	)
}

// ========================================
// ADMIN DTOs
// ========================================

// ListUsersRequest - FR-ADM-003: List users with filters
type ListUsersRequest struct {
	Role       *Role  `form:"role"`
	IsVerified *bool  `form:"is_verified"`
	IsActive   *bool  `form:"is_active"`
	Search     string `form:"search"` // Search by email or name
	Page       int    `form:"page" binding:"min=1"`
	Limit      int    `form:"limit" binding:"min=1,max=100"`
	SortBy     string `form:"sort_by"`    // "created_at", "points", "last_login_at"
	SortOrder  string `form:"sort_order"` // "asc", "desc"
}

// SetDefaults sets default values for pagination
func (r *ListUsersRequest) SetDefaults() {
	if r.Page == 0 {
		r.Page = 1
	}
	if r.Limit == 0 {
		r.Limit = 20
	}
	if r.SortBy == "" {
		r.SortBy = "created_at"
	}
	if r.SortOrder == "" {
		r.SortOrder = "desc"
	}
}

func (r ListUsersRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.When(r.Role != nil, validation.In(RoleUser, RoleAdmin, RoleWarehouse, RoleCSKH)),
		),
		validation.Field(&r.SortBy,
			validation.In("created_at", "points", "last_login_at", "email"),
		),
		validation.Field(&r.SortOrder,
			validation.In("asc", "desc"),
		),
	)
}

// ListUsersResponse - Paginated user list
type ListUsersResponse struct {
	Users      []UserDTO      `json:"users"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta - Pagination metadata
type PaginationMeta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	TotalPages  int `json:"total_pages"`
}

// UpdateRoleRequest - Admin updates user role
type UpdateRoleRequest struct {
	Role Role `json:"role" binding:"required"`
}

func (r UpdateRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.Required,
			validation.In(RoleUser, RoleAdmin, RoleWarehouse, RoleCSKH),
		),
	)
}

// UpdateStatusRequest - Admin activates/deactivates user
type UpdateStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// ========================================
// LOYALTY POINTS DTOs
// ========================================

// AddPointsRequest - Admin adds loyalty points
type AddPointsRequest struct {
	Points int    `json:"points" binding:"required,min=1"`
	Reason string `json:"reason,omitempty"`
}

// DeductPointsRequest - System deducts points (order redemption)
type DeductPointsRequest struct {
	Points int    `json:"points" binding:"required,min=1"`
	Reason string `json:"reason,omitempty"`
}
