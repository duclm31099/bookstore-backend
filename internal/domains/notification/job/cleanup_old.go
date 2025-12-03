package job

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"bookstore-backend/internal/config"
	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
)

// ================================================
// CLEANUP OLD READ NOTIFICATIONS JOB HANDLER
// ================================================

type CleanupOldNotificationsHandler struct {
	notificationService service.NotificationService
	jobConfig           config.JobConfig
}

func NewCleanupOldNotificationsHandler(
	notificationService service.NotificationService,
	jobConfig config.JobConfig,
) *CleanupOldNotificationsHandler {
	return &CleanupOldNotificationsHandler{
		notificationService: notificationService,
		jobConfig:           jobConfig,
	}
}

// Payload optional: cho phép override số ngày, nhưng mặc định 30
type cleanupPayload struct {
	Days int `json:"days"`
}

func (h *CleanupOldNotificationsHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload cleanupPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		// Nếu payload lỗi thì dùng default 30 ngày
		logger.Error("Failed to unmarshal cleanup_old payload, using default 30 days", err)
	}

	days := payload.Days
	if days <= 0 {
		days = h.jobConfig.CleanupRetentionDays // mặc định theo yêu cầu
	}

	olderThan := time.Duration(days) * 24 * time.Hour

	logger.Info("Starting CleanupOldNotifications job", map[string]interface{}{
		"days":       days,
		"older_than": olderThan.String(),
	})

	deleted, err := h.notificationService.CleanupOldReadNotifications(ctx, olderThan)
	if err != nil {
		return fmt.Errorf("cleanup old read notifications: %w", err)
	}

	logger.Info("Completed CleanupOldNotifications job", map[string]interface{}{
		"days":          days,
		"deleted_count": deleted,
	})

	return nil
}
