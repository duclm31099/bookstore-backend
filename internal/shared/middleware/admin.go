package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware checks if user has admin role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context (set by AuthMiddleware)
		roleInterface, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Access denied: admin role required",
			})
			c.Abort()
			return
		}

		role, ok := roleInterface.(string)
		if !ok || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Access denied: admin role required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
