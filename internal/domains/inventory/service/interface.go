package service

import (
	"bookstore-backend/internal/domains/inventory/model"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ServiceInterface defines business logic for multi-warehouse inventory management
type ServiceInterface interface {
	// ========================================
	// INVENTORY MANAGEMENT
	// ========================================

	// CreateInventory creates inventory for one or all warehouses
	// If req.WarehouseID = nil && req.CreateForAllWarehouses = true:
	//   - Query all active warehouses
	//   - Create inventory for each warehouse with same initial stock
	// Otherwise: Create for specified warehouse only
	// Validates:
	//   - Book exists (FK constraint)
	//   - Warehouse exists and is active
	//   - No duplicate (warehouse_id, book_id)
	// Returns list of created inventory records
	CreateInventory(ctx context.Context, req model.CreateInventoryRequest) ([]model.InventoryResponse, error)

	// GetInventoryByWarehouseAndBook retrieves specific inventory
	// Returns ErrInventoryNotFound if not exists
	GetInventoryByWarehouseAndBook(ctx context.Context, warehouseID, bookID uuid.UUID) (*model.InventoryResponse, error)

	// UpdateInventory updates inventory with optimistic locking
	// Only updates non-nil fields in request
	// Validates:
	//   - version matches (optimistic lock)
	//   - quantity >= reserved (constraint)
	//   - reserved >= 0, quantity >= 0
	// Returns ErrOptimisticLockFailed if concurrent update
	UpdateInventory(ctx context.Context, warehouseID, bookID uuid.UUID, req model.UpdateInventoryRequest) (*model.InventoryResponse, error)

	// DeleteInventory soft deletes inventory
	// Validates quantity = 0 AND reserved = 0
	// Returns ErrCannotDeleteNonEmptyInventory otherwise
	DeleteInventory(ctx context.Context, warehouseID, bookID uuid.UUID) error

	// ListInventories paginated list with filters
	// Filters: book_id, warehouse_id, is_low_stock, has_available_stock
	// Includes warehouse name via join
	ListInventories(ctx context.Context, req model.ListInventoryRequest) (*model.ListInventoryResponse, error)

	// ========================================
	// STOCK RESERVATION & CHECKOUT (FR-INV-003)
	// ========================================

	// ReserveStock reserves stock for order checkout
	// Flow:
	//   1. If warehouse_id not specified: auto-select nearest warehouse
	//   2. Call reserve_stock() function with pessimistic lock
	//   3. Return reservation details
	// Use case: User clicks "Checkout" button
	// Timeout: 15 minutes (background job will auto-release)
	// Returns ErrInsufficientStock if not enough available
	ReserveStock(ctx context.Context, req model.ReserveStockRequest) (*model.ReserveStockResponse, error)

	// ReleaseStock releases reserved stock
	// Use cases:
	//   - Payment timeout (15m)
	//   - User cancels order
	//   - Payment failed
	// Validates: cannot release more than currently reserved
	ReleaseStock(ctx context.Context, req model.ReleaseStockRequest) (*model.ReleaseStockResponse, error)

	// CompleteSale completes sale after payment success
	// Decreases both quantity and reserved
	// Creates audit log with action = 'SALE'
	CompleteSale(ctx context.Context, req model.CompleteSaleRequest) (*model.CompleteSaleResponse, error)

	// ========================================
	// WAREHOUSE SELECTION (FR-INV-002)
	// ========================================

	// FindOptimalWarehouse finds best warehouse for customer
	// Algorithm:
	//   1. Calculate distance from customer address to all warehouses (Haversine)
	//   2. Filter warehouses with sufficient available stock
	//   3. Return nearest warehouse
	// Fallback: If no warehouse within 50km, return nearest nationally
	FindOptimalWarehouse(ctx context.Context, req model.FindWarehouseRequest) (*model.WarehouseRecommendation, error)

	// CheckAvailability checks if order can be fulfilled
	// For each item in request:
	//   - Check available stock across all warehouses
	//   - Recommend warehouse split if needed
	// Returns:
	//   - Per-item fulfillability
	//   - Recommended warehouse allocations
	//   - Overall can_fulfill status
	// Does NOT reserve stock (read-only operation)
	CheckAvailability(ctx context.Context, req model.CheckAvailabilityRequest) (*model.CheckAvailabilityResponse, error)

	// GetStockSummary gets total stock for a book across all warehouses
	// Uses books_total_stock VIEW
	// Returns warehouse breakdown and total available
	GetStockSummary(ctx context.Context, bookID uuid.UUID) (*model.StockSummaryResponse, error)

	// ========================================
	// STOCK ADJUSTMENT (FR-INV-005)
	// ========================================

	// AdjustStock manually adjusts inventory (admin only)
	// Use cases:
	//   - Periodic stock count reconciliation
	//   - Damaged goods write-off
	//   - Lost inventory adjustment
	// Creates audit log with:
	//   - action = 'ADJUSTMENT'
	//   - reason (required)
	//   - changed_by (admin user)
	//   - IP address
	// Validates: new quantity >= reserved
	AdjustStock(ctx context.Context, req model.AdjustStockRequest) (*model.AdjustStockResponse, error)

	// RestockInventory adds new stock (restock from supplier)
	// Increases quantity
	// Updates last_restocked_at timestamp
	// Creates audit log with action = 'RESTOCK'
	RestockInventory(ctx context.Context, req model.RestockRequest) (*model.RestockResponse, error)

	// BulkUpdateStock imports stock updates from CSV (FR-INV-006)
	// Validates CSV format:
	//   - warehouse_code, isbn, quantity_to_add, reason
	// Returns job_id for async processing
	// Background job processes CSV rows with validation
	BulkUpdateStock(ctx context.Context, csvPath string, uploadedBy uuid.UUID) (*model.BulkUpdateJobResponse, error)

	// GetBulkUpdateStatus checks import job status
	// Returns: processing/completed/failed, progress, errors
	GetBulkUpdateStatus(ctx context.Context, jobID uuid.UUID) (*model.BulkUpdateStatusResponse, error)

	// ========================================
	// ALERTS & NOTIFICATIONS (FR-INV-004)
	// ========================================

	// GetLowStockAlerts gets unresolved low stock alerts
	// Trigger auto-creates alerts when quantity < alert_threshold
	// Trigger auto-resolves when restocked
	// Returns prioritized list (critical > high > medium)
	GetLowStockAlerts(ctx context.Context) ([]model.LowStockAlert, error)

	// GetOutOfStockItems gets all out-of-stock items
	// quantity = 0 across all warehouses
	// Includes days_since_stockout calculation
	GetOutOfStockItems(ctx context.Context) ([]model.OutOfStockItem, error)

	// MarkAlertResolved manually marks alert as resolved (admin)
	// Normally auto-resolved by trigger
	MarkAlertResolved(ctx context.Context, alertID uuid.UUID) error

	// ========================================
	// AUDIT & REPORTING (FR-INV-005)
	// ========================================

	// GetAuditTrail retrieves audit log with filters
	// Filters: warehouse, book, date range, action type, user
	// Supports pagination
	// Use case: Compliance reporting, investigation
	GetAuditTrail(ctx context.Context, req model.AuditTrailRequest) (*model.AuditTrailResponse, error)

	// GetInventoryHistory gets full history for specific book+warehouse
	// Shows all movements: restock, reserve, release, sale, adjustment
	// Ordered by created_at DESC
	GetInventoryHistory(ctx context.Context, warehouseID, bookID uuid.UUID, limit, offset int) (*model.InventoryHistoryResponse, error)

	// ExportAuditLog exports audit log to CSV for compliance
	// Date range required (max 1 year)
	// Returns CSV file path or download URL
	ExportAuditLog(ctx context.Context, req model.ExportAuditRequest) (*model.ExportResponse, error)

	// ========================================
	// DASHBOARD & ANALYTICS
	// ========================================

	// GetDashboardSummary gets comprehensive dashboard
	// Includes:
	//   - Overall metrics (total books, stock, reserved)
	//   - Health score (0-100)
	//   - Low stock count, out of stock count
	//   - Per-warehouse breakdown
	//   - Recent movements
	GetDashboardSummary(ctx context.Context) (*model.DashboardSummaryResponse, error)

	// GetWarehousePerformance gets warehouse-specific metrics
	// - Stock levels, utilization rate
	// - Reservation rate (reserved/total)
	// - Movement trends (last 30 days)
	// - Low stock items count
	GetWarehousePerformance(ctx context.Context, warehouseID uuid.UUID) (*model.WarehousePerformanceResponse, error)

	// // GetInventoryValue calculates total inventory value
	// // Requires join with books table for pricing
	// // Breakdown by warehouse and category
	// // Use case: Financial reporting
	// GetInventoryValue(ctx context.Context) (*model.InventoryValueResponse, error)

	// // GetMovementTrends gets movement trends over time
	// // Aggregates by day/week/month
	// // Includes: inbound (restock), outbound (sales), net movement
	// // Use case: Forecasting, trend analysis
	// GetMovementTrends(ctx context.Context, days int) ([]model.MovementTrend, error)

	// GetReservationAnalysis analyzes reservation patterns
	// - Total reserved across system
	// - Reservation rate by warehouse
	// - Average reservation duration
	// - Conversion rate (reserved → sale)
	GetReservationAnalysis(ctx context.Context) (*model.ReservationAnalysisResponse, error)

	// ========================================
	// WAREHOUSE MANAGEMENT
	// ========================================

	// CreateWarehouse creates new warehouse (admin only)
	// Validates:
	//   - Unique code
	//   - Valid coordinates (latitude, longitude)
	//   - Province in Vietnam
	CreateWarehouse(ctx context.Context, req model.CreateWarehouseRequest) (*model.WarehouseResponse, error)

	// UpdateWarehouse updates warehouse details
	// Can update: name, address, coordinates, is_active
	// Uses optimistic locking (version column)
	UpdateWarehouse(ctx context.Context, warehouseID uuid.UUID, req model.UpdateWarehouseRequest) (*model.WarehouseResponse, error)

	// ListWarehouses lists all warehouses with filters
	// Filters: is_active, province
	ListWarehouses(ctx context.Context, req model.ListWarehousesRequest) ([]model.WarehouseResponse, error)

	// GetWarehouseByID retrieves warehouse by ID
	GetWarehouseByID(ctx context.Context, warehouseID uuid.UUID) (*model.WarehouseResponse, error)

	// DeactivateWarehouse soft-deletes warehouse (sets deleted_at)
	// Validates: no active inventory (all quantity = 0)
	// Historical orders still reference this warehouse
	DeactivateWarehouse(ctx context.Context, warehouseID uuid.UUID) error

	// ReserveStockWithTx reserves stock within an existing transaction
	// Used by Order service to ensure atomic operations
	ReserveStockWithTx(
		ctx context.Context,
		tx pgx.Tx,
		warehouseID uuid.UUID,
		bookID uuid.UUID,
		quantity int,
		userid *uuid.UUID,
	) error

	// ReleaseStockWithTx releases reserved stock within an existing transaction
	// Used by Order service when cancelling orders
	ReleaseStockWithTx(
		ctx context.Context,
		tx pgx.Tx,
		warehouseID uuid.UUID,
		bookID uuid.UUID,
		quantity int,
		userid *uuid.UUID,
	) error

	// ============================================
	// QUERY METHODS (read-only, no transactions)
	// ============================================

	// CheckAvailableStock checks if enough stock is available
	// Returns true if (quantity - reserved) >= requiredQty
	CheckAvailableStock(
		ctx context.Context,
		warehouseID uuid.UUID,
		bookID uuid.UUID,
		requiredQty int,
	) (bool, error)

	// GetAvailableQuantity returns the available quantity for a book at a warehouse
	GetAvailableQuantity(
		ctx context.Context,
		warehouseID uuid.UUID,
		bookID uuid.UUID,
	) (int, error)
}
