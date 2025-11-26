package middleware

import (
	"context"

	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ClientIPMiddleware extracts the client IP address from the request
// and injects it into the context for downstream handlers to use.
//
// This middleware should be registered early in the middleware chain
// to ensure all handlers have access to the client IP.
//
// Usage:
//
//	router.Use(middleware.ClientIPMiddleware())
func ClientIPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract client IP using utility function
		clientIP := utils.ExtractClientIP(c)

		// Inject IP into gin context (gin-specific)
		c.Set("client_ip", clientIP)

		// Inject IP into request context (for passing to services)
		ctx := context.WithValue(c.Request.Context(), "client_ip", clientIP)
		c.Request = c.Request.WithContext(ctx)

		// Log IP info for debugging (can be disabled in production)
		logger.Info("Client IP extracted", map[string]interface{}{
			"ip":         clientIP,
			"is_private": utils.IsPrivateIP(clientIP),
			"path":       c.Request.URL.Path,
		})

		// Continue to next handler
		c.Next()
	}
}

// GetClientIPFromContext retrieves the client IP from context
// Returns empty string if not found
func GetClientIPFromContext(ctx context.Context) string {
	if ip := ctx.Value("client_ip"); ip != nil {
		if ipStr, ok := ip.(string); ok {
			return ipStr
		}
	}
	return ""
}
