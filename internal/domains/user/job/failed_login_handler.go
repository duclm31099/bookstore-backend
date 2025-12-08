package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/domains/user"
	"bookstore-backend/internal/shared"
	types "bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"
)

const (
	MaxFailedAttempts = 5
	LockoutDuration   = 15 * time.Minute
	AttemptWindow     = 15 * time.Minute
)

type FailedLoginHandler struct {
	cache       cache.Cache
	userRepo    user.Repository
	asynqClient *asynq.Client // ✅ For triggering alert
}

func NewFailedLoginHandler(
	cache cache.Cache,
	userRepo user.Repository,
	asynqClient *asynq.Client,
) *FailedLoginHandler {
	return &FailedLoginHandler{
		cache:       cache,
		userRepo:    userRepo,
		asynqClient: asynqClient,
	}
}

func (h *FailedLoginHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload types.FailedLoginPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal FailedLogin payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("user_id", payload.UserID).
		Str("ip_address", payload.IPAddress).
		Msg("Processing failed login attempt")

	attemptKey := fmt.Sprintf("failed_login:%s", payload.UserID)
	lockKey := fmt.Sprintf("account_locked:%s", payload.UserID)

	// Check if already locked
	isLocked, err := h.cache.Exists(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("check lock status: %w", err)
	}

	if isLocked {
		return nil
	}

	// Increment counter
	attempts, err := h.cache.Increment(ctx, attemptKey)
	if err != nil {
		return fmt.Errorf("increment counter: %w", err)
	}

	// Set expiry on first attempt
	if attempts == 1 {
		if err := h.cache.Expire(ctx, attemptKey, AttemptWindow); err != nil {
			logger.Error("Failed to set expiry", err)
		}
	}

	log.Info().
		Str("user_id", payload.UserID).
		Int64("attempts", attempts).
		Msg("Failed login attempts counted")

	// Lock account if threshold exceeded
	if attempts >= MaxFailedAttempts {
		if err := h.lockAccount(ctx, payload); err != nil {
			return fmt.Errorf("lock account: %w", err)
		}

		// Clear counter
		h.cache.Delete(ctx, attemptKey)
	} else if attempts >= 3 {
		// Warning after 3 attempts
		h.sendWarningAlert(ctx, payload, int(attempts))
	}

	return nil
}

func (h *FailedLoginHandler) lockAccount(ctx context.Context, payload types.FailedLoginPayload) error {
	lockKey := fmt.Sprintf("account_locked:%s", payload.UserID)

	// Set lock in cache
	if err := h.cache.Set(ctx, lockKey, "1", LockoutDuration); err != nil {
		return err
	}

	// Get user info
	user, err := h.userRepo.FindByID(ctx, utils.ParseStringToUUID(payload.UserID))
	if err != nil {
		return err
	}

	// Trigger alert
	alertPayload := types.SecurityAlertPayload{
		UserID:    payload.UserID,
		Email:     user.Email,
		AlertType: types.AlertAccountLocked,
		DeviceInfo: map[string]string{
			"detail": fmt.Sprintf("Locked after %d failed attempts", MaxFailedAttempts),
		},
		IPAddress: payload.IPAddress,
	}

	h.triggerSecurityAlert(ctx, alertPayload)

	log.Warn().
		Str("user_id", payload.UserID).
		Dur("duration", LockoutDuration).
		Msg("Account locked")

	return nil
}

func (h *FailedLoginHandler) sendWarningAlert(ctx context.Context, payload types.FailedLoginPayload, attempts int) {
	user, err := h.userRepo.FindByID(ctx, utils.ParseStringToUUID(payload.UserID))
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user for warning")
		return
	}

	alertPayload := types.SecurityAlertPayload{
		UserID:    payload.UserID,
		Email:     user.Email,
		AlertType: types.AlertSuspiciousActivity,
		DeviceInfo: map[string]string{
			"detail": fmt.Sprintf("%d failed login attempts", attempts),
		},
		IPAddress: payload.IPAddress,
	}

	h.triggerSecurityAlert(ctx, alertPayload)
}

func (h *FailedLoginHandler) triggerSecurityAlert(ctx context.Context, payload types.SecurityAlertPayload) {
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(shared.TypeSendSecurityAlert, data)

	_, err := h.asynqClient.EnqueueContext(
		ctx,
		task,
		asynq.Queue(shared.QueueAuth),
		asynq.MaxRetry(2),
		asynq.Timeout(30*time.Second),
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to enqueue security alert")
	}
}
