package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/cart/service"
	"bookstore-backend/internal/domains/user"
	"bookstore-backend/internal/shared/middleware"
	"bookstore-backend/internal/shared/response"
	"bookstore-backend/pkg/jwt"
	"bookstore-backend/pkg/logger"
)

// UserHandler xử lý HTTP requests cho user domain
// Struct này là stateless - chỉ chứa dependencies
type UserHandler struct {
	service     user.Service // Business logic layer
	cartService service.ServiceInterface
	jwtManager  *jwt.Manager
}

// NewUserHandler tạo handler instance
// Constructor injection - nhận service qua parameter
func NewUserHandler(
	service user.Service,
	cartService service.ServiceInterface,
	jwtManager *jwt.Manager,

) *UserHandler {
	return &UserHandler{
		service:     service,
		cartService: cartService,
		jwtManager:  jwtManager,
	}
}

// ========================================
// AUTHENTICATION ENDPOINTS
// ========================================

// Register xử lý POST /auth/register - FR-AUTH-001
// @Summary      Register new user
// @Description  Create new user account with email verification
// @Tags         Authentication
// @Router       /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	// STEP 1: PARSE REQUEST BODY
	// gin.Context.ShouldBindJSON: unmarshal JSON body vào struct
	// - Tự động validate JSON format
	// - Map JSON fields vào Go struct fields (theo json tags)
	var req user.RegisterRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		return
	}
	logger.Info("bind and validate", map[string]interface{}{
		"req": req,
	})
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

func (h *UserHandler) RefreshToken(c *gin.Context) {
	// ✅ Lấy refresh token từ cookie (không phải body)
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Missing refresh token", nil)
		return
	}

	// Call service để validate và generate new tokens
	newLoginResp, err := h.service.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// ✅ Set new refresh token in cookie
	c.SetCookie(
		"refresh_token",
		newLoginResp.RefreshToken,
		7*24*3600,
		"/",
		"",
		true,
		true,
	)

	// ✅ Remove from response body
	newLoginResp.RefreshToken = ""

	response.Success(c, http.StatusOK, "Token refreshed", newLoginResp)
}

// ResendVerification xử lý POST /auth/resend-verification
// @Summary      Resend verification email
// @Description  Send verification email again to user email
// @Router       /auth/resend-verification [post]
func (h *UserHandler) ResendVerification(c *gin.Context) {
	// STEP 1: PARSE REQUEST
	var req user.ResendVerificationRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		return
	}

	// STEP 3: RESEND VERIFICATION EMAIL
	if err := h.service.ResendVerification(c.Request.Context(), req.Email); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 4: SUCCESS
	response.Success(c, http.StatusOK, "If your email exists and not verified, verification link has been sent", nil)
}

// Login xử lý POST /auth/login - FR-AUTH-002
// @Summary      User login
// @Description  Authenticate user and return JWT tokens
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	// STEP 1: PARSE REQUEST
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	logger.Info("get request", map[string]interface{}{
		"req": req,
	})
	// STEP 2: VALIDATE
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// STEP 3: AUTHENTICATE
	// Service verify password, generate JWT tokens
	res, err := h.service.Login(c.Request.Context(), req)
	logger.Info("login request", map[string]interface{}{
		"res":   res,
		"error": err,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	// ✅ STEP 4: SET REFRESH TOKEN IN HTTPONLYCOOKIE
	c.SetCookie(
		"refresh_token",  // Cookie name
		res.RefreshToken, // Cookie value
		7*24*3600,        // Max age (7 days in seconds)
		"/",              // Path
		"",               // Domain (empty = auto-detect)
		true,             // Secure (HTTPS only)
		true,             // HttpOnly (JavaScript cannot access)
	)

	// ✅ STEP 5: REMOVE REFRESH TOKEN FROM RESPONSE BODY
	res.RefreshToken = "" // ← Đừng trả về body nữa

	// Set refresh token cookie
	c.SetCookie("refresh_token", res.RefreshToken, 7*24*3600, "/", "", true, true)
	res.RefreshToken = ""

	// Merge cart if user had anonymous session
	sessionID := middleware.GetSessionID(c)
	if sessionID != "" {
		if err := h.cartService.MergeCart(c.Request.Context(), sessionID, res.User.ID); err != nil {
			// Log error but DON'T fail login
			logger.Info("Failed to merge cart after login", map[string]interface{}{
				"user_id":    res.User.ID,
				"session_id": sessionID,
				"error":      err.Error(),
			})

		}

		// Clear session cookie
		c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", true, true)
	}

	// STEP 6: SUCCESS
	// Return JWT tokens để client lưu (localStorage/cookie)
	response.Success(c, http.StatusOK, "Login successful", res)
}

// Logout xử lý POST /auth/logout
// @Summary      User logout
// @Description  Logout user and clear refresh token cookie
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	// STEP 1: GET USER ID FROM CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		// If no user in context, still clear cookie and return success
		// This handles cases where token is expired but user wants to logout
		c.SetCookie("refresh_token", "", -1, "/", "", true, true)
		response.Success(c, http.StatusOK, "Logged out successfully", nil)
		return
	}

	// STEP 2: CALL SERVICE LAYER
	// Service logs the logout event for security monitoring
	if err := h.service.Logout(c.Request.Context(), userID); err != nil {
		h.handleError(c, err)
		return
	}

	// STEP 3: CLEAR REFRESH TOKEN COOKIE
	// Set MaxAge to -1 to delete the cookie
	c.SetCookie(
		"refresh_token", // Cookie name
		"",              // Empty value
		-1,              // MaxAge -1 = delete cookie
		"/",             // Path
		"",              // Domain
		true,            // Secure
		true,            // HttpOnly
	)

	// STEP 4: SUCCESS RESPONSE
	// Client should also clear access token from localStorage
	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

// VerifyEmail xử lý GET /auth/verify-email?token=xxx - FR-AUTH-001
// @Summary      Verify email address
// @Description  Confirm email verification token
// @Router       /auth/verify-email
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
// @Router       /auth/forgot-password
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
// @Router       /auth/reset-password
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
// @Security     BearerAuth
// @Router       /users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	// STEP 1: GET USER ID FROM CONTEXT
	// Middleware Auth() đã parse JWT và set userID vào context
	// c.Get("userID"): lấy value từ Gin context (type assertion cần thiết)
	userID, err := getUserIDFromContext(c)
	logger.Info("USER ID", map[string]interface{}{
		"USER_ID": userID,
	})
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
// @Security     BearerAuth
// @Router       /users/me
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
// @Security     BearerAuth
// @Router       /users/me/password
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
// @Router       /admin/users/{id}/status
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
	value, exists := c.Get("user_id")
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

// ✅ REFACTORED - Eliminate repetition
func (h *UserHandler) bindAndValidate(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return err
	}

	return nil
}
