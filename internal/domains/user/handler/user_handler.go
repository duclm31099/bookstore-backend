package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/user"
	"bookstore-backend/internal/shared/response"
)

// UserHandler xử lý HTTP requests cho user domain
// Struct này là stateless - chỉ chứa dependencies
type UserHandler struct {
	service user.Service // Business logic layer
}

// NewUserHandler tạo handler instance
// Constructor injection - nhận service qua parameter
func NewUserHandler(service user.Service) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

// ========================================
// AUTHENTICATION ENDPOINTS
// ========================================

// Register xử lý POST /auth/register - FR-AUTH-001
// @Summary      Register new user
// @Description  Create new user account with email verification
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body user.RegisterRequest true "Registration details"
// @Success      201 {object} response.Success{data=user.UserDTO}
// @Failure      400 {object} response.Error
// @Failure      409 {object} response.Error "Email already exists"
// @Router       /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	// STEP 1: PARSE REQUEST BODY
	// gin.Context.ShouldBindJSON: unmarshal JSON body vào struct
	// - Tự động validate JSON format
	// - Map JSON fields vào Go struct fields (theo json tags)
	var req user.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Bad request: JSON malformed hoặc missing required fields
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 2: VALIDATE BUSINESS RULES
	// DTO validation: email format, password strength, etc.
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 3: CALL SERVICE LAYER
	// Service xử lý business logic: hash password, check duplicates, save to DB
	// Context từ request: cho phép cancel operation khi client disconnect
	userDTO, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		// STEP 4: ERROR HANDLING
		// Map domain errors thành HTTP status codes
		h.handleError(c, err)
		return
	}

	// STEP 5: SUCCESS RESPONSE
	// 201 Created: resource mới được tạo thành công
	// Location header: URL của resource mới (optional)
	c.Header("Location", "/api/v1/users/"+userDTO.ID.String())
	response.Success(c, http.StatusCreated, "User registered successfully. Please check your email to verify.", userDTO)
}

// Login xử lý POST /auth/login - FR-AUTH-002
// @Summary      User login
// @Description  Authenticate user and return JWT tokens
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body user.LoginRequest true "Login credentials"
// @Success      200 {object} response.Success{data=user.LoginResponse}
// @Failure      400 {object} response.Error
// @Failure      401 {object} response.Error "Invalid credentials"
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	// STEP 1: PARSE REQUEST
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 2: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 3: AUTHENTICATE
	// Service verify password, generate JWT tokens
	loginResp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 4: SUCCESS
	// Return JWT tokens để client lưu (localStorage/cookie)
	response.Success(c, http.StatusOK, "Login successful", loginResp)
}

// VerifyEmail xử lý GET /auth/verify-email?token=xxx - FR-AUTH-001
// @Summary      Verify email address
// @Description  Confirm email verification token
// @Tags         Authentication
// @Param        token query string true "Verification token"
// @Success      200 {object} response.Success
// @Failure      400 {object} response.Error "Invalid or expired token"
// @Router       /auth/verify-email [get]
func (h *UserHandler) VerifyEmail(c *gin.Context) {
	// STEP 1: GET TOKEN FROM QUERY PARAMS
	// c.Query("token"): lấy ?token=xxx từ URL
	// Alternative: c.Param("token") nếu dùng /verify/:token
	token := c.Query("token")
	if token == "" {
		response.Error(c, http.StatusBadRequest, "Token is required", nil)
		return
	}

	// STEP 2: VERIFY TOKEN
	if err := h.service.VerifyEmail(c.Request.Context(), token); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 3: SUCCESS - có thể redirect về frontend
	// Option 1: JSON response (cho API)
	response.Success(c, http.StatusOK, "Email verified successfully", nil)

	// Option 2: HTML redirect (cho email link)
	// c.Redirect(http.StatusFound, "https://bookstore.com/verified")
}

// ForgotPassword xử lý POST /auth/forgot-password - FR-AUTH-003
// @Summary      Request password reset
// @Description  Send password reset link to email
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body user.ForgotPasswordRequest true "Email address"
// @Success      200 {object} response.Success
// @Router       /auth/forgot-password [post]
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	// STEP 1: PARSE REQUEST
	var req user.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 2: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 3: SEND RESET EMAIL
	// Service luôn return success (security: không expose email có tồn tại không)
	if err := h.service.ForgotPassword(c.Request.Context(), req); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 4: SUCCESS (generic message)
	// Không nói "email sent" vì có thể email không tồn tại
	response.Success(c, http.StatusOK, "If your email exists, you will receive a password reset link", nil)
}

