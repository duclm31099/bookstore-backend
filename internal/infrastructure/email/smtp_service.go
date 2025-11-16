package email

// internal/infrastructure/email/smtp_service.go
import (
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog/log"
)

type EmailService interface {
	SendEmail(ctx context.Context, req EmailRequest) error
	SendResetPasswordEmail(ctx context.Context, data ResetPasswordData) error
	SendVerificationEmail(ctx context.Context, data VerificationEmailData) error
}

type smtpEmailService struct {
	smtpAddr string
	smtpFrom string
}

func NewDevEmailService(smtpHost, smtpPort string) EmailService {
	return &smtpEmailService{
		smtpAddr: smtpHost + ":" + smtpPort,
		smtpFrom: "noreply@bookstore.dev",
	}
}

func (s *smtpEmailService) SendResetPasswordEmail(ctx context.Context, data ResetPasswordData) error {
	subject := "Đặt lại mật khẩu tài khoản Bookstore"
	body := fmt.Sprintf(`Chào bạn,

	Vui lòng sử  dụng token sau để  đặt lại mật khẩu:
	%s

	Link có hiệu lực %s.

	Nếu bạn không đăng ký tài khoản này, vui lòng bỏ qua email này.`, data.Token, data.ExpiresIn)
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.smtpFrom, data.Email, subject, body))
	return smtp.SendMail(s.smtpAddr, nil, s.smtpFrom, []string{data.Email}, msg)
}

func (s *smtpEmailService) SendVerificationEmail(ctx context.Context, data VerificationEmailData) error {

	subject := "Xác thực tài khoản Bookstore"
	body := fmt.Sprintf(`Chào bạn,

	Vui lòng click vào link sau để xác thực tài khoản:
	%s

	Link có hiệu lực %s.

	Nếu bạn không đăng ký tài khoản này, vui lòng bỏ qua email này.`, data.VerifyLink, data.ExpiresIn)

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.smtpFrom, data.Email, subject, body))

	// Gửi email qua SMTP
	err := smtp.SendMail(s.smtpAddr, nil, s.smtpFrom, []string{data.Email}, msg)

	if err != nil {
		logger.Info("Failed to send email", map[string]interface{}{
			"error":     err.Error(),
			"to":        data.Email,
			"smtp_addr": s.smtpAddr,
		})
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// ✅ Implement SendEmail method
func (s *smtpEmailService) SendEmail(ctx context.Context, req EmailRequest) error {
	// Validate
	if len(req.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}
	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	// Build message
	message := s.buildMessage(req)

	// Send email
	err := smtp.SendMail(s.smtpAddr, nil, s.smtpFrom, req.To, []byte(message))
	if err != nil {
		log.Error().
			Err(err).
			Strs("to", req.To).
			Str("subject", req.Subject).
			Msg("Failed to send email")
		return fmt.Errorf("send email: %w", err)
	}

	log.Info().
		Strs("to", req.To).
		Str("subject", req.Subject).
		Msg("Email sent successfully")

	return nil
}

// buildMessage constructs the email message with headers and body
func (s *smtpEmailService) buildMessage(req EmailRequest) string {
	var builder strings.Builder

	// Headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", s.smtpFrom))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))

	if len(req.Cc) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.Cc, ", ")))
	}

	if len(req.Bcc) > 0 {
		builder.WriteString(fmt.Sprintf("Bcc: %s\r\n", strings.Join(req.Bcc, ", ")))
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))

	// Content type
	if req.IsHTML {
		builder.WriteString("MIME-Version: 1.0\r\n")
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	builder.WriteString("\r\n")

	// Body
	builder.WriteString(req.Body)

	return builder.String()
}
