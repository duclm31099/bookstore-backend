package job

import (
	"bookstore-backend/internal/domains/user"
	"bookstore-backend/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

type CleanupExpiredTokensPayload struct {
	Date time.Time `json:"date,omitempty"`
}

type CleanupExpiredTokenHandler struct {
	userRepo user.Repository
}

func NewCleanupExpiredTokenHandler(userRepo user.Repository) *CleanupExpiredTokenHandler {
	return &CleanupExpiredTokenHandler{
		userRepo: userRepo,
	}
}

func (h *CleanupExpiredTokenHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload CleanupExpiredTokensPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		logger.Error("Unmarshal fail due to ", err)
		return err
	}

	cleanupDate := time.Now()
	if !payload.Date.IsZero() {
		cleanupDate = payload.Date
	}

	// Cleanup verification tokens đã hết hạn (>24 giờ)
	verifycationCutoff := cleanupDate.Add(-24 * time.Hour)
	deletedVerify, err := h.userRepo.DeleteExpiredVerifyTokens(ctx, verifycationCutoff)
	if err != nil {
		logger.Error("Delete expired verify token fail due to ", err)
		return err
	}

	// Cleanup reset tokens đã hết hạn (>1 giờ)
	resetCutoff := cleanupDate.Add(-1 * time.Hour)
	deletedReset, err := h.userRepo.DeleteExpiredResetTokens(ctx, resetCutoff)
	if err != nil {
		logger.Error("Delete expired reset token failed due to ", err)
		return err
	}
	logger.Info("Cleanup Expired Token result", map[string]interface{}{
		"deleted_verify_tokens": deletedVerify,
		"deleted_reset_tokens":  deletedReset,
	})

	return nil
}
