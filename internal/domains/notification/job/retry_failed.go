package job

import (
	"context"
	"fmt"

	"bookstore-backend/internal/config"
	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"

	"github.com/hibiken/asynq"
)

// ================================================
// RETRY FAILED DELIVERIES JOB HANDLER
// ================================================

type RetryFailedDeliveriesHandler struct {
	deliveryService service.DeliveryService
	jobConfig       config.JobConfig
}

func NewRetryFailedDeliveriesHandler(
	deliveryService service.DeliveryService,
	jobConfig config.JobConfig,
) *RetryFailedDeliveriesHandler {
	return &RetryFailedDeliveriesHandler{
		deliveryService: deliveryService,
		jobConfig:       jobConfig,
	}
}

func (h *RetryFailedDeliveriesHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload shared.RetryFailedPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		logger.Error("Failed to unmarshal retry_failed_deliveries payload, using default 50", err)
	}
	limit := payload.Limit
	if limit <= 0 || limit > 100 || payload.Limit == 0 {
		limit = h.jobConfig.RetryFailedLimit // mặc định dev, có thể tăng lên prod
	}

	logger.Info("Starting RetryFailedDeliveries job", map[string]interface{}{
		"limit": limit,
	})

	if err := h.deliveryService.RetryFailedDeliveries(ctx, limit); err != nil {
		return fmt.Errorf("retry failed deliveries: %w", err)
	}

	logger.Info("Completed RetryFailedDeliveries job", map[string]interface{}{
		"limit": limit,
	})

	return nil
}