// ResetPassword xử lý POST /auth/reset-password - FR-AUTH-003
// @Summary      Reset password
// @Description  Set new password using reset token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body user.ResetPasswordRequest true "Reset token and new password"
// @Success      200 {object} response.Success
// @Failure      400 {object} response.Error "Invalid or expired token"
// @Router       /auth/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	// STEP 1: PARSE REQUEST
	var req user.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 2: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 3: RESET PASSWORD
	if err := h.service.ResetPassword(c.Request.Context(), req); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 4: SUCCESS
	response.Success(c, http.StatusOK, "Password reset successfully. Please login with your new password.", nil)
}

// ========================================
// USER PROFILE ENDPOINTS (PROTECTED)
// ========================================
// Các endpoints này require authentication
// Middleware Auth() sẽ validate JWT và set user info vào context

// GetProfile xử lý GET /users/me
// @Summary      Get current user profile
// @Description  Get authenticated user's profile
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Success{data=user.UserDTO}
// @Failure      401 {object} response.Error "Unauthorized"
// @Router       /users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	// STEP 1: GET USER ID FROM CONTEXT
	// Middleware Auth() đã parse JWT và set userID vào context
	// c.Get("userID"): lấy value từ Gin context (type assertion cần thiết)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// STEP 2: GET PROFILE
	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 3: SUCCESS
	response.Success(c, http.StatusOK, "Profile retrieved successfully", profile)
}

// UpdateProfile xử lý PUT /users/me
// @Summary      Update user profile
// @Description  Update authenticated user's profile (name, phone)
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body user.UpdateProfileRequest true "Profile updates"
// @Success      200 {object} response.Success{data=user.UserDTO}
// @Failure      400 {object} response.Error
// @Failure      401 {object} response.Error
// @Router       /users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// STEP 1: GET USER ID
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// STEP 2: PARSE REQUEST
	var req user.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 3: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 4: UPDATE PROFILE
	updatedProfile, err := h.service.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 5: SUCCESS
	response.Success(c, http.StatusOK, "Profile updated successfully", updatedProfile)
}

// ChangePassword xử lý PUT /users/me/password
// @Summary      Change password
// @Description  Change authenticated user's password
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body user.ChangePasswordRequest true "Current and new password"
// @Success      200 {object} response.Success
// @Failure      400 {object} response.Error
// @Failure      401 {object} response.Error "Invalid current password"
// @Router       /users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	// STEP 1: GET USER ID
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// STEP 2: PARSE REQUEST
	var req user.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 3: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 4: CHANGE PASSWORD
	if err := h.service.ChangePassword(c.Request.Context(), userID, req); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 5: SUCCESS
	response.Success(c, http.StatusOK, "Password changed successfully", nil)
}

// ========================================
// ADMIN ENDPOINTS (PROTECTED + ROLE CHECK)
// ========================================
// Các endpoints này require role admin
// Middleware RequireRole(admin) sẽ check user role

// ListUsers xử lý GET /admin/users - FR-ADM-003
// @Summary      List all users (Admin)
// @Description  Get paginated list of users with filters
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        role query string false "Filter by role"
// @Param        is_verified query bool false "Filter by verification status"
// @Param        is_active query bool false "Filter by active status"
// @Param        search query string false "Search by email or name"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        sort_by query string false "Sort field" default(created_at)
// @Param        sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success      200 {object} response.Success{data=user.ListUsersResponse}
// @Failure      400 {object} response.Error
// @Failure      403 {object} response.Error "Forbidden"
// @Router       /admin/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// STEP 1: PARSE QUERY PARAMS
	// c.ShouldBindQuery: bind URL query params vào struct
	// Example: ?page=2&limit=10&role=user&search=john
	var req user.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err)
		return
	}

	// STEP 2: SET DEFAULTS & VALIDATE
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 3: GET USERS LIST
	result, err := h.service.ListUsers(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 4: SUCCESS
	response.Success(c, http.StatusOK, "Users retrieved successfully", result)
}

