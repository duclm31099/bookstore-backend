package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"bookstore-backend/internal/domains/user"
	"bookstore-backend/internal/infrastructure/email"
	"bookstore-backend/internal/shared"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/jwt"
	"bookstore-backend/pkg/logger"
	// TODO: Cần tạo JWT helper
)

// userService implement user.Service interface
type userService struct {
	repo        user.Repository // Data access layer
	jwtManager  *jwt.Manager    // JWT signing secret
	asynqClient *asynq.Client
	cache       cache.Cache
}

// NewUserService tạo service instance
// Inject repository qua constructor (Dependency Injection)
func NewUserService(
	repo user.Repository,
	jwtManager *jwt.Manager,
	asynqClient *asynq.Client,
	cache cache.Cache) user.Service {
	return &userService{
		repo:        repo,
		jwtManager:  jwtManager,
		asynqClient: asynqClient, // Thêm dòng này!
		cache:       cache,
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
	passwordHash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 4. GENERATE VERIFICATION TOKEN
	// Random 32-byte hex string (64 chars)
	verificationToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	verificationTokenHash := s.hashToken(verificationToken)
	expiresAt := time.Now().Add(3 * 24 * time.Hour)
	// 5. CREATE USER ENTITY
	now := time.Now()
	newUser := &user.User{
		Email:                      req.Email,
		PasswordHash:               string(passwordHash),
		FullName:                   req.FullName,
		Phone:                      stringPtr(req.Phone), // Convert string to *string
		Role:                       user.RoleUser,        // Default role
		IsActive:                   true,                 // Active by default
		Points:                     0,                    // Start with 0 points
		IsVerified:                 false,                // Require email verification
		VerificationToken:          &verificationTokenHash,
		VerificationSentAt:         &now,
		VerificationTokenExpiresAt: &expiresAt,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	// 6. PERSIST TO DATABASE
	id, err := s.repo.Create(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	newUser.ID = id
	// 7. SEND VERIFICATION EMAIL (Async)
	// Gửi job qua Asynq
	link := fmt.Sprintf("http://localhost:8080/api/v1/auth/verify-email?token=%s", verificationTokenHash)
	payload := email.VerificationEmailData{
		VerifyLink: link,
		Email:      req.Email,
		ExpiresIn:  "24 giờ",
	}
	b, _ := json.Marshal(payload)
	task := asynq.NewTask(shared.TypeSendVerificationEmail, b)
	s.asynqClient.Enqueue(task, asynq.Queue("high"), asynq.Timeout(30*time.Second), asynq.MaxRetry(3))

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
		return nil, user.ErrInvalidCredentials
	}

	// ✅ 2.1. CHECK IF ACCOUNT IS LOCKED (Failed Login Protection)
	lockKey := fmt.Sprintf("account_locked:%s", u.ID)
	isLocked, _ := s.cache.Exists(ctx, lockKey)
	if isLocked {
		ttl, _ := s.cache.TTL(ctx, lockKey)
		remainingMinutes := int(ttl.Minutes())

		log.Warn().
			Str("user_id", u.ID.String()).
			Str("email", req.Email).
			Int("remaining_minutes", remainingMinutes).
			Msg("Login attempt on locked account")

		return nil, fmt.Errorf("tài khoản bị khóa tạm thời, vui lòng thử lại sau %d phút", remainingMinutes)
	}

	// 3. CHECK USER STATUS
	if !u.IsActive {
		return nil, user.ErrUserInactive
	}
	if !u.IsVerified {
		return nil, user.ErrUserNotVerified
	}

	// 4. VERIFY PASSWORD
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password))
	if err != nil {
		// ✅ 4.1. TRACK FAILED LOGIN ATTEMPT
		ipAddress := s.extractIPFromContext(ctx)
		s.trackFailedLogin(ctx, u.ID.String(), ipAddress)

		log.Warn().
			Str("user_id", u.ID.String()).
			Str("email", req.Email).
			Str("ip_address", ipAddress).
			Msg("Failed login attempt - wrong password")

		return nil, user.ErrInvalidCredentials
	}

	// ✅ 4.2. CLEAR FAILED LOGIN ATTEMPTS ON SUCCESS
	attemptKey := fmt.Sprintf("failed_login:%s", u.ID)
	if err := s.cache.Delete(ctx, attemptKey); err != nil {
		// Log but don't fail the login
		log.Warn().Err(err).Str("user_id", u.ID.String()).Msg("Failed to clear login attempts")
	}

	// 5. GENERATE JWT TOKENS
	accessToken, err := s.generateAccessToken(u)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(u)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 6. UPDATE LAST LOGIN TIME (fire-and-forget)
	// go func() {
	// 	_ = s.repo.UpdateLastLogin(context.Background(), u.ID)
	// }()

	// ✅ 6.1. LOG SUCCESSFUL LOGIN (for security monitoring)
	ipAddress := s.extractIPFromContext(ctx)
	log.Info().
		Str("user_id", u.ID.String()).
		Str("email", u.Email).
		Str("ip_address", ipAddress).
		Msg("Successful login")

	// 7. RETURN LOGIN RESPONSE
	dto := u.ToDTO()
	return &user.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		User:         dto,
	}, nil
}

