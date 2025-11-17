package job

import (
	"bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/internal/domains/cart/repository"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

type ClearCartHandler struct {
	cartRepo repository.RepositoryInterface
}

func NewClearCartHandler(cartRepo repository.RepositoryInterface) *ClearCartHandler {
	return &ClearCartHandler{
		cartRepo: cartRepo,
	}
}

func (h *ClearCartHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload model.ClearCartPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger.Info("Processing clear cart task", map[string]interface{}{
		"cart_id": payload.CartID,
		"user_id": payload.UserID,
	})

	// Clear cart items
	deletedCount, err := h.cartRepo.ClearCartItems(ctx, payload.CartID)
	if err != nil {
		logger.Info("Failed to clear cart items", map[string]interface{}{
			"cart_id": payload.CartID,
			"error":   err.Error(),
		})
		return fmt.Errorf("clear cart items: %w", err)
	}

	logger.Info("Cleared cart successfully", map[string]interface{}{
		"cart_id":       payload.CartID,
		"deleted_count": deletedCount,
	})

	return nil
}
