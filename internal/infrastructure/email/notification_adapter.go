package email

import (
	"context"
	"fmt"
	"time"
)

// ================================================
// NOTIFICATION EMAIL ADAPTER
// Adapts existing EmailService to notification domain interface
// ================================================

type NotificationEmailProvider struct {
	emailService EmailService
}

func NewNotificationEmailProvider(emailService EmailService) *NotificationEmailProvider {
	return &NotificationEmailProvider{
		emailService: emailService,
	}
}

// SendEmail implements notification DeliveryService.EmailProvider interface
func (p *NotificationEmailProvider) SendEmail(ctx context.Context, to, subject, body string) (messageID string, err error) {
	req := EmailRequest{
		To:      []string{to},
		Subject: subject,
		Body:    body,
		IsHTML:  true, // Notification emails are HTML
	}

	if err := p.emailService.SendEmail(ctx, req); err != nil {
		return "", fmt.Errorf("send notification email: %w", err)
	}

	// Generate a pseudo message ID (SMTP doesn't return one)
	messageID = fmt.Sprintf("smtp-%s-%d", to, time.Now().Unix())
	return messageID, nil
}