// Logout handles user logout - clears session and logs the event
func (s *userService) Logout(ctx context.Context, userID uuid.UUID) error {
	// 1. LOG LOGOUT EVENT (for security monitoring)
	ipAddress := s.extractIPFromContext(ctx)
	log.Info().
		Str("user_id", userID.String()).
		Str("ip_address", ipAddress).
		Msg("User logged out")

	// 2. OPTIONAL: Clear any server-side session data
	// If you're using Redis for session management, clear it here
	// sessionKey := fmt.Sprintf("session:%s", userID)
	// _ = s.cache.Delete(ctx, sessionKey)

	// 3. OPTIONAL: Invalidate refresh token
	// If you're storing refresh tokens in database/cache, invalidate them here
	// refreshTokenKey := fmt.Sprintf("refresh_token:%s", userID)
	// _ = s.cache.Delete(ctx, refreshTokenKey)

	return nil
}

// trackFailedLogin enqueues a background job to track failed login attempts
func (s *userService) trackFailedLogin(ctx context.Context, userID, ipAddress string) {
	payload := shared.FailedLoginPayload{
		UserID:    userID,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal failed login payload")
		return
	}

	task := asynq.NewTask(shared.TypeProcessFailedLogin, data)

	// ✅ Enqueue with proper options
	_, err = s.asynqClient.EnqueueContext(
		ctx,
		task,
		asynq.Queue("default"),        // Default priority
		asynq.MaxRetry(1),             // Retry once if failed
		asynq.Timeout(10*time.Second), // 10s timeout
		asynq.ProcessIn(time.Second),  // Process immediately
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to enqueue failed login job")
	} else {
		log.Debug().
			Str("user_id", userID).
			Str("ip_address", ipAddress).
			Msg("Failed login job enqueued")
	}
}

// extractIPFromContext extracts IP address from request context
func (s *userService) extractIPFromContext(ctx context.Context) string {
	// ✅ Try to get IP from context (set by middleware)
	if ip, ok := ctx.Value("client_ip").(string); ok && ip != "" {
		return ip
	}

	// Try X-Forwarded-For header (set by middleware)
	if ip, ok := ctx.Value("x_forwarded_for").(string); ok && ip != "" {
		return ip
	}

	// Fallback to "unknown"
	return "unknown"
}

func (s *userService) RefreshToken(ctx context.Context, refreshTokenStr string) (*user.LoginResponse, error) {
	// 1. Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id in token: %w", err)
	}
	// 2. Get user
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, user.ErrUserNotFound
	}

	// 3. Check user still active
	if !u.IsActive {
		return nil, user.ErrUserInactive
	}
	// 4. Generate new tokens
	accessToken, err := s.generateAccessToken(u)

	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(u)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 5. Return (RefreshToken sẽ set vào cookie ở handler)
	dto := u.ToDTO()
	return &user.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
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
	logger.Info("Verify email done", map[string]interface{}{
		"MarkAsVerified": true,
	})
	return nil
}

