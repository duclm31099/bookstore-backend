package middleware

import (
	"bookstore-backend/pkg/logger"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthMiddleware - Middleware xác thực JWT token
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Lấy token từ Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 2. Extract token từ "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}
		token := parts[1]

		// 3. Verify và parse JWT
		claims := jwt.MapClaims{}
		parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			// Kiểm tra signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !parsedToken.Valid {
			c.JSON(401, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		logger.Info("claims", map[string]interface{}{
			"claims": claims,
		})
		// 4. Extract userID từ claims
		userIDStr, ok := claims["user_id"].(string) // "sub" = subject (user ID)
		if !ok {
			c.JSON(401, gin.H{"error": "invalid user ID in token"})
			c.Abort()
			return
		}

		// 5. Convert string sang uuid.UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid UUID format"})
			c.Abort()
			return
		}

		// 6. Set userID vào context ✓ ĐÂY LÀ CHÌA KHÓA
		c.Set("userID", userID)

		// Tiếp tục xử lý request
		c.Next()
	}
}
