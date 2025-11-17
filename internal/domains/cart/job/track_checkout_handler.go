package job

import (
	"bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

type TrackCheckoutHandler struct {
	// Add analytics service here if you have one
}

func NewTrackCheckoutHandler() *TrackCheckoutHandler {
	return &TrackCheckoutHandler{}
}

func (h *TrackCheckoutHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload model.TrackCheckoutPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger.Info("Processing track checkout task", map[string]interface{}{
		"order_id":       payload.OrderID,
		"order_number":   payload.OrderNumber,
		"user_id":        payload.UserID,
		"total":          payload.Total.String(),
		"item_count":     payload.ItemCount,
		"payment_method": payload.PaymentMethod,
		"promo_code":     payload.PromoCode,
		"discount":       payload.Discount.String(),
	})

	// TODO: Send to analytics service (Google Analytics, Mixpanel, etc.)
	// Example: h.analyticsService.TrackCheckout(ctx, payload)

	logger.Info("Tracked checkout successfully", map[string]interface{}{
		"order_id": payload.OrderID,
	})

	return nil
}
