package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bookstore-backend/internal/domains/order/model"
)

// =====================================================
// ORDER REPOSITORY INTERFACE
// =====================================================
type OrderRepository interface {
	// Transaction management
	BeginTx(ctx context.Context) (pgx.Tx, error)
	CommitTx(ctx context.Context, tx pgx.Tx) error
	RollbackTx(ctx context.Context, tx pgx.Tx) error

	// Order operations
	CreateOrder(ctx context.Context, order *model.Order) error
	CreateOrderWithTx(ctx context.Context, tx pgx.Tx, order *model.Order) error
	GetOrderByID(ctx context.Context, orderID uuid.UUID) (*model.Order, error)
	GetOrderByIDAndUserID(ctx context.Context, orderID, userID uuid.UUID) (*model.Order, error)
	GetOrderByNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string, version int) error
	UpdateOrderStatusWithTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, status string, version int) error
	CancelOrder(ctx context.Context, orderID uuid.UUID, reason string, version int) error
	UpdateOrderTracking(ctx context.Context, orderID uuid.UUID, trackingNumber string, version int) error
	UpdateOrderAdminNote(ctx context.Context, orderID uuid.UUID, adminNote string, version int) error

	// Order items operations
	CreateOrderItems(ctx context.Context, items []model.OrderItem) error
	CreateOrderItemsWithTx(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error
	GetOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error)

	// List operations
	ListOrdersByUserID(ctx context.Context, userID uuid.UUID, status string, page, limit int) ([]model.Order, int, error)
	ListAllOrders(ctx context.Context, status string, page, limit int) ([]model.Order, int, error)
	CountOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) (int, error)

	// Order status history
	CreateOrderStatusHistory(ctx context.Context, history *model.OrderStatusHistory) error
	CreateOrderStatusHistoryWithTx(ctx context.Context, tx pgx.Tx, history *model.OrderStatusHistory) error
	GetOrderStatusHistory(ctx context.Context, orderID uuid.UUID) ([]model.OrderStatusHistory, error)
}

// =====================================================
// WAREHOUSE REPOSITORY INTERFACE
// =====================================================
type WarehouseRepository interface {
	GetWarehouseByCode(ctx context.Context, code string) (*Warehouse, error)
	GetWarehouseIDByCode(ctx context.Context, code string) (uuid.UUID, error)
}

// Warehouse entity (simplified for order domain)
type Warehouse struct {
	ID       uuid.UUID
	Name     string
	Code     string
	Province string
	IsActive bool
}

// =====================================================
// INVENTORY REPOSITORY INTERFACE
// =====================================================
type InventoryRepository interface {
	// Reserve inventory using database function
	ReserveStock(ctx context.Context, tx pgx.Tx, warehouseID, bookID uuid.UUID, quantity int, userID uuid.UUID) error

	// Release inventory using database function
	ReleaseStock(ctx context.Context, tx pgx.Tx, warehouseID, bookID uuid.UUID, quantity int, userID uuid.UUID) error

	// Check available stock
	CheckAvailableStock(ctx context.Context, warehouseID, bookID uuid.UUID, requiredQty int) (bool, error)

	// Get available quantity
	GetAvailableQuantity(ctx context.Context, warehouseID, bookID uuid.UUID) (int, error)
}

// =====================================================
// ADDRESS REPOSITORY INTERFACE
// =====================================================
type AddressRepository interface {
	GetAddressByID(ctx context.Context, addressID uuid.UUID) (*Address, error)
	GetAddressByIDAndUserID(ctx context.Context, addressID, userID uuid.UUID) (*Address, error)
}

// Address entity (simplified for order domain)
type Address struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ReceiverName string
	Phone        string
	Province     string
	District     string
	Ward         string
	FullAddress  string
	IsDefault    bool
}

// =====================================================
// CART REPOSITORY INTERFACE
// =====================================================
type CartRepository interface {
	GetCartByUserID(ctx context.Context, userID uuid.UUID) (*Cart, error)
	GetCartItemsByCartID(ctx context.Context, cartID uuid.UUID) ([]CartItem, error)
	ClearCart(ctx context.Context, cartID uuid.UUID) error
	ClearCartWithTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) error
}

// Cart entities (simplified for order domain)
type Cart struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ItemsCount int
	Subtotal   string
}

type CartItem struct {
	ID           uuid.UUID
	CartID       uuid.UUID
	BookID       uuid.UUID
	BookTitle    string
	BookSlug     string
	BookCoverURL *string
	AuthorName   *string
	Quantity     int
	Price        string
	Subtotal     string
}

// =====================================================
// PROMOTION REPOSITORY INTERFACE
// =====================================================
type PromotionRepository interface {
	GetPromotionByCode(ctx context.Context, code string) (*Promotion, error)
	ValidatePromotion(ctx context.Context, promoID uuid.UUID, userID uuid.UUID, orderAmount string) error
	CreatePromotionUsage(ctx context.Context, promoID, userID, orderID uuid.UUID) error
	CreatePromotionUsageWithTx(ctx context.Context, tx pgx.Tx, promoID, userID, orderID uuid.UUID) error
}

// Promotion entity (simplified for order domain)
type Promotion struct {
	ID                uuid.UUID
	Code              string
	Type              string // "percentage" or "fixed"
	Value             string // decimal value
	MinOrderAmount    string
	MaxDiscountAmount *string
	UsageLimit        *int
	UsageCount        int
	StartDate         *string
	EndDate           *string
	IsActive          bool
}
