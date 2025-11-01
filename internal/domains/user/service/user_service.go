package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"bookstore-backend/internal/domains/user"
	// TODO: Cần tạo JWT helper
)

// userService implement user.Service interface
type userService struct {
	repo      user.Repository // Data access layer
	jwtSecret string          // JWT signing secret
}

// NewUserService tạo service instance
// Inject repository qua constructor (Dependency Injection)
func NewUserService(repo user.Repository, jwtSecret string) user.Service {
	return &userService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// ========================================
// AUTHENTICATION
// ========================================

// Register tạo user mới - FR-AUTH-001
func (s *userService) Register(ctx context.Context, req user.RegisterRequest) (*user.UserDTO, error) {
	// 1. VALIDATE INPUT
	// DTO validation đã được gọi ở handler layer, nhưng double-check an toàn hơn
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 2. BUSINESS RULE: Check email already exists
	exists, err := s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email exists: %w", err)
	}
	if exists {
		return nil, user.ErrEmailAlreadyExists
	}

	// 3. HASH PASSWORD
	// bcrypt cost = 12: balance giữa security và performance
	// Higher cost = slower but more secure
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 4. GENERATE VERIFICATION TOKEN
	// Random 32-byte hex string (64 chars)
	verificationToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 5. CREATE USER ENTITY
	now := time.Now()
	newUser := &user.User{
		ID:                 uuid.New(),
		Email:              req.Email,
		PasswordHash:       string(passwordHash),
		FullName:           req.FullName,
		Phone:              stringPtr(req.Phone), // Convert string to *string
		Role:               user.RoleUser,        // Default role
		IsActive:           true,                 // Active by default
		Points:             0,                    // Start with 0 points
		IsVerified:         false,                // Require email verification
		VerificationToken:  &verificationToken,
		VerificationSentAt: &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// 6. PERSIST TO DATABASE
	if err := s.repo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 7. SEND VERIFICATION EMAIL (Async)
	// TODO: Queue email job với Asynq
	// go s.emailService.SendVerificationEmail(newUser.Email, verificationToken)

	// 8. RETURN DTO (không expose sensitive data)
	dto := newUser.ToDTO()
	return &dto, nil
}

// Login xác thực user và trả về JWT tokens - FR-AUTH-002
func (s *userService) Login(ctx context.Context, req user.LoginRequest) (*user.LoginResponse, error) {
	// 1. VALIDATE INPUT
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 2. FIND USER BY EMAIL
	u, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Không expose "email not found" - security best practice
		// Attacker không biết email có tồn tại không
		return nil, user.ErrInvalidCredentials
	}

	// 3. CHECK USER STATUS
	// Business rule: user phải active và verified
	if !u.IsActive {
		return nil, user.ErrUserInactive
	}
	if !u.IsVerified {
		return nil, user.ErrUserNotVerified
	}

	// 4. VERIFY PASSWORD
	// bcrypt.CompareHashAndPassword is constant-time comparison (security)
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password))
	if err != nil {
		// Wrong password
		return nil, user.ErrInvalidCredentials
	}

	// 5. GENERATE JWT TOKENS
	// TODO: Implement JWT helper
	accessToken, err := s.generateAccessToken(u)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(u)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 6. UPDATE LAST LOGIN TIME (fire-and-forget)
	// Không quan trọng lắm, nếu fail thì bỏ qua
	go func() {
		_ = s.repo.UpdateLastLogin(context.Background(), u.ID)
	}()

	// 7. RETURN LOGIN RESPONSE
	dto := u.ToDTO()
	return &user.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute), // Access token TTL
		User:         dto,
	}, nil
}

// VerifyEmail xác nhận email - FR-AUTH-001
func (s *userService) VerifyEmail(ctx context.Context, token string) error {
	// 1. FIND USER BY TOKEN
	u, err := s.repo.FindByVerificationToken(ctx, token)
	if err != nil {
		return err // ErrInvalidToken or database error
	}

	// 2. CHECK ALREADY VERIFIED
	if u.IsVerified {
		return nil // Idempotent - return success
	}

	// 3. MARK AS VERIFIED
	if err := s.repo.MarkAsVerified(ctx, u.ID); err != nil {
		return fmt.Errorf("mark as verified: %w", err)
	}

	// 4. SEND WELCOME EMAIL (Async)
	// TODO: Queue welcome email
	// go s.emailService.SendWelcomeEmail(u.Email, u.FullName)

	return nil
}

// ForgotPassword gửi reset password link - FR-AUTH-003
func (s *userService) ForgotPassword(ctx context.Context, req user.ForgotPasswordRequest) error {
	// 1. VALIDATE INPUT
	if err := req.Validate(); err != nil {
		return err
	}

	// 2. FIND USER BY EMAIL
	u, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Security: không expose email có tồn tại không
		// Luôn return success để attacker không biết
		return nil
	}

	// 3. GENERATE RESET TOKEN
	resetToken, err := generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	// 4. SET TOKEN EXPIRY (15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)
	u.ResetToken = &resetToken
	u.ResetTokenExpiresAt = &expiresAt

	// 5. UPDATE USER
	if err := s.repo.Update(ctx, u); err != nil {
		return fmt.Errorf("update reset token: %w", err)
	}

	// 6. SEND RESET EMAIL (Async)
	// TODO: Queue password reset email
	// go s.emailService.SendPasswordResetEmail(u.Email, resetToken)

	return nil
}