// ForgotPassword gửi reset password link - FR-AUTH-003
func (s *userService) ForgotPassword(ctx context.Context, req user.ForgotPasswordRequest) error {

	// 1. FIND USER BY EMAIL
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

	// 4. SET TOKEN EXPIRY (24 hours)
	expiresAt := time.Now().Add(24 * 60 * time.Minute)
	u.ResetToken = &resetToken
	u.ResetTokenExpiresAt = &expiresAt

	// 5. UPDATE USER WITH RESET TOKEN
	if err := s.repo.UpdateResetToken(ctx, u.ID.String(), &resetToken, &expiresAt); err != nil {
		return fmt.Errorf("update reset token: %w", err)
	}

	// 6. SEND RESET EMAIL (Async)
	payload := user.ResetPasswordPayload{
		UserID:     u.ID.String(),
		Email:      req.Email,
		ResetToken: resetToken,
	}
	b, _ := json.Marshal(payload)
	task := asynq.NewTask(shared.TypeSendResetEmail, b)
	s.asynqClient.Enqueue(task, asynq.Queue("high"), asynq.Timeout(30*time.Second), asynq.MaxRetry(3))
	log.Printf("🔐 Reset token for %s: %s (expires: %v)", u.Email, resetToken, expiresAt)
	return nil
}

// service.go
func (s *userService) ResendVerification(ctx context.Context, email string) error {
	// 1. Find user by email
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// Security: generic message
		return nil
	}

	// 2. Check already verified
	if u.IsVerified {
		return nil
	}

	// 3. Generate new verification token
	token, err := generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	// 4. Update verification token
	newTokenHash := s.hashToken(token)
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.repo.UpdateVerificationToken(ctx, u.ID.String(), &newTokenHash, &expiresAt); err != nil {
		return fmt.Errorf("update token: %w", err)
	}

	// 5. Send email
	payload := user.VerifyEmailPayload{
		Token:  newTokenHash,
		UserID: u.ID.String(),
		Email:  u.Email,
	}
	b, _ := json.Marshal(payload)
	task := asynq.NewTask(shared.TypeSendVerificationEmail, b)
	s.asynqClient.Enqueue(task, asynq.Queue("high"), asynq.Timeout(30*time.Second), asynq.MaxRetry(3))

	return nil
}

// UpdateVerificationToken - Generate and update verification token
func (s *userService) UpdateVerificationToken(
	ctx context.Context,
	userID string,
) (string, error) {
	// 1. Generate new verification token (32-char hex)
	token, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}

	// 2. Get current time
	now := time.Now()

	// 3. Update database with new token
	// Token expires in 24 hours
	expiresAt := now.Add(24 * time.Hour)

	err = s.repo.UpdateVerificationToken(ctx, userID, &token, &expiresAt)
	if err != nil {
		return "", fmt.Errorf("update verification token in db: %w", err)
	}

	// 5. Log for audit trail
	logger.Info("verification token updated",
		map[string]interface{}{
			"user_id":    userID,
			"expires_at": expiresAt,
		})

	return token, nil
}

// ResetPassword đổi password mới - FR-AUTH-003
func (s *userService) ResetPassword(ctx context.Context, req user.ResetPasswordRequest) error {

	// 1. FIND USER BY RESET TOKEN
	u, err := s.repo.FindByResetToken(ctx, req.Token)
	if err != nil {
		return user.ErrInvalidToken
	}

	// 2. HASH NEW PASSWORD
	passwordHash, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// 3. UPDATE PASSWORD
	if err := s.repo.UpdatePassword(ctx, u.ID, string(passwordHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// 4. SEND CONFIRMATION EMAIL (Async)
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

// hashToken - Hash token using SHA256 for storage
func (s *userService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// stringPtr convert string thành *string (helper cho nullable fields)
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// generateAccessToken tạo JWT access token (24 hours TTL)
func (s *userService) generateAccessToken(u *user.User) (string, error) {
	return s.jwtManager.GenerateAccessToken(
		u.ID.String(),
		u.Email,
		u.Role.String(),
	)
}

// generateRefreshToken tạo JWT refresh token (3 days TTL)
func (s *userService) generateRefreshToken(u *user.User) (string, error) {
	return s.jwtManager.GenerateRefreshToken(u.ID.String())
}
func (s *userService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}
