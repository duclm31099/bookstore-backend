package email

// internal/infrastructure/email/smtp_service.go
import (
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"
	"net/smtp"
)

type VerificationEmailData struct {
	Email      string
	VerifyLink string
	ExpiresIn  string
}
type ResetPasswordData struct {
	Email     string
	Token     string
	ExpiresIn string
}

type EmailService interface {
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
