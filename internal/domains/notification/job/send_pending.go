package job

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"

	"bookstore-backend/internal/config"
	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
)

// ================================================
// SEND PENDING NOTIFICATIONS JOB HANDLER
// ================================================

// SendPendingNotificationsHandler xử lý job gửi notification chưa gửi
type SendPendingNotificationsHandler struct {
	notificationService service.NotificationService
	jobConfig           config.JobConfig
}

// NewSendPendingNotificationsHandler tạo handler mới
func NewSendPendingNotificationsHandler(
	notificationService service.NotificationService,
	jobConfig config.JobConfig,
) *SendPendingNotificationsHandler {
	return &SendPendingNotificationsHandler{
		notificationService: notificationService,
		jobConfig:           jobConfig,
	}
}

// ProcessTask là entrypoint của job
// Flow:
// 1. Đọc payload (có thể có trường limit, nếu không dùng mặc định 100)
// 2. Gọi NotificationService.ProcessUnsentNotifications(ctx, limit)
// 3. Log kết quả, không retry toàn job nếu một số notification fail (service tự xử lý)
func (h *SendPendingNotificationsHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	// 1. Parse payload (optional)
	type Payload struct {
		Limit int `json:"limit"`
	}

	var payload Payload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		// Nếu payload lỗi, vẫn có thể dùng default
		logger.Error("Failed to unmarshal send pending payload, using default limit", err)
	}

	// 2. Xác định limit
	limit := payload.Limit
	if limit <= 0 || limit > 100 || payload.Limit == 0 {
		limit = h.jobConfig.SendPendingLimit // mặc định theo yêu cầu
	}

	logger.Info("Starting SendPendingNotifications job", map[string]interface{}{
		"limit": limit,
	})

	// 3. Gọi service xử lý
	if err := h.notificationService.ProcessUnsentNotifications(ctx, limit); err != nil {
		// QUAN TRỌNG:
		// - ProcessUnsentNotifications sẽ cố gắng gửi từng notification.
		// - Nếu gặp lỗi hệ thống (DB, provider down...), trả lỗi để Asynq retry job.
		// - Nếu chỉ một vài notification fail (provider error từng cái),
		//   service nên log và tiếp tục, không fail job.
		return fmt.Errorf("process unsent notifications: %w", err)
	}

	logger.Info("Completed SendPendingNotifications job", map[string]interface{}{
		"limit": limit,
	})

	return nil
}
