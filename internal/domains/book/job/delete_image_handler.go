package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	bookService "bookstore-backend/internal/domains/book/service"
)

// DeleteImagesHandler xóa tất cả ảnh của book (khi xóa/update book)
type DeleteImagesHandler struct {
	bookImageService bookService.BookImageService
}

func NewDeleteImagesHandler(bookImageService bookService.BookImageService) *DeleteImagesHandler {
	return &DeleteImagesHandler{
		bookImageService: bookImageService,
	}
}

// ProcessTask xóa ảnh từ MinIO và DB
func (h *DeleteImagesHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		BookID string `json:"book_id"`
	}

	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal DeleteImages payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("book_id", payload.BookID).
		Msg("Deleting book images")

	// Xóa tất cả ảnh của book
	err := h.bookImageService.DeleteBookImages(ctx, payload.BookID)
	if err != nil {
		log.Error().
			Err(err).
			Str("book_id", payload.BookID).
			Msg("Failed to delete book images")
		return fmt.Errorf("delete images: %w", err)
	}

	log.Info().
		Str("book_id", payload.BookID).
		Msg("Book images deleted successfully")

	return nil
}
