package service

import (
	"context"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/order/model"
)

// =====================================================
// ORDER SERVICE INTERFACE
// =====================================================
type OrderService interface {
	// Create new order from cart items
	CreateOrder(ctx context.Context, userID uuid.UUID, req model.CreateOrderRequest) (*model.CreateOrderResponse, error)

	// Get order detail by ID
	GetOrderDetail(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*model.OrderDetailResponse, error)

	// List user's orders with pagination
	ListOrders(ctx context.Context, userID uuid.UUID, req model.ListOrdersRequest) (*model.ListOrdersResponse, error)

	// Cancel order (by user)
	CancelOrder(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, req model.CancelOrderRequest) error

	// Update order status (admin only)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, req model.UpdateOrderStatusRequest) error

	// Reorder (create new order from existing order)
	ReorderFromExisting(ctx context.Context, userID uuid.UUID, req model.ReorderRequest) (*model.CreateOrderResponse, error)

	// Admin: List all orders
	ListAllOrders(ctx context.Context, req model.ListOrdersRequest) (*model.ListOrdersResponse, error)
	// GetOrderByIDWithoutUser gets order without user verification (for system operations)
	GetOrderByIDWithoutUser(ctx context.Context, orderID uuid.UUID) (*model.OrderDetailResponse, error)

	// CancelOrderBySystem cancels order via system action (payment timeout, fraud, etc.)
	CancelOrderBySystem(ctx context.Context, orderID uuid.UUID, reason string, source string) error
	// Get order by number
	GetOrderByNumber(ctx context.Context, orderNumber string, userID uuid.UUID) (*model.OrderDetailResponse, error)
}
