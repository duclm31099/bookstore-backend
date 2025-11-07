package repository

import (
	"bookstore-backend/internal/domains/inventory/model"
	"context"

	"github.com/google/uuid"
)

// Repository defines the contract for inventory data access
type RepositoryInterface interface {
	// Create inserts a new inventory record
	// Returns ErrInventoryAlreadyExists if duplicate (book_id, warehouse_location) exists
	Create(ctx context.Context, inventory *model.Inventory) error

	// CreateBatch inserts multiple inventory records in a single transaction
	// Used when creating inventory for all warehouses (warehouse_location = "ALL")
	CreateBatch(ctx context.Context, inventories []model.Inventory) error

	// GetByID retrieves inventory by primary key
	// Returns ErrInventoryNotFound if not exists
	GetByID(ctx context.Context, id uuid.UUID) (*model.Inventory, error)

	// GetByBookAndWarehouse retrieves inventory by unique constraint (book_id, warehouse_location)
	// Returns ErrInventoryNotFound if not exists
	GetByBookAndWarehouse(ctx context.Context, bookID uuid.UUID, warehouse string) (*model.Inventory, error)

	// Update updates inventory with optimistic locking
	// Returns ErrOptimisticLockFailed if version mismatch
	// Returns ErrInventoryNotFound if not exists
	Update(ctx context.Context, id uuid.UUID, inventory *model.Inventory) error

	// Delete removes inventory record
	// Should validate quantity = 0 and reserved_quantity = 0 before deletion
	// Returns ErrInventoryNotFound if not exists
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves paginated inventory records with filters
	// Returns empty slice if no records match
	List(ctx context.Context, filter model.ListInventoryRequest) ([]model.Inventory, int, error)

	// ExistsByBookID checks if any inventory exists for a book
	// ExistsByBookID(ctx context.Context, bookID uuid.UUID) (bool, error)

	// ReserveStock atomically reserves stock with row-level locking
	// Returns updated inventory after reservation
	// Returns ErrInsufficientStock if not enough available stock
	ReserveStock(ctx context.Context, bookID uuid.UUID, warehouse string, quantity int, referenceType string, referenceID uuid.UUID) (*model.Inventory, error)

	// ReleaseStock atomically releases reserved stock
	// Returns updated inventory after release
	// Returns ErrInvalidReleaseQuantity if trying to release more than reserved
	ReleaseStock(ctx context.Context, bookID uuid.UUID, warehouse string, quantity int, referenceID uuid.UUID) (*model.Inventory, error)

	// GetInventoriesByBook retrieves all warehouse inventories for a specific book
	// Returns sorted by warehouse location for consistent ordering
	GetInventoriesByBook(ctx context.Context, bookID uuid.UUID) ([]model.Inventory, error)

	// GetInventoriesByBooks retrieves inventories for multiple books (single query for efficiency)
	// Returns map of bookID -> []Inventory for quick lookup
	GetInventoriesByBooks(ctx context.Context, bookIDs []uuid.UUID) (map[uuid.UUID][]model.Inventory, error)

	// CreateMovement creates inventory movement record
	CreateMovement(ctx context.Context, movement *model.InventoryMovement) error

	// ListMovements lists movements with filters and pagination
	ListMovements(ctx context.Context, filter model.ListMovementsRequest) ([]model.InventoryMovement, int, error)

	// GetMovementsByInventoryID gets all movements for specific inventory
	// GetMovementsByInventoryID(ctx context.Context, inventoryID uuid.UUID, page, limit int) ([]model.InventoryMovement, int, error)

	// GetMovementStatsForBook gets aggregated stats for all movements of a book across warehouses
	GetMovementStatsForBook(ctx context.Context, bookID uuid.UUID) (*model.MovementStatsResponse, error)

	// GetDashboardMetrics retrieves all metrics for dashboard
	GetDashboardMetrics(ctx context.Context) (*model.DashboardSummary, error)

	// GetWarehouseMetrics retrieves metrics for all warehouses
	GetWarehouseMetrics(ctx context.Context) ([]model.WarehouseMetrics, error)

	// GetLowStockItems retrieves all items below threshold
	GetLowStockItems(ctx context.Context) ([]model.LowStockItem, error)

	// GetOutOfStockItems retrieves all zero-quantity items
	GetOutOfStockItems(ctx context.Context) ([]model.OutOfStockItem, error)

	// GetReservedAnalysis retrieves reserved stock analysis
	GetReservedAnalysis(ctx context.Context) (*model.ReservedStockAnalysis, error)

	// GetMovementTrends retrieves movement trends (last 7-30 days)
	GetMovementTrends(ctx context.Context, days int) ([]model.MovementTrend, error)
}
