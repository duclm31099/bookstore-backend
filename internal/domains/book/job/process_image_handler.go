package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	bookService "bookstore-backend/internal/domains/book/service"
)

// ProcessImageHandler xử lý resize và upload variants của ảnh book
type ProcessImageHandler struct {
	bookImageService bookService.BookImageService
}

func NewProcessImageHandler(bookImageService bookService.BookImageService) *ProcessImageHandler {
	return &ProcessImageHandler{
		bookImageService: bookImageService,
	}
}

// ProcessTask xử lý background job resize ảnh
func (h *ProcessImageHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		ImageID string `json:"image_id"`
	}

	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal ProcessImage payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("image_id", payload.ImageID).
		Msg("Processing book image variants")

	// Gọi service xử lý ảnh
	err := h.bookImageService.ProcessImage(ctx, payload.ImageID)
	if err != nil {
		log.Error().
			Err(err).
			Str("image_id", payload.ImageID).
			Msg("Failed to process image")
		return fmt.Errorf("process image: %w", err)
	}

	log.Info().
		Str("image_id", payload.ImageID).
		Msg("Book image processed successfully")

	return nil
}
