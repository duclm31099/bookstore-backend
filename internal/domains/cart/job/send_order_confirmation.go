package job

import (
	"bookstore-backend/internal/domains/cart/model"
	emailInfra "bookstore-backend/internal/infrastructure/email"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

type SendOrderConfirmationHandler struct {
	emailService emailInfra.EmailService
}

func NewSendOrderConfirmationHandler(emailService emailInfra.EmailService) *SendOrderConfirmationHandler {
	return &SendOrderConfirmationHandler{
		emailService: emailService,
	}
}

func (h *SendOrderConfirmationHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload model.SendOrderConfirmationPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger.Info("Processing send order confirmation task", map[string]interface{}{
		"order_id":     payload.OrderID,
		"order_number": payload.OrderNumber,
		"email":        payload.UserEmail,
	})

	// Build email content
	subject := fmt.Sprintf("Đơn hàng #%s đã được đặt thành công", payload.OrderNumber)
	body := h.buildEmailBody(payload)

	// Send email
	emailReq := emailInfra.EmailRequest{
		To:      []string{payload.UserEmail},
		Subject: subject,
		Body:    body,
		IsHTML:  false,
	}

	if err := h.emailService.SendEmail(ctx, emailReq); err != nil {
		logger.Info("Failed to send order confirmation email", map[string]interface{}{
			"order_id": payload.OrderID,
			"email":    payload.UserEmail,
			"error":    err.Error(),
		})
		return fmt.Errorf("send email: %w", err)
	}

	logger.Info("Sent order confirmation email successfully", map[string]interface{}{
		"order_id": payload.OrderID,
		"email":    payload.UserEmail,
	})

	return nil
}

func (h *SendOrderConfirmationHandler) buildEmailBody(payload model.SendOrderConfirmationPayload) string {
	paymentMethodText := map[string]string{
		"cash_on_delivery": "Thanh toán khi nhận hàng (COD)",
		"bank_transfer":    "Chuyển khoản ngân hàng",
		"e_wallet":         "Ví điện tử",
		"credit_card":      "Thẻ tín dụng",
	}

	method := paymentMethodText[payload.PaymentMethod]
	if method == "" {
		method = payload.PaymentMethod
	}

	return fmt.Sprintf(`Chào bạn,

Cảm ơn bạn đã đặt hàng tại Bookstore!

Chi tiết đơn hàng:
- Mã đơn hàng: %s
- Ngày đặt: %s
- Tổng tiền: %s VND
- Phương thức thanh toán: %s

Dự kiến giao hàng: %s

Theo dõi đơn hàng của bạn tại: https://bookstore.com/orders/%s

Trân trọng,
Bookstore Team`,
		payload.OrderNumber,
		payload.OrderCreatedAt,
		payload.Total.String(),
		method,
		payload.EstimatedDelivery,
		payload.OrderNumber,
	)
}
