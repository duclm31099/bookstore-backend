package job

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"

	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/pkg/logger"
)

// ================================================
// CLEANUP EXPIRED NOTIFICATIONS JOB HANDLER
// ================================================

type CleanupExpiredNotificationsHandler struct {
	notificationService service.NotificationService
}

func NewCleanupExpiredNotificationsHandler(
	notificationService service.NotificationService,
) *CleanupExpiredNotificationsHandler {
	return &CleanupExpiredNotificationsHandler{
		notificationService: notificationService,
	}
}

func (h *CleanupExpiredNotificationsHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	logger.Info("Starting CleanupExpiredNotifications job", nil)
	deleted, err := h.notificationService.CleanupExpiredNotifications(ctx)
	if err != nil {
		return fmt.Errorf("cleanup expired notifications: %w", err)
	}
	logger.Info("Completed CleanupExpiredNotifications job", map[string]interface{}{
		"deleted_count": deleted,
	})
	return nil
}
