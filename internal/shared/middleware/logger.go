package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Info().
			Str("request_id", c.GetString("request_id")).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("latency_ms", latency).
			Str("ip", c.ClientIP()).
			Msg("HTTP Request")
	}
}
