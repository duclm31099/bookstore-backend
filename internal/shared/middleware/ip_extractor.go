package middleware

import (
	"bookstore-backend/pkg/logger"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// IPExtractorMiddleware extracts client IP and adds to context
func IPExtractorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := extractIPAddress(c.Request)

		// Add IP to context
		ctx := context.WithValue(c.Request.Context(), "client_ip", ip)

		// Also add X-Forwarded-For if available
		if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
			ctx = context.WithValue(ctx, "x_forwarded_for", xff)
		}

		c.Next()
	}
}

// extractIPAddress extracts real client IP from request
func extractIPAddress(r *http.Request) string {
	// 1. Try X-Real-IP header (set by reverse proxy like Nginx)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// 2. Try X-Forwarded-For header (may contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP (original client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 3. Fallback to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	logger.Info("extractIPAddress", map[string]interface{}{
		"ip": ip,
	})
	return ip
}