// UpdateUserRole xử lý PUT /admin/users/:id/role
// @Summary      Update user role (Admin)
// @Description  Change user's role (admin only)
// @Tags         Admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID (UUID)"
// @Param        request body user.UpdateRoleRequest true "New role"
// @Success      200 {object} response.Success
// @Failure      400 {object} response.Error
// @Failure      403 {object} response.Error
// @Failure      404 {object} response.Error
// @Router       /admin/users/{id}/role [put]
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	// STEP 1: GET USER ID FROM URL PATH
	// c.Param("id"): lấy :id từ route path /users/:id/role
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	// STEP 2: PARSE REQUEST BODY
	var req user.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 3: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 4: UPDATE ROLE
	if err := h.service.UpdateUserRole(c.Request.Context(), userID, req); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 5: SUCCESS
	response.Success(c, http.StatusOK, "User role updated successfully", nil)
}

// UpdateUserStatus xử lý PUT /admin/users/:id/status
// @Summary      Update user status (Admin)
// @Description  Activate or deactivate user account
// @Tags         Admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID (UUID)"
// @Param        request body user.UpdateStatusRequest true "Active status"
// @Success      200 {object} response.Success
// @Failure      400 {object} response.Error
// @Failure      403 {object} response.Error
// @Failure      404 {object} response.Error
// @Router       /admin/users/{id}/status [put]
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	// STEP 1: GET USER ID
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	// STEP 2: PARSE REQUEST
	var req user.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// STEP 3: UPDATE STATUS
	if err := h.service.UpdateUserStatus(c.Request.Context(), userID, req); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 4: SUCCESS
	response.Success(c, http.StatusOK, "User status updated successfully", nil)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// getUserIDFromContext lấy user ID từ Gin context
// Middleware Auth() đã set value này sau khi verify JWT
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	// c.Get("userID"): lấy value từ context
	// Returns: (interface{}, bool) - value và existence flag
	value, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, errors.New("user ID not found in context")
	}

	// TYPE ASSERTION: convert interface{} sang uuid.UUID
	// Có 2 cách:
	// 1. value.(uuid.UUID) - panic nếu type sai
	// 2. value.(uuid.UUID), ok - safe, return false nếu type sai
	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID type in context")
	}

	return userID, nil
}

// handleError map domain errors thành HTTP responses
// Centralized error handling - giảm duplicate code
func (h *UserHandler) handleError(c *gin.Context, err error) {
	// errors.Is: check error type trong error chain
	// Tốt hơn err == ErrXXX vì unwrap error chain

	switch {
	// 400 Bad Request - client error
	case errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrInvalidPhone),
		errors.Is(err, user.ErrPasswordTooWeak),
		errors.Is(err, user.ErrSamePassword):
		response.Error(c, http.StatusBadRequest, err.Error(), nil)

	// 401 Unauthorized - authentication failed
	case errors.Is(err, user.ErrInvalidCredentials),
		errors.Is(err, user.ErrUserNotVerified),
		errors.Is(err, user.ErrUserInactive):
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)

	// 403 Forbidden - authorization failed
	case errors.Is(err, user.ErrForbidden):
		response.Error(c, http.StatusForbidden, err.Error(), nil)

	// 404 Not Found
	case errors.Is(err, user.ErrUserNotFound):
		response.Error(c, http.StatusNotFound, err.Error(), nil)

	// 409 Conflict - resource already exists
	case errors.Is(err, user.ErrEmailAlreadyExists):
		response.Error(c, http.StatusConflict, err.Error(), nil)

	// 410 Gone - expired resource
	case errors.Is(err, user.ErrInvalidToken),
		errors.Is(err, user.ErrTokenExpired):
		response.Error(c, http.StatusGone, err.Error(), nil)

	// 429 Too Many Requests - rate limiting
	case errors.Is(err, user.ErrTooManyAttempts):
		response.Error(c, http.StatusTooManyRequests, err.Error(), nil)

	// 500 Internal Server Error - unexpected errors
	default:
		// Log error nhưng không expose details cho client (security)
		// TODO: Log với logger (zap, logrus)
		// logger.Error("internal error", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Internal server error", nil)
	}
}
