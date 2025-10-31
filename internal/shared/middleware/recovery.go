package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Str("request_id", c.GetString("request_id")).
					Interface("error", err).
					Msg("Panic recovered")

				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "SYS_001",
						"message": "Internal server error",
					},
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