// ResetPassword đổi password mới - FR-AUTH-003
func (s *userService) ResetPassword(ctx context.Context, req user.ResetPasswordRequest) error {
	// 1. VALIDATE INPUT
	if err := req.Validate(); err != nil {
		return err
	}

	// 2. FIND USER BY RESET TOKEN
	u, err := s.repo.FindByResetToken(ctx, req.Token)
	if err != nil {
		return err // ErrInvalidToken or database error
	}

	// 3. HASH NEW PASSWORD
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 4. UPDATE PASSWORD
	if err := s.repo.UpdatePassword(ctx, u.ID, string(passwordHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// 5. SEND CONFIRMATION EMAIL (Async)
	// TODO: Queue password changed email
	// go s.emailService.SendPasswordChangedEmail(u.Email)

	return nil
}

// ChangePassword user tự đổi password - FR-USER-003
func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, req user.ChangePasswordRequest) error {
	// 1. VALIDATE INPUT
	if err := req.Validate(); err != nil {
		return err
	}

	// 2. GET CURRENT USER
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// 3. VERIFY CURRENT PASSWORD
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword))
	if err != nil {
		return user.ErrInvalidCredentials
	}

	// 4. BUSINESS RULE: new password khác old password
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.NewPassword))
	if err == nil {
		// Same password
		return user.ErrSamePassword
	}

	// 5. HASH NEW PASSWORD
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 6. UPDATE PASSWORD
	if err := s.repo.UpdatePassword(ctx, userID, string(passwordHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

// ========================================
// USER PROFILE
// ========================================

// GetProfile lấy thông tin user hiện tại
func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*user.UserDTO, error) {
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	dto := u.ToDTO()
	return &dto, nil
}

// UpdateProfile cập nhật thông tin user
func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req user.UpdateProfileRequest) (*user.UserDTO, error) {
	// 1. VALIDATE INPUT
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 2. GET CURRENT USER
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 3. UPDATE FIELDS (chỉ update fields được gửi lên)
	if req.FullName != "" {
		u.FullName = req.FullName
	}
	if req.Phone != nil {
		u.Phone = req.Phone
	}

	// 4. PERSIST CHANGES
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	// 5. RETURN UPDATED DTO
	dto := u.ToDTO()
	return &dto, nil
}

// ========================================
// ADMIN FUNCTIONS
// ========================================

// ListUsers admin list users với filters - FR-ADM-003
func (s *userService) ListUsers(ctx context.Context, req user.ListUsersRequest) (*user.ListUsersResponse, error) {
	// 1. VALIDATE & SET DEFAULTS
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 2. GET USERS FROM REPOSITORY
	users, total, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	// 3. CONVERT TO DTOs
	userDTOs := make([]user.UserDTO, len(users))
	for i, u := range users {
		userDTOs[i] = u.ToDTO()
	}

	// 4. CALCULATE PAGINATION
	totalPages := (total + req.Limit - 1) / req.Limit // Ceiling division

	return &user.ListUsersResponse{
		Users: userDTOs,
		Pagination: user.PaginationMeta{
			CurrentPage: req.Page,
			PerPage:     req.Limit,
			Total:       total,
			TotalPages:  totalPages,
		},
	}, nil
}

// UpdateUserRole admin update user role
func (s *userService) UpdateUserRole(ctx context.Context, userID uuid.UUID, req user.UpdateRoleRequest) error {
	// 1. VALIDATE
	if err := req.Validate(); err != nil {
		return err
	}

	// 2. CHECK USER EXISTS
	_, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// 3. UPDATE ROLE
	if err := s.repo.UpdateRole(ctx, userID, req.Role); err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	return nil
}

// UpdateUserStatus admin activate/deactivate user
func (s *userService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, req user.UpdateStatusRequest) error {
	// 1. CHECK USER EXISTS
	_, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// 2. UPDATE STATUS
	if err := s.repo.UpdateStatus(ctx, userID, req.IsActive); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	return nil
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// generateSecureToken tạo random secure token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// stringPtr convert string thành *string (helper cho nullable fields)
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// generateAccessToken tạo JWT access token (15 minutes TTL)
func (s *userService) generateAccessToken(u *user.User) (string, error) {
	// TODO: Implement JWT generation
	// claims := jwt.MapClaims{
	//     "sub":   u.ID.String(),
	//     "email": u.Email,
	//     "role":  u.Role,
	//     "type":  "access",
	//     "exp":   time.Now().Add(15 * time.Minute).Unix(),
	//     "iat":   time.Now().Unix(),
	// }
	// return jwt.GenerateToken(claims, s.jwtSecret)
	return "TODO_ACCESS_TOKEN", nil
}

// generateRefreshToken tạo JWT refresh token (3 days TTL)
func (s *userService) generateRefreshToken(u *user.User) (string, error) {
	// TODO: Implement JWT generation
	// claims := jwt.MapClaims{
	//     "sub":  u.ID.String(),
	//     "type": "refresh",
	//     "exp":  time.Now().Add(3 * 24 * time.Hour).Unix(),
	//     "iat":  time.Now().Unix(),
	// }
	// return jwt.GenerateToken(claims, s.jwtSecret)
	return "TODO_REFRESH_TOKEN", nil
}
