package handler

import (
	bookService "bookstore-backend/internal/domains/book/service"
	"bookstore-backend/internal/shared/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type BulkImportHandler struct {
	service bookService.BulkImportServiceInterface
}

// NewBulkImportHandler tạo handler mới
func NewBulkImportHandler(service bookService.BulkImportServiceInterface) *BulkImportHandler {
	return &BulkImportHandler{
		service: service,
	}
}

// ImportBooks - POST /v1/admin/books/bulk-import
// Yêu cầu: Admin role (middleware check trước khi vào handler)
func (h *BulkImportHandler) ImportBooks(c *gin.Context) {
	// 1. Lấy user ID từ context (đã được middleware auth set trước đó)
	userID, ok := c.Get("user_id")
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", "user_id not found in context")
		return
	}

	// 2. Parse multipart form và lấy file "file"
	file, err := c.FormFile("file")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from request")
		response.Error(c, http.StatusBadRequest, "Invalid request", "file is required (multipart/form-data)")
		return
	}

	log.Info().
		Str("user_id", userID.(string)).
		Str("file_name", file.Filename).
		Int64("file_size", file.Size).
		Msg("[BulkImportHandler] Received bulk import request")

	// 3. Gọi service để xử lý import (sync)
	result, svcErr := h.service.ImportBooks(c.Request.Context(), file, userID.(string))
	if svcErr != nil {
		log.Error().Err(svcErr).Msg("Bulk import service error")
		response.Error(c, http.StatusInternalServerError, "Bulk import failed", svcErr.Error())
		return
	}

	// 4. Nếu result.Success = false → trả về 400 với chi tiết lỗi
	if !result.Success {
		// Có thể trả về 422 Unprocessable Entity (phù hợp cho validation error)
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success":      false,
			"total_rows":   result.TotalRows,
			"success_rows": result.SuccessRows,
			"failed_rows":  result.FailedRows,
			"errors":       result.Errors,
		})
		return
	}

	// 5. Thành công → trả về 201 Created với summary và list book IDs
	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"message":       "Bulk import completed successfully",
		"total_rows":    result.TotalRows,
		"success_rows":  result.SuccessRows,
		"created_books": result.CreatedBooks,
	})
}
