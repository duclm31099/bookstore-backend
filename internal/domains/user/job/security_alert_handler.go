package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/domains/user"
	"bookstore-backend/internal/infrastructure/email"
	"bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
)

type SecurityAlertHandler struct {
	emailService email.EmailService
	userRepo     user.Repository // ✅ Use shared interface
}

func NewSecurityAlertHandler(
	emailService email.EmailService,
	userRepo user.Repository,
) *SecurityAlertHandler {
	return &SecurityAlertHandler{
		emailService: emailService,
		userRepo:     userRepo,
	}
}

func (h *SecurityAlertHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload shared.SecurityAlertPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal SecurityAlert payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("user_id", payload.UserID).
		Str("alert_type", string(payload.AlertType)).
		Str("ip_address", payload.IPAddress).
		Msg("Processing security alert")

	uid := utils.ParseStringToUUID(payload.UserID)
	// Get user basic info
	user, err := h.userRepo.FindByID(ctx, uid)
	if err != nil {
		log.Error().Err(err).Str("user_id", payload.UserID).Msg("User not found")
		return fmt.Errorf("get user info: %w", err)
	}

	// Build email content
	subject, body := h.buildEmailContent(payload, user.FullName)

	// Send email
	if err := h.emailService.SendEmail(ctx, email.EmailRequest{
		To:      []string{payload.Email},
		Subject: subject,
		Body:    body,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to send security alert email")
		return fmt.Errorf("send email: %w", err)
	}

	log.Info().
		Str("user_id", payload.UserID).
		Str("alert_type", string(payload.AlertType)).
		Msg("Security alert sent successfully")

	return nil
}

func (h *SecurityAlertHandler) buildEmailContent(payload shared.SecurityAlertPayload, fullName string) (string, string) {
	now := time.Now().Format("2006-01-02 15:04:05")

	switch payload.AlertType {
	case shared.AlertNewDeviceLogin:
		subject := "🔐 Đăng nhập từ thiết bị mới"
		body := fmt.Sprintf(`
Xin chào %s,

Chúng tôi phát hiện một lần đăng nhập từ thiết bị mới:

- Thời gian: %s
- Thiết bị: %s
- Trình duyệt: %s
- IP Address: %s

Nếu đây là bạn, bạn có thể bỏ qua email này.
Nếu không phải, vui lòng đổi mật khẩu ngay lập tức.

Trân trọng,
Bookstore Team
        `, fullName, now,
			payload.DeviceInfo["device"],
			payload.DeviceInfo["browser"],
			payload.IPAddress)
		return subject, body

	case shared.AlertAccountLocked:
		subject := "⚠️ Tài khoản bị khóa tạm thời"
		body := fmt.Sprintf(`
Xin chào %s,

Tài khoản của bạn đã bị khóa tạm thời (15 phút) do phát hiện nhiều lần đăng nhập thất bại:

- Thời gian: %s
- IP Address: %s

Nếu không phải bạn thực hiện, vui lòng đổi mật khẩu sau khi mở khóa.

Trân trọng,
Bookstore Team
        `, fullName, now, payload.IPAddress)
		return subject, body

	case shared.AlertPasswordChanged:
		subject := "✅ Mật khẩu đã được thay đổi"
		body := fmt.Sprintf(`
Xin chào %s,

Mật khẩu tài khoản của bạn vừa được thay đổi:

- Thời gian: %s
- IP Address: %s

Nếu không phải bạn thực hiện, vui lòng liên hệ ngay.

Trân trọng,
Bookstore Team
        `, fullName, now, payload.IPAddress)
		return subject, body

	default:
		return "Cảnh báo bảo mật", fmt.Sprintf("Phát hiện hoạt động bảo mật lúc %s", now)
	}
}
