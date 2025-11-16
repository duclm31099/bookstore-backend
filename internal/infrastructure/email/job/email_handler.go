// ✅ internal/infrastructure/email/job/email_handler.go
package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/infrastructure/email"
)

// ============================================
// Email Verification Handler
// ============================================

type EmailVerificationHandler struct {
	emailService email.EmailService
}

func NewEmailVerificationHandler(emailService email.EmailService) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		emailService: emailService,
	}
}

func (h *EmailVerificationHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload email.VerificationEmailData
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal EmailVerification payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("email", payload.Email).
		Msg("Processing email verification")

	if err := h.emailService.SendVerificationEmail(ctx, payload); err != nil {
		log.Error().Err(err).Msg("Failed to send verification email")
		return fmt.Errorf("send verification email: %w", err)
	}

	log.Info().
		Str("email", payload.Email).
		Msg("Verification email sent successfully")

	return nil
}

// ============================================
// Reset Password Email Handler
// ============================================

type ResetPasswordEmailHandler struct {
	emailService email.EmailService
}

func NewResetPasswordEmailHandler(emailService email.EmailService) *ResetPasswordEmailHandler {
	return &ResetPasswordEmailHandler{
		emailService: emailService,
	}
}

func (h *ResetPasswordEmailHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload email.ResetPasswordData
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal ResetPasswordEmail payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("email", payload.Email).
		Msg("Processing reset password email")

	if err := h.emailService.SendResetPasswordEmail(ctx, payload); err != nil {
		log.Error().Err(err).Msg("Failed to send reset password email")
		return fmt.Errorf("send reset password email: %w", err)
	}

	log.Info().
		Str("email", payload.Email).
		Msg("Reset password email sent successfully")

	return nil
}
