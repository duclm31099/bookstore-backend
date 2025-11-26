package job

import (
	"bookstore-backend/internal/domains/cart/model"
	inventoryModel "bookstore-backend/internal/domains/inventory/model"
	inventoryService "bookstore-backend/internal/domains/inventory/service"
	orderModel "bookstore-backend/internal/domains/order/model"
	orderRepo "bookstore-backend/internal/domains/order/repository"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

type AutoReleaseReservationHandler struct {
	orderRepo        orderRepo.OrderRepository
	inventoryService inventoryService.ServiceInterface
}

func NewAutoReleaseReservationHandler(
	orderRepo orderRepo.OrderRepository,
	inventoryService inventoryService.ServiceInterface,
) *AutoReleaseReservationHandler {
	return &AutoReleaseReservationHandler{
		orderRepo:        orderRepo,
		inventoryService: inventoryService,
	}
}

func (h *AutoReleaseReservationHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload model.AutoReleaseReservationPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger.Info("Processing auto-release reservation task", map[string]interface{}{
		"order_id":     payload.OrderID,
		"order_number": payload.OrderNumber,
	})

	// 1. Get order
	order, err := h.orderRepo.GetOrderByID(ctx, payload.OrderID)
	if err != nil {
		logger.Info("Failed to get order", map[string]interface{}{
			"order_id": payload.OrderID,
			"error":    err.Error(),
		})
		return fmt.Errorf("get order: %w", err)
	}

	// 2. Skip nếu đã paid
	if order.PaymentStatus == "paid" {
		logger.Info("Order already paid, skip auto-release", map[string]interface{}{
			"order_id": payload.OrderID,
			"status":   order.Status,
		})
		return nil
	}

	// 3. Skip nếu order đã bị huỷ / trả hàng bởi flow khác
	if order.Status == orderModel.OrderStatusCancelled || order.Status == orderModel.OrderStatusReturned {
		logger.Info("Order already cancelled/returned, skip auto-release", map[string]interface{}{
			"order_id": payload.OrderID,
			"status":   order.Status,
		})
		return nil
	}

	// Optional (an toàn hơn): chỉ auto-cancel những trạng thái cho phép
	if !(order.Status == orderModel.OrderStatusPending || order.Status == orderModel.OrderStatusConfirmed) {
		logger.Info("Order status not eligible for auto-cancel", map[string]interface{}{
			"order_id": payload.OrderID,
			"status":   order.Status,
		})
		return nil
	}

	// 4. Get order items để release reservations
	orderItems, err := h.orderRepo.GetOrderItemsByOrderID(ctx, payload.OrderID)
	if err != nil {
		logger.Info("Failed to get order items", map[string]interface{}{
			"order_id": payload.OrderID,
			"error":    err.Error(),
		})
		return fmt.Errorf("get order items: %w", err)
	}

	// 5. Release từng item (reservation)
	for _, item := range orderItems {
		if item.WarehouseID == nil {
			continue
		}

		releaseReq := inventoryModel.ReleaseStockRequest{
			WarehouseID: *item.WarehouseID,
			BookID:      item.BookID,
			Quantity:    item.Quantity,
			ReferenceID: payload.OrderID,
			UserID:      &payload.UserID,
			Reason:      stringPtr("payment_timeout"),
		}

		if _, err := h.inventoryService.ReleaseStock(ctx, releaseReq); err != nil {
			logger.Info("Failed to release stock", map[string]interface{}{
				"order_id":     payload.OrderID,
				"book_id":      item.BookID,
				"warehouse_id": item.WarehouseID,
				"error":        err.Error(),
			})
			// Tiếp tục với item khác, không return error
			continue
		}

		logger.Info("Released stock", map[string]interface{}{
			"order_id": payload.OrderID,
			"book_id":  item.BookID,
		})
	}

	// 6. Update order status to cancelled (auto-cancel)
	err = h.orderRepo.CancelOrder(ctx, payload.OrderID, "Payment timeout - auto-cancelled", order.Version)
	if err != nil {
		logger.Info("Failed to cancel order", map[string]interface{}{
			"order_id": payload.OrderID,
			"error":    err.Error(),
		})
		return fmt.Errorf("cancel order: %w", err)
	}

	logger.Info("Auto-released reservations and cancelled order", map[string]interface{}{
		"order_id":     payload.OrderID,
		"order_number": payload.OrderNumber,
	})

	return nil
}

func stringPtr(s string) *string {
	return &s
}
