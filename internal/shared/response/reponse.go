package response

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Error struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type Meta struct {
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
	Total int `json:"total,omitempty"`
}

// Success responses
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, statusCode int, data interface{}, meta *Meta) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error responses
func ErrorResponse(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	})
}

func ErrorWithDetails(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// Common error responses
func BadRequest(c *gin.Context, message string) {
	ErrorResponse(c, 400, "BAD_REQUEST", message)
}

func Unauthorized(c *gin.Context, message string) {
	ErrorResponse(c, 401, "UNAUTHORIZED", message)
}

func Forbidden(c *gin.Context, message string) {
	ErrorResponse(c, 403, "FORBIDDEN", message)
}

func NotFound(c *gin.Context, message string) {
	ErrorResponse(c, 404, "NOT_FOUND", message)
}

func Conflict(c *gin.Context, message string) {
	ErrorResponse(c, 409, "CONFLICT", message)
}

func InternalServerError(c *gin.Context, message string) {
	ErrorResponse(c, 500, "INTERNAL_SERVER_ERROR", message)
}
