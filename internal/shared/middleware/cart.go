package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ===================================
// INTERFACES
// ===================================

// CartServiceInterface defines minimal interface for cart operations
// Used for dependency injection to avoid circular dependencies
type CartServiceInterface interface {
	GetUserCartID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	GetOrCreateCartBySession(ctx context.Context, sessionID string) (uuid.UUID, error)
}

// ===================================
// CONSTANTS
// ===================================

const (
	// Cookie settings
	SessionCookieName = "session_id"
	SessionMaxAge     = 60 * 60 * 24 * 30 // 30 days in seconds

	// Context keys
	ContextKeyCartID          = "cart_id"
	ContextKeyIsAnonymousCart = "is_anonymous_cart"
	ContextKeySessionID       = "session_id"
	ContextKeyUserID          = "user_id"
)

// ===================================
// MIDDLEWARE CONFIGURATION
// ===================================

// CartMiddlewareConfig holds configuration for cart middleware
type CartMiddlewareConfig struct {
	CartService    CartServiceInterface
	CookieDomain   string        // e.g., "bookstore.com" or "" for current domain
	CookiePath     string        // Default: "/"
	CookieSecure   bool          // true for HTTPS only
	CookieSameSite http.SameSite // Strict, Lax, or None
}

// DefaultCartMiddlewareConfig returns secure default configuration
func DefaultCartMiddlewareConfig(cartService CartServiceInterface) CartMiddlewareConfig {
	return CartMiddlewareConfig{
		CartService:    cartService,
		CookieDomain:   "", // Current domain
		CookiePath:     "/",
		CookieSecure:   true,                 // HTTPS only (set false for localhost dev)
		CookieSameSite: http.SameSiteLaxMode, // Lax: balance security & UX
	}
}

// ===================================
// CART MIDDLEWARE
// ===================================

// CartMiddleware handles cart identification for both authenticated and anonymous users
//
// Flow:
// 1. Check if user is authenticated (user_id in context from auth middleware)
// 2. If authenticated → fetch user's cart from database
// 3. If not authenticated → use session_id from cookie
// 4. If no session_id → generate new UUID and set cookie
// 5. Set cart_id, is_anonymous_cart, session_id in context for handlers
//
// Usage:
//
//	router.Use(middleware.CartMiddleware(config))
// middleware/cart_middleware.go

// CartMiddleware now works with OptionalAuthMiddleware
// Assumes OptionalAuthMiddleware has run before this
// Sets: user_id (if auth), session_id, cart_id, is_anonymous_cart
func CartMiddleware(config CartMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cartID uuid.UUID
		var isAnonymous bool
		var sessionID string

		// ===================================
		// STEP 1: Check if authenticated (from OptionalAuthMiddleware)
		// ===================================
		userID, isAuth := GetAuthenticatedUserID(c) // Our helper function

		if isAuth && userID != nil {
			// Try to get user's cart
			userCartID, err := config.CartService.GetUserCartID(c.Request.Context(), *userID)
			if err == nil && userCartID != uuid.Nil {
				cartID = userCartID
				isAnonymous = false

				c.Set(ContextKeyCartID, cartID)
				c.Set(ContextKeyIsAnonymousCart, isAnonymous)
				c.Set(ContextKeyUserID, *userID)
				c.Next()
				return
			}
		}

		// ===================================
		// STEP 2: Anonymous user - get/create session
		// ===================================
		sessionID = getSessionID(c)
		if sessionID == "" {
			// Generate new session ID
			sessionID = uuid.New().String()
			setSessionCookie(c, sessionID, config)
		}

		// ===================================
		// STEP 3: Get or create anonymous cart
		// ===================================
		anonCartID, err := config.CartService.GetOrCreateCartBySession(c.Request.Context(), sessionID)
		if err != nil {
			c.Set("cart_error", err.Error())
		} else {
			cartID = anonCartID
		}

		isAnonymous = true

		// ===================================
		// STEP 4: Set context
		// ===================================
		c.Set(ContextKeyCartID, cartID)
		c.Set(ContextKeyIsAnonymousCart, isAnonymous)
		c.Set(ContextKeySessionID, sessionID)

		c.Next()
	}
}

// ===================================
// HELPER FUNCTIONS
// ===================================

// getSessionID retrieves session ID from cookie
func getSessionID(c *gin.Context) string {
	sessionID, err := c.Cookie(SessionCookieName)
	if err != nil || sessionID == "" {
		return ""
	}

	// Validate UUID format for security
	if _, err := uuid.Parse(sessionID); err != nil {
		return "" // Invalid format → generate new
	}

	return sessionID
}

