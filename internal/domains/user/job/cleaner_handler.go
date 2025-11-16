package job

import (
	"bookstore-backend/internal/domains/user"
	"bookstore-backend/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
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

	log.Info().
		Time("cleanup_date", cleanupDate).
		Msg("Starting cleanup of expired tokens")

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
	log.Info().
		Int("verification_tokens_deleted", deletedVerify).
		Int("reset_tokens_deleted", deletedReset).
		Msg("Successfully cleaned up expired tokens")

	return nil
}
