package handlers

import (
	model "bookstore-backend/internal/domains/user"
	"bookstore-backend/internal/infrastructure/email"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// internal/infrastructure/queue/email_verification_handler.go
func EmailVerificationHandler(emailSvc email.EmailService) func(ctx context.Context, t *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var p model.VerifyEmailPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return asynq.SkipRetry // Sai format payload, skip retry
		}
		link := fmt.Sprintf("http://localhost:8080/api/v1/auth/verify-email?token=%s", p.Token)
		emailData := email.VerificationEmailData{
			Email:      p.Email,
			VerifyLink: link,
			ExpiresIn:  "24 giờ",
		}

		err := emailSvc.SendVerificationEmail(ctx, emailData)
		if err != nil {
			return err // Lỗi mạng, SMTP, retry lại
		}
		return nil // Thành công
	}
}

// internal/infrastructure/queue/email_verification_handler.go
func EmailResetPasswordHandler(emailSvc email.EmailService) func(ctx context.Context, t *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var p model.ResetPasswordPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return asynq.SkipRetry // Sai format payload, skip retry
		}

		emailData := email.ResetPasswordData{
			Email:     p.Email,
			Token:     p.ResetToken,
			ExpiresIn: "24 giờ",
		}
		err := emailSvc.SendResetPasswordEmail(ctx, emailData)
		if err != nil {
			return err // Lỗi mạng, SMTP, retry lại
		}
		return nil // Thành công
	}
}
