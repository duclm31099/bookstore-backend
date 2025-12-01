package user

import (
	"context"

	"github.com/google/uuid"
)

// Service định nghĩa business logic layer contract
type Service interface {
	// Authentication
	Register(ctx context.Context, req RegisterRequest) (*UserDTO, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	VerifyEmail(ctx context.Context, token string) error
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	ResendVerification(ctx context.Context, email string) error
	UpdateVerificationToken(ctx context.Context, id string) (string, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
	// User Profile
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserDTO, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserDTO, error)

	// Admin Functions
	ListUsers(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, req UpdateRoleRequest) error
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, req UpdateStatusRequest) error
}
