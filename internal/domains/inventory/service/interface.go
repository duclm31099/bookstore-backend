package service

import (
	"bookstore-backend/internal/domains/inventory/model"
	"context"

	"github.com/google/uuid"
)

// Service defines the business logic for inventory management
type ServiceInterface interface {
	// CreateInventory creates new inventory record(s)
	// If request.WarehouseLocation = "ALL", creates inventory for all 4 warehouses
	// Otherwise creates for specified warehouse only
	// Returns ErrInventoryAlreadyExists if duplicate exists
	// Returns ErrBookNotFound if book does not exist
	CreateInventory(ctx context.Context, req model.CreateInventoryRequest) ([]model.InventoryResponse, error)

	// GetInventoryByID retrieves inventory by ID
	// Returns ErrInventoryNotFound if not exists
	GetInventoryByID(ctx context.Context, id uuid.UUID) (*model.InventoryResponse, error)

	// SearchInventory retrieves inventory by book_id + warehouse_location
	// Returns ErrInventoryNotFound if not exists
	SearchInventory(ctx context.Context, req model.SearchInventoryRequest) (*model.InventoryResponse, error)

	// UpdateInventory updates inventory with optimistic locking
	// Only updates non-nil fields in request
	// Returns ErrOptimisticLockFailed if version mismatch (concurrent update detected)
	// Returns ErrInventoryNotFound if not exists
	// Returns ErrReservedExceedsQuantity if validation fails
	UpdateInventory(ctx context.Context, id uuid.UUID, req model.UpdateInventoryRequest) (*model.InventoryResponse, error)

	// DeleteInventory deletes inventory
	// Only allows deletion if quantity = 0 AND reserved_quantity = 0
	// Returns ErrCannotDeleteNonEmptyInventory if validation fails
	// Returns ErrInventoryNotFound if not exists
	DeleteInventory(ctx context.Context, id uuid.UUID) error

	// ListInventories retrieves paginated inventory list with filters
	// Supports filtering by: book_id, warehouse_location, is_low_stock
	// Returns empty list if no records match
	ListInventories(ctx context.Context, req model.ListInventoryRequest) (*model.ListInventoryResponse, error)

	// ReserveStock reserves inventory for pending order/purchase
	// Atomically increases reserved_quantity if enough available stock exists
	// Returns ErrInsufficientStock if not enough available stock
	// Creates inventory_movement record for audit trail
	ReserveStock(ctx context.Context, req model.ReserveStockRequest) (*model.ReserveStockResponse, error)

	// ReleaseStock releases previously reserved inventory
	// Used when order is cancelled, expired, or payment failed
	// Atomically decreases reserved_quantity
	// Returns ErrInvalidReleaseQuantity if trying to release more than reserved
	ReleaseStock(ctx context.Context, req model.ReleaseStockRequest) (*model.ReleaseStockResponse, error)

	// CheckAvailability checks stock availability for multiple items
	// Returns per-item details and overall fulfillability status
	// Can check across all warehouses or prefer specific warehouse
	// Does NOT reserve stock, just checks availability
	CheckAvailability(ctx context.Context, req model.CheckAvailabilityRequest) (*model.CheckAvailabilityResponse, error)

	// GetStockSummary gets total available stock across all warehouses for a book
	GetStockSummary(ctx context.Context, bookID uuid.UUID) (*model.StockSummaryResponse, error)

	// CreateMovement creates manual inventory adjustment with audit trail
	// Validates movement type and quantity
	// Returns created movement record
	CreateMovement(ctx context.Context, req model.CreateMovementRequest) (*model.MovementResponse, error)

	// ListMovements lists inventory movements with pagination and filters
	// Can filter by inventory_id, movement_type, reference_type
	ListMovements(ctx context.Context, req model.ListMovementsRequest) (*model.ListMovementsResponse, error)

	// GetMovementStats gets aggregated movement statistics for a book
	GetMovementStats(ctx context.Context, bookID uuid.UUID) (*model.MovementStatsResponse, error)

	// GetInventoryDashboard gets comprehensive dashboard metrics
	// Includes summary, warehouse metrics, low stock alerts, etc.
	GetInventoryDashboard(ctx context.Context, req model.DashboardRequest) (*model.InventoryDashboardResponse, error)

	// GetInventoryValue calculates total inventory value with breakdown
	// Requires books table join for pricing information
	GetInventoryValue(ctx context.Context) (*model.InventoryValueResponse, error)

	// GetLowStockAlerts gets all items below low stock threshold
	GetLowStockAlerts(ctx context.Context) ([]model.LowStockItem, error)

	// GetOutOfStockItems gets all items with zero quantity
	GetOutOfStockItems(ctx context.Context) ([]model.OutOfStockItem, error)
}