// setSessionCookie sets secure session cookie
func setSessionCookie(c *gin.Context, sessionID string, config CartMiddlewareConfig) {
	c.SetCookie(
		SessionCookieName,   // name
		sessionID,           // value
		SessionMaxAge,       // maxAge (30 days)
		config.CookiePath,   // path
		config.CookieDomain, // domain
		config.CookieSecure, // secure (HTTPS only)
		true,                // httpOnly (prevent XSS)
	)
}

// ===================================
// CONTEXT HELPERS FOR HANDLERS
// ===================================

// GetCartID retrieves cart ID from context
func GetCartID(c *gin.Context) (uuid.UUID, error) {
	cartIDValue, exists := c.Get(ContextKeyCartID)
	if !exists {
		return uuid.Nil, ErrCartIDNotFound
	}

	cartID, ok := cartIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrInvalidCartID
	}

	return cartID, nil
}

// IsAnonymousCart checks if current cart is anonymous
func IsAnonymousCart(c *gin.Context) bool {
	isAnon, exists := c.Get(ContextKeyIsAnonymousCart)
	if !exists {
		return true // Default to anonymous if not set
	}

	anonymous, ok := isAnon.(bool)
	if !ok {
		return true
	}

	return anonymous
}

// GetSessionID retrieves session ID from context
func GetSessionID(c *gin.Context) string {
	sessionID, exists := c.Get(ContextKeySessionID)
	if !exists {
		return ""
	}

	sid, ok := sessionID.(string)
	if !ok {
		return ""
	}

	return sid
}

// ===================================
// ERRORS
// ===================================

var (
	ErrCartIDNotFound = fmt.Errorf("cart_id not found in context")
	ErrInvalidCartID  = fmt.Errorf("invalid cart_id type in context")
)

// OptionalAuthMiddleware allows both authenticated and anonymous users
// - If JWT token exists & valid → set user_id in context
// - If no JWT or invalid → continue as anonymous (no error)
// - Always sets is_authenticated flag for downstream middleware
//
// Usage:
//
//	router.Use(OptionalAuthMiddleware(jwtSecret))
//
// In handlers:
//
//	userID, exists := c.Get("user_id")  // UUID or nil
//	isAuth, _ := c.Get("is_authenticated") // bool
func OptionalAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ===================================
		// STEP 1: Extract token from header
		// ===================================
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided → anonymous user
			c.Set("is_authenticated", false)
			c.Set("user_id", nil)
			c.Next()
			return
		}

		// Expected format: "Bearer <token>"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			// Invalid format → treat as anonymous (don't error)
			c.Set("is_authenticated", false)
			c.Set("user_id", nil)
			c.Next()
			return
		}

		token := headerParts[1]

		// ===================================
		// STEP 2: Verify token
		// ===================================
		claims, err := VerifyToken(token, jwtSecret)
		if err != nil {
			// Token invalid/expired → anonymous (log but don't error)
			// In production, log security event
			c.Set("is_authenticated", false)
			c.Set("user_id", nil)
			c.Next()
			return
		}
		// ===================================
		// STEP 3: Extract user_id from claims
		// ===================================
		userIDStr, ok := claims["user_id"].(string) // "sub" = subject (standard JWT claim)
		if !ok || userIDStr == "" {
			// No user ID in token → anonymous
			c.Set("is_authenticated", false)
			c.Set("user_id", nil)
			c.Next()
			return
		}

		// Parse UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Invalid UUID format → anonymous
			c.Set("is_authenticated", false)
			c.Set("user_id", nil)
			c.Next()
			return
		}

		// ===================================
		// STEP 4: Valid authenticated user
		// ===================================
		c.Set("is_authenticated", true)
		c.Set("user_id", userID)

		c.Next()
	}
}

// ===================================
// HELPER FUNCTION: Get User ID (if authenticated)
// ===================================

// GetAuthenticatedUserID retrieves user ID if user is authenticated
// Returns: (userID, true) if authenticated, (nil, false) if anonymous
func GetAuthenticatedUserID(c *gin.Context) (*uuid.UUID, bool) {
	isAuth, exists := c.Get("is_authenticated")
	if !exists || !isAuth.(bool) {
		return nil, false
	}

	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		return nil, false
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return nil, false
	}

	return &uid, true
}

// IsAuthenticated checks if user is authenticated
func IsAuthenticated(c *gin.Context) bool {
	isAuth, exists := c.Get("is_authenticated")
	return exists && isAuth.(bool)
}

func VerifyToken(tokenString string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		jwt.MapClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}
