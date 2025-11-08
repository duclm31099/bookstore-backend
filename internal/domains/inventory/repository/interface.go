package repository

import (
	"bookstore-backend/internal/domains/inventory/model"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RepositoryInterface defines the contract for inventory data access
// Tương tác trực tiếp với schema warehouse_inventory mới
type RepositoryInterface interface {
	// ========================================
	// CORE CRUD OPERATIONS
	// ========================================

	// Create inserts a new inventory record for one warehouse + one book
	// PRIMARY KEY: (warehouse_id, book_id)
	// Returns ErrInventoryAlreadyExists if duplicate exists
	// Returns ErrBookNotFound or ErrWarehouseNotFound if FK violation
	Create(ctx context.Context, inventory *model.Inventory) error

	// CreateBatch inserts multiple inventory records in single transaction
	// Used for bulk import from CSV (FR-INV-006)
	// All-or-nothing: rollback if any row fails
	CreateBatch(ctx context.Context, inventories []model.Inventory) error

	// GetByWarehouseAndBook retrieves inventory by composite primary key
	// Returns ErrInventoryNotFound if not exists
	GetByWarehouseAndBook(ctx context.Context, warehouseID, bookID uuid.UUID) (*model.Inventory, error)

	// Update updates inventory with optimistic locking
	// Uses version column to prevent concurrent modification (FR-INV-001)
	// Returns ErrOptimisticLockFailed if version mismatch
	// Returns ErrInventoryNotFound if not exists
	// Trigger tự động tạo audit log entry
	Update(ctx context.Context, warehouseID, bookID uuid.UUID, inventory *model.Inventory) error

	// Delete removes inventory record
	// Only allowed if quantity = 0 AND reserved = 0
	// Returns ErrCannotDeleteNonEmptyInventory if validation fails
	Delete(ctx context.Context, warehouseID, bookID uuid.UUID) error

	// List retrieves paginated inventory records with filters
	// Supports filters: book_id, warehouse_id, is_low_stock, has_available_stock
	// Joins with warehouses table to get warehouse name
	List(ctx context.Context, filter model.ListInventoryRequest) ([]model.Inventory, int, error)

	// ========================================
	// STOCK RESERVATION (FR-INV-003)
	// ========================================

	// ReserveStock calls DB function reserve_stock()
	// Uses pessimistic locking (SELECT FOR UPDATE NOWAIT)
	// Atomically increases reserved quantity
	// Returns updated inventory after reservation
	// Returns ErrInsufficientStock if not enough available
	// Returns ErrOptimisticLockFailed if concurrent modification
	ReserveStock(ctx context.Context, warehouseID, bookID uuid.UUID, quantity int, userID *uuid.UUID) (*model.Inventory, error)

	// ReleaseStock calls DB function release_stock()
	// Atomically decreases reserved quantity
	// Used when order cancelled/expired (timeout 15m)
	// Returns updated inventory
	ReleaseStock(ctx context.Context, warehouseID, bookID uuid.UUID, quantity int, userID *uuid.UUID) (*model.Inventory, error)

	// CompleteSale calls DB function complete_sale()
	// Decreases both quantity AND reserved
	// Used when payment success
	// Returns updated inventory
	CompleteSale(ctx context.Context, warehouseID, bookID uuid.UUID, quantity int, userID *uuid.UUID) (*model.Inventory, error)

	// ========================================
	// WAREHOUSE SELECTION (FR-INV-002)
	// ========================================

	// FindNearestWarehouse calls DB function find_nearest_warehouse()
	// Uses Haversine formula to calculate distance
	// Returns warehouse nearest to customer address with sufficient stock
	// Returns error if no warehouse has enough stock
	FindNearestWarehouse(ctx context.Context, bookID uuid.UUID, latitude, longitude float64, requiredQuantity int) (*model.NearestWarehouse, error)

	// GetInventoriesByBook retrieves all warehouse inventories for a book
	// Sorted by available stock DESC (most stock first)
	// Only includes active warehouses (is_active = true, deleted_at IS NULL)
	GetInventoriesByBook(ctx context.Context, bookID uuid.UUID) ([]model.Inventory, error)

	// GetTotalStockForBook queries VIEW books_total_stock
	// Aggregates total quantity, reserved, available across all warehouses
	// Returns 0 values if book has no inventory
	GetTotalStockForBook(ctx context.Context, bookID uuid.UUID) (*model.TotalStockResponse, error)

	// ========================================
	// LOW STOCK ALERTS (FR-INV-004)
	// ========================================

	// GetLowStockAlerts queries low_stock_alerts table
	// Filters by is_resolved status
	// Trigger tự động tạo alerts khi quantity < alert_threshold
	// Trigger tự động resolve khi restocked
	GetLowStockAlerts(ctx context.Context, resolved bool) ([]model.LowStockAlert, error)

	// ========================================
	// AUDIT TRAIL (FR-INV-005)
	// ========================================

	// GetAuditLog queries partitioned table inventory_audit_log
	// Supports filters: warehouse_id, book_id, date range
	// Ordered by created_at DESC
	// Trigger tự động tạo log entries khi inventory thay đổi
	GetAuditLog(ctx context.Context, warehouseID, bookID *uuid.UUID, startDate, endDate *time.Time, limit, offset int) ([]model.AuditLogEntry, int, error)

	// ========================================
	// DASHBOARD & ANALYTICS
	// ========================================

	// GetDashboardMetrics returns aggregated metrics
	// - Total books, quantity, reserved, available
	// - Low stock count, out of stock count
	// - Health score calculation
	GetDashboardMetrics(ctx context.Context) (*model.DashboardSummary, error)

	// GetWarehouseMetrics returns per-warehouse breakdown
	// - Book count, quantity, reserved per warehouse
	// - Utilization rate, reservation rate
	// - Health score per warehouse
	GetWarehouseMetrics(ctx context.Context) ([]model.WarehouseMetrics, error)

	// ... (các methods hiện tại)

	// ========================================
	// WAREHOUSE MANAGEMENT (BỔ SUNG)
	// ========================================

	// ListWarehouses retrieves all warehouses with filters
	// Filters: is_active, province
	// Sorted by name ASC
	ListWarehouses(ctx context.Context, req model.ListWarehousesRequest) ([]model.Warehouse, error)

	// GetWarehouseByID retrieves warehouse by ID
	// Returns ErrWarehouseNotFound if not exists or deleted
	GetWarehouseByID(ctx context.Context, warehouseID uuid.UUID) (*model.Warehouse, error)

	// CreateWarehouse creates new warehouse
	// Returns ErrWarehouseCodeExists if code already exists
	CreateWarehouse(ctx context.Context, warehouse *model.Warehouse) error

	// UpdateWarehouse updates warehouse with optimistic locking
	// Returns ErrOptimisticLockFailed if version mismatch
	UpdateWarehouse(ctx context.Context, warehouseID uuid.UUID, warehouse *model.Warehouse) error

	// DeactivateWarehouse soft deletes warehouse (sets deleted_at)
	// Validates: all inventory quantities = 0 before deletion
	DeactivateWarehouse(ctx context.Context, warehouseID uuid.UUID) error

	// ========================================
	// ANALYTICS & METRICS (BỔ SUNG)
	// ========================================

	// GetMovementTrends returns daily movement trends
	// Aggregates inbound/outbound movements over specified days
	// Used for dashboard charts and forecasting
	GetMovementTrends(ctx context.Context, days int) ([]model.MovementTrend, error)

	// GetReservationMetrics returns reservation-specific metrics
	// - Total reserved, reservation rate
	// - Average reservation duration
	// - Breakdown by warehouse
	GetReservationMetrics(ctx context.Context) (*model.ReservationMetrics, error)

	// ReserveStockWithTx reserves stock using provided transaction
	ReserveStockWithTx(
		ctx context.Context, tx pgx.Tx,
		warehouseID uuid.UUID, bookID uuid.UUID, quantity int, userid *uuid.UUID,
	) error

	// ReleaseStockWithTx releases stock using provided transaction
	ReleaseStockWithTx(ctx context.Context, tx pgx.Tx,
		warehouseID uuid.UUID, bookID uuid.UUID, quantity int, userid *uuid.UUID) error
	// GetAvailableQuantity returns available quantity (quantity - reserved)
	GetAvailableQuantity(ctx context.Context, warehouseID uuid.UUID, bookID uuid.UUID) (int, error)
}
