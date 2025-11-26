package repository

import (
	bookModel "bookstore-backend/internal/domains/book/model"
	"bookstore-backend/internal/domains/inventory/model"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresRepository implements Repository interface
type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(pool *pgxpool.Pool) RepositoryInterface {
	return &postgresRepository{
		pool: pool,
	}
}

// Create implements Repository.Create
// Tạo inventory record cho 1 warehouse + 1 book
func (r *postgresRepository) Create(ctx context.Context, inventory *model.Inventory) error {
	query := `
		INSERT INTO warehouse_inventory (
			warehouse_id, book_id, quantity, reserved,
			alert_threshold, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		RETURNING updated_at, version
	`

	err := r.pool.QueryRow(ctx, query,
		inventory.WarehouseID,
		inventory.BookID,
		inventory.Quantity,
		inventory.Reserved,
		inventory.AlertThreshold,
		inventory.UpdatedBy,
	).Scan(&inventory.UpdatedAt, &inventory.Version)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation on (warehouse_id, book_id)
				return model.ErrInventoryAlreadyExists
			}
			if pgErr.Code == "23503" { // foreign_key_violation
				if pgErr.ConstraintName == "warehouse_inventory_book_id_fkey" {
					return bookModel.ErrBookNotFound
				}
				return fmt.Errorf("warehouse not found")
			}
		}
		return fmt.Errorf("failed to insert inventory: %w", err)
	}

	// Trigger sẽ tự động tạo audit log và low stock alert nếu cần
	return nil
}

// CreateBatch implements Repository.CreateBatch
func (r *postgresRepository) CreateBatch(ctx context.Context, inventories []model.Inventory) error {
	// Sử dụng transaction để batch insert
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare batch insert
	batch := &pgx.Batch{}
	query := `
		INSERT INTO warehouse_inventory (
			warehouse_id, book_id, quantity, reserved,
			alert_threshold, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, inv := range inventories {
		batch.Queue(query,
			inv.WarehouseID,
			inv.BookID,
			inv.Quantity,
			inv.Reserved,
			inv.AlertThreshold,
			inv.UpdatedBy,
		)
	}

	// Send batch
	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	// Check all results
	for i := 0; i < len(inventories); i++ {
		_, err := br.Exec()
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code == "23505" {
					return model.ErrInventoryAlreadyExists
				}
				if pgErr.Code == "23503" {
					return fmt.Errorf("foreign key violation at row %d: %w", i, err)
				}
			}
			return fmt.Errorf("failed to insert row %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

// GetByID - XÓA METHOD NÀY vì schema mới không có id riêng
// Thay bằng GetByWarehouseAndBook

// GetByWarehouseAndBook implements Repository.GetByWarehouseAndBook
func (r *postgresRepository) GetByWarehouseAndBook(ctx context.Context, warehouseID, bookID uuid.UUID) (*model.Inventory, error) {
	query := `
		SELECT 
			warehouse_id, book_id, quantity, reserved,
			alert_threshold, version, last_restocked_at, updated_at, updated_by
		FROM warehouse_inventory
		WHERE warehouse_id = $1 AND book_id = $2
	`

	var inventory model.Inventory
	err := r.pool.QueryRow(ctx, query, warehouseID, bookID).Scan(
		&inventory.WarehouseID,
		&inventory.BookID,
		&inventory.Quantity,
		&inventory.Reserved,
		&inventory.AlertThreshold,
		&inventory.Version,
		&inventory.LastRestockAt,
		&inventory.UpdatedAt,
		&inventory.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewInventoryNotFoundByBookError(bookID, warehouseID.String())
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Calculate available quantity (không còn column riêng)
	inventory.AvailableQuantity = inventory.Quantity - inventory.Reserved

	return &inventory, nil
}

// List implements Repository.List
func (r *postgresRepository) List(ctx context.Context, filter model.ListInventoryRequest) ([]model.Inventory, int, error) {
	queryBuilder := `
		SELECT 
			wi.warehouse_id, wi.book_id, wi.quantity, wi.reserved,
			wi.alert_threshold, wi.version, wi.last_restocked_at, 
			wi.updated_at, wi.updated_by,
			w.name as warehouse_name  -- Join để lấy tên kho
		FROM warehouse_inventory wi
		INNER JOIN warehouses w ON wi.warehouse_id = w.id
		WHERE 1=1
	`
	countQuery := `
		SELECT COUNT(*) 
		FROM warehouse_inventory wi
		INNER JOIN warehouses w ON wi.warehouse_id = w.id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filter.BookID != nil {
		queryBuilder += fmt.Sprintf(" AND wi.book_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND wi.book_id = $%d", argCount)
		args = append(args, *filter.BookID)
		argCount++
	}

	// Filter by low stock (quantity < alert_threshold)
	if filter.IsLowStock != nil && *filter.IsLowStock {
		queryBuilder += " AND wi.quantity < wi.alert_threshold"
		countQuery += " AND wi.quantity < wi.alert_threshold"
	}

	// Get total count
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count inventories: %w", err)
	}

	// Add ordering, pagination
	queryBuilder += " ORDER BY wi.updated_at DESC, wi.warehouse_id, wi.book_id"
	offset := (filter.Page - 1) * filter.Limit
	queryBuilder += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list inventories: %w", err)
	}
	defer rows.Close()

	inventories := make([]model.Inventory, 0, filter.Limit)
	for rows.Next() {
		var inv model.Inventory
		var warehouseName string
		err := rows.Scan(
			&inv.WarehouseID,
			&inv.BookID,
			&inv.Quantity,
			&inv.Reserved,
			&inv.AlertThreshold,
			&inv.Version,
			&inv.LastRestockAt,
			&inv.UpdatedAt,
			&inv.UpdatedBy,
			&warehouseName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inv.AvailableQuantity = inv.Quantity - inv.Reserved
		inv.WarehouseName = warehouseName // Map warehouse name
		inventories = append(inventories, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, totalCount, nil
}

// Update implements Repository.Update with optimistic locking
func (r *postgresRepository) Update(ctx context.Context, warehouseID, bookID uuid.UUID, inventory *model.Inventory) error {
	query := `
		UPDATE warehouse_inventory
		SET 
			quantity = $3,
			reserved = $4,
			alert_threshold = $5,
			last_restocked_at = $6,
			version = version + 1,
			updated_by = $7,
			updated_at = NOW()
		WHERE warehouse_id = $1 
		  AND book_id = $2 
		  AND version = $8  -- Optimistic lock check
		RETURNING version, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		warehouseID,
		bookID,
		inventory.Quantity,
		inventory.Reserved,
		inventory.AlertThreshold,
		inventory.LastRestockAt,
		inventory.UpdatedBy,
		inventory.Version, // Current version
	).Scan(&inventory.Version, &inventory.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if exists
			var exists bool
			checkQuery := "SELECT EXISTS(SELECT 1 FROM warehouse_inventory WHERE warehouse_id = $1 AND book_id = $2)"
			checkErr := r.pool.QueryRow(ctx, checkQuery, warehouseID, bookID).Scan(&exists)

			if checkErr != nil {
				return fmt.Errorf("failed to check inventory existence: %w", checkErr)
			}

			if !exists {
				return model.NewInventoryNotFoundByBookError(bookID, warehouseID.String())
			}

			// Exists but version mismatch
			return model.ErrOptimisticLockFailed
		}
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	// Trigger tự động tạo audit log
	return nil
}

// Delete implements Repository.Delete
func (r *postgresRepository) Delete(ctx context.Context, warehouseID, bookID uuid.UUID) error {
	// Only allow delete if no stock and no reservations
	query := `
		DELETE FROM warehouse_inventory
		WHERE warehouse_id = $1 
		  AND book_id = $2
		  AND quantity = 0
		  AND reserved = 0
	`

	result, err := r.pool.Exec(ctx, query, warehouseID, bookID)
	if err != nil {
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	if result.RowsAffected() == 0 {
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM warehouse_inventory WHERE warehouse_id = $1 AND book_id = $2)"
		checkErr := r.pool.QueryRow(ctx, checkQuery, warehouseID, bookID).Scan(&exists)

		if checkErr != nil {
			return fmt.Errorf("failed to check inventory existence: %w", checkErr)
		}

		if !exists {
			return model.NewInventoryNotFoundByBookError(bookID, warehouseID.String())
		}

		return model.ErrCannotDeleteNonEmptyInventory
	}

	return nil
}

// ReserveStock - SỬ DỤNG DB FUNCTION thay vì tự implement
func (r *postgresRepository) ReserveStock(
	ctx context.Context,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userID *uuid.UUID,
) (*model.Inventory, error) {
	// Gọi function reserve_stock từ database
	query := `
		SELECT reserve_stock($1, $2, $3, $4)
	`

	var success bool
	err := r.pool.QueryRow(ctx, query, warehouseID, bookID, quantity, userID).Scan(&success)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Custom error code BIZ01 = insufficient stock
			if pgErr.Code == "BIZ01" {
				return nil, model.NewInsufficientStockError(quantity, 0) // Available sẽ ở trong message
			}
			// 40001 = serialization_failure (concurrent modification)
			if pgErr.Code == "40001" {
				return nil, model.ErrOptimisticLockFailed
			}
		}
		// Check for "does not exist" error
		if errors.Is(err, pgx.ErrNoRows) || (err != nil && errors.As(err, &pgErr) && pgErr.Message == "Inventory record not found") {
			return nil, model.NewInventoryNotFoundByBookError(bookID, warehouseID.String())
		}
		return nil, fmt.Errorf("failed to reserve stock: %w", err)
	}

	// Fetch updated inventory
	return r.GetByWarehouseAndBook(ctx, warehouseID, bookID)
}

// ReleaseStock - SỬ DỤNG DB FUNCTION
func (r *postgresRepository) ReleaseStock(
	ctx context.Context,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userID *uuid.UUID,
) (*model.Inventory, error) {
	query := `SELECT release_stock($1, $2, $3, $4)`

	var success bool
	err := r.pool.QueryRow(ctx, query, warehouseID, bookID, quantity, userID).Scan(&success)

	if err != nil {
		return nil, fmt.Errorf("failed to release stock: %w", err)
	}

	return r.GetByWarehouseAndBook(ctx, warehouseID, bookID)
}

// CompleteSale - SỬ DỤNG DB FUNCTION (giảm cả quantity và reserved)
func (r *postgresRepository) CompleteSale(
	ctx context.Context,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userID *uuid.UUID,
) (*model.Inventory, error) {
	query := `SELECT complete_sale($1, $2, $3, $4)`

	var success bool
	err := r.pool.QueryRow(ctx, query, warehouseID, bookID, quantity, userID).Scan(&success)

	if err != nil {
		return nil, fmt.Errorf("failed to complete sale: %w", err)
	}

	return r.GetByWarehouseAndBook(ctx, warehouseID, bookID)
}

// FindNearestWarehouse - SỬ DỤNG DB FUNCTION
func (r *postgresRepository) FindNearestWarehouse(
	ctx context.Context,
	bookID uuid.UUID,
	latitude, longitude float64,
	requiredQuantity int,
) (*model.NearestWarehouse, error) {
	query := `
		SELECT warehouse_id, warehouse_name, available_quantity, distance_km
		FROM find_nearest_warehouse($1, $2, $3, $4)
	`

	var result model.NearestWarehouse
	err := r.pool.QueryRow(ctx, query, bookID, latitude, longitude, requiredQuantity).Scan(
		&result.WarehouseID,
		&result.WarehouseName,
		&result.AvailableQuantity,
		&result.DistanceKM,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no warehouse found with sufficient stock")
		}
		return nil, fmt.Errorf("failed to find nearest warehouse: %w", err)
	}

	result.BookID = bookID
	return &result, nil
}

// GetInventoriesByBook - Query tất cả warehouses có stock cho 1 book
func (r *postgresRepository) GetInventoriesByBook(ctx context.Context, bookID uuid.UUID) ([]model.Inventory, error) {
	query := `
		SELECT 
			wi.warehouse_id, wi.book_id, wi.quantity, wi.reserved,
			wi.alert_threshold, wi.version, wi.last_restocked_at, 
			wi.updated_at, wi.updated_by,
			w.name as warehouse_name,
			w.province
		FROM warehouse_inventory wi
		INNER JOIN warehouses w ON wi.warehouse_id = w.id
		WHERE wi.book_id = $1
		  AND w.is_active = true
		  AND w.deleted_at IS NULL
		ORDER BY (wi.quantity - wi.reserved) DESC  -- Kho có stock nhiều nhất lên đầu
	`

	rows, err := r.pool.Query(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventories by book: %w", err)
	}
	defer rows.Close()

	inventories := make([]model.Inventory, 0)
	for rows.Next() {
		var inv model.Inventory
		var warehouseName, province string
		err := rows.Scan(
			&inv.WarehouseID,
			&inv.BookID,
			&inv.Quantity,
			&inv.Reserved,
			&inv.AlertThreshold,
			&inv.Version,
			&inv.LastRestockAt,
			&inv.UpdatedAt,
			&inv.UpdatedBy,
			&warehouseName,
			&province,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory: %w", err)
		}
		inv.AvailableQuantity = inv.Quantity - inv.Reserved
		inv.WarehouseName = fmt.Sprintf("%s (%s)", warehouseName, province)
		inventories = append(inventories, inv)
	}

	return inventories, nil
}

// GetTotalStockForBook - Sử dụng VIEW books_total_stock
func (r *postgresRepository) GetTotalStockForBook(ctx context.Context, bookID uuid.UUID) (*model.TotalStockResponse, error) {
	query := `
		SELECT 
			book_id,
			total_quantity,
			total_reserved,
			available,
			warehouse_count
		FROM books_total_stock
		WHERE book_id = $1
	`

	var result model.TotalStockResponse
	err := r.pool.QueryRow(ctx, query, bookID).Scan(
		&result.BookID,
		&result.TotalQuantity,
		&result.TotalReserved,
		&result.TotalAvailable,
		&result.WarehouseCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Book exists but no inventory
			return &model.TotalStockResponse{
				BookID:         bookID,
				TotalQuantity:  0,
				TotalReserved:  0,
				TotalAvailable: 0,
				WarehouseCount: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get total stock: %w", err)
	}

	return &result, nil
}

// GetLowStockAlerts - Query bảng low_stock_alerts
func (r *postgresRepository) GetLowStockAlerts(ctx context.Context, resolved bool) ([]model.LowStockAlert, error) {
	query := `
		SELECT 
			lsa.id,
			lsa.warehouse_id,
			lsa.book_id,
			lsa.current_quantity,
			lsa.alert_threshold,
			lsa.is_resolved,
			lsa.resolved_at,
			lsa.created_at,
			w.name as warehouse_name
		FROM low_stock_alerts lsa
		INNER JOIN warehouses w ON lsa.warehouse_id = w.id
		WHERE lsa.is_resolved = $1
		ORDER BY lsa.created_at DESC
		LIMIT 100
	`

	rows, err := r.pool.Query(ctx, query, resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to query low stock alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]model.LowStockAlert, 0)
	for rows.Next() {
		var alert model.LowStockAlert
		err := rows.Scan(
			&alert.ID,
			&alert.WarehouseID,
			&alert.BookID,
			&alert.CurrentQuantity,
			&alert.AlertThreshold,
			&alert.IsResolved,
			&alert.ResolvedAt,
			&alert.CreatedAt,
			&alert.WarehouseName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetAuditLog - Query inventory_audit_log với time range
func (r *postgresRepository) GetAuditLog(
	ctx context.Context,
	warehouseID *uuid.UUID,
	bookID *uuid.UUID,
	startDate, endDate *time.Time,
	limit, offset int,
) ([]model.AuditLogEntry, int, error) {
	queryBuilder := `
		SELECT 
			id, warehouse_id, book_id, action,
			old_quantity, new_quantity, old_reserved, new_reserved,
			quantity_change, reason, changed_by, ip_address, created_at
		FROM inventory_audit_log
		WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM inventory_audit_log WHERE 1=1"

	args := []interface{}{}
	argCount := 1

	if warehouseID != nil {
		queryBuilder += fmt.Sprintf(" AND warehouse_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND warehouse_id = $%d", argCount)
		args = append(args, *warehouseID)
		argCount++
	}

	if bookID != nil {
		queryBuilder += fmt.Sprintf(" AND book_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND book_id = $%d", argCount)
		args = append(args, *bookID)
		argCount++
	}

	if startDate != nil {
		queryBuilder += fmt.Sprintf(" AND created_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *startDate)
		argCount++
	}

	if endDate != nil {
		queryBuilder += fmt.Sprintf(" AND created_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *endDate)
		argCount++
	}

	// Count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Query with pagination
	queryBuilder += " ORDER BY created_at DESC"
	queryBuilder += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	logs := make([]model.AuditLogEntry, 0)
	for rows.Next() {
		var log model.AuditLogEntry
		err := rows.Scan(
			&log.ID,
			&log.WarehouseID,
			&log.BookID,
			&log.Action,
			&log.OldQuantity,
			&log.NewQuantity,
			&log.OldReserved,
			&log.NewReserved,
			&log.QuantityChange,
			&log.Reason,
			&log.ChangedBy,
			&log.IPAddress,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// GetDashboardMetrics - Aggregate metrics
func (r *postgresRepository) GetDashboardMetrics(ctx context.Context) (*model.DashboardSummary, error) {
	query := `
		SELECT 
			COUNT(DISTINCT book_id) as total_books,
			COALESCE(SUM(quantity), 0) as total_quantity,
			COALESCE(SUM(reserved), 0) as total_reserved,
			COALESCE(SUM(quantity - reserved), 0) as total_available,
			COUNT(*) FILTER (WHERE quantity < alert_threshold) as low_stock_count,
			COUNT(*) FILTER (WHERE quantity = 0) as out_of_stock_count
		FROM warehouse_inventory
	`

	var summary model.DashboardSummary
	err := r.pool.QueryRow(ctx, query).Scan(
		&summary.TotalBooks,
		&summary.TotalQuantity,
		&summary.TotalReserved,
		&summary.TotalAvailable,
		&summary.LowStockCount,
		&summary.OutOfStockCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	// Calculate health score
	if summary.TotalQuantity > 0 {
		availabilityScore := (float64(summary.TotalAvailable) / float64(summary.TotalQuantity)) * 100
		lowStockPenalty := float64(summary.LowStockCount) * 2
		summary.HealthScore = availabilityScore - lowStockPenalty
		if summary.HealthScore < 0 {
			summary.HealthScore = 0
		}
	}

	if summary.HealthScore >= 80 {
		summary.HealthStatus = "healthy"
	} else if summary.HealthScore >= 50 {
		summary.HealthStatus = "warning"
	} else {
		summary.HealthStatus = "critical"
	}

	return &summary, nil
}

// ========================================
// WAREHOUSE MANAGEMENT IMPLEMENTATION
// ========================================

// ListWarehouses implements Repository.ListWarehouses
func (r *postgresRepository) ListWarehouses(ctx context.Context, req model.ListWarehousesRequest) ([]model.Warehouse, error) {
	queryBuilder := `
		SELECT 
			id, name, code, address, province, 
			latitude, longitude, is_active, version,
			created_at, updated_at, deleted_at
		FROM warehouses
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if req.IsActive != nil {
		queryBuilder += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *req.IsActive)
		argCount++
	}

	if req.Province != nil {
		queryBuilder += fmt.Sprintf(" AND province = $%d", argCount)
		args = append(args, *req.Province)
		argCount++
	}

	// Order by name
	queryBuilder += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w", err)
	}
	defer rows.Close()

	warehouses := make([]model.Warehouse, 0)
	for rows.Next() {
		var wh model.Warehouse
		err := rows.Scan(
			&wh.ID,
			&wh.Name,
			&wh.Code,
			&wh.Address,
			&wh.Province,
			&wh.Latitude,
			&wh.Longitude,
			&wh.IsActive,
			&wh.Version,
			&wh.CreatedAt,
			&wh.UpdatedAt,
			&wh.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse: %w", err)
		}
		warehouses = append(warehouses, wh)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouses: %w", err)
	}

	return warehouses, nil
}

// GetWarehouseByID implements Repository.GetWarehouseByID
func (r *postgresRepository) GetWarehouseByID(ctx context.Context, warehouseID uuid.UUID) (*model.Warehouse, error) {
	query := `
		SELECT 
			id, name, code, address, province,
			latitude, longitude, is_active, version,
			created_at, updated_at, deleted_at
		FROM warehouses
		WHERE id = $1 AND deleted_at IS NULL
	`

	var wh model.Warehouse
	err := r.pool.QueryRow(ctx, query, warehouseID).Scan(
		&wh.ID,
		&wh.Name,
		&wh.Code,
		&wh.Address,
		&wh.Province,
		&wh.Latitude,
		&wh.Longitude,
		&wh.IsActive,
		&wh.Version,
		&wh.CreatedAt,
		&wh.UpdatedAt,
		&wh.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewWarehouseNotFoundError(warehouseID)
		}
		return nil, fmt.Errorf("failed to get warehouse: %w", err)
	}

	return &wh, nil
}

// CreateWarehouse implements Repository.CreateWarehouse
func (r *postgresRepository) CreateWarehouse(ctx context.Context, warehouse *model.Warehouse) error {
	query := `
		INSERT INTO warehouses (
			id, name, code, address, province,
			latitude, longitude, is_active, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING created_at, updated_at
	`

	// Generate ID if not provided
	if warehouse.ID == uuid.Nil {
		warehouse.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		warehouse.ID,
		warehouse.Name,
		warehouse.Code,
		warehouse.Address,
		warehouse.Province,
		warehouse.Latitude,
		warehouse.Longitude,
		warehouse.IsActive,
		1, // Initial version
	).Scan(&warehouse.CreatedAt, &warehouse.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				if pgErr.ConstraintName == "warehouses_code_key" {
					return model.ErrWarehouseCodeExists
				}
			}
		}
		return fmt.Errorf("failed to create warehouse: %w", err)
	}

	warehouse.Version = 1
	return nil
}

// UpdateWarehouse implements Repository.UpdateWarehouse
func (r *postgresRepository) UpdateWarehouse(ctx context.Context, warehouseID uuid.UUID, warehouse *model.Warehouse) error {
	query := `
		UPDATE warehouses
		SET 
			name = $2,
			address = $3,
			province = $4,
			latitude = $5,
			longitude = $6,
			is_active = $7,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 
		  AND version = $8
		  AND deleted_at IS NULL
		RETURNING version, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		warehouseID,
		warehouse.Name,
		warehouse.Address,
		warehouse.Province,
		warehouse.Latitude,
		warehouse.Longitude,
		warehouse.IsActive,
		warehouse.Version, // Current version for optimistic lock
	).Scan(&warehouse.Version, &warehouse.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if exists
			var exists bool
			checkQuery := "SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1 AND deleted_at IS NULL)"
			checkErr := r.pool.QueryRow(ctx, checkQuery, warehouseID).Scan(&exists)

			if checkErr != nil {
				return fmt.Errorf("failed to check warehouse existence: %w", checkErr)
			}

			if !exists {
				return model.NewWarehouseNotFoundError(warehouseID)
			}

			return model.ErrOptimisticLockFailed
		}
		return fmt.Errorf("failed to update warehouse: %w", err)
	}

	return nil
}

// DeactivateWarehouse implements Repository.DeactivateWarehouse
func (r *postgresRepository) DeactivateWarehouse(ctx context.Context, warehouseID uuid.UUID) error {
	// Check if warehouse has any inventory with quantity > 0
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 
			FROM warehouse_inventory 
			WHERE warehouse_id = $1 
			  AND quantity > 0
		)
	`

	var hasStock bool
	err := r.pool.QueryRow(ctx, checkQuery, warehouseID).Scan(&hasStock)
	if err != nil {
		return fmt.Errorf("failed to check warehouse inventory: %w", err)
	}

	if hasStock {
		return model.ErrCannotDeleteWarehouseWithStock
	}

	// Soft delete (set deleted_at)
	query := `
		UPDATE warehouses
		SET deleted_at = NOW(),
		    is_active = false,
		    updated_at = NOW()
		WHERE id = $1 
		  AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, warehouseID)
	if err != nil {
		return fmt.Errorf("failed to deactivate warehouse: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.NewWarehouseNotFoundError(warehouseID)
	}

	return nil
}

// ========================================
// ANALYTICS & METRICS IMPLEMENTATION
// ========================================

// GetWarehouseMetrics implements Repository.GetWarehouseMetrics
func (r *postgresRepository) GetWarehouseMetrics(ctx context.Context) ([]model.WarehouseMetrics, error) {
	query := `
		SELECT 
			w.id,
			w.name,
			COUNT(DISTINCT wi.book_id) as book_count,
			COALESCE(SUM(wi.quantity), 0) as total_quantity,
			COALESCE(SUM(wi.reserved), 0) as total_reserved,
			COALESCE(SUM(wi.quantity - wi.reserved), 0) as total_available,
			COUNT(*) FILTER (WHERE wi.quantity < wi.alert_threshold) as low_stock_count,
			COUNT(*) FILTER (WHERE wi.quantity = 0) as out_of_stock_count,
			MAX(wi.updated_at) as last_movement
		FROM warehouses w
		LEFT JOIN warehouse_inventory wi ON w.id = wi.warehouse_id
		WHERE w.deleted_at IS NULL
		  AND w.is_active = true
		GROUP BY w.id, w.name
		ORDER BY w.name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouse metrics: %w", err)
	}
	defer rows.Close()

	metrics := make([]model.WarehouseMetrics, 0)

	for rows.Next() {
		var m model.WarehouseMetrics
		err := rows.Scan(
			&m.WarehouseID,
			&m.WarehouseName,
			&m.BookCount,
			&m.TotalQuantity,
			&m.TotalReserved,
			&m.TotalAvailable,
			&m.LowStockCount,
			&m.OutOfStockCount,
			&m.LastMovement,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse metric: %w", err)
		}

		// Calculate utilization (assuming max capacity = 2x current quantity)
		if m.TotalQuantity > 0 {
			m.Utilization = (float64(m.TotalQuantity) / float64(m.TotalQuantity*2)) * 100
		}

		// Calculate reservation rate
		if m.TotalQuantity > 0 {
			m.ReservationRate = (float64(m.TotalReserved) / float64(m.TotalQuantity)) * 100
		}

		// Calculate health score
		// Formula: (available/quantity)*100 - (low_stock_count/book_count)*10
		if m.TotalQuantity > 0 && m.BookCount > 0 {
			availabilityScore := (float64(m.TotalAvailable) / float64(m.TotalQuantity)) * 100
			lowStockPenalty := (float64(m.LowStockCount) / float64(m.BookCount)) * 10
			m.HealthScore = availabilityScore - lowStockPenalty
			if m.HealthScore < 0 {
				m.HealthScore = 0
			}
		}

		metrics = append(metrics, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse metrics: %w", err)
	}

	return metrics, nil
}

// GetMovementTrends implements Repository.GetMovementTrends
func (r *postgresRepository) GetMovementTrends(ctx context.Context, days int) ([]model.MovementTrend, error) {
	query := `
		SELECT 
			DATE(created_at) as period,
			COALESCE(SUM(CASE 
				WHEN action IN ('RESTOCK') THEN quantity_change 
				ELSE 0 
			END), 0) as inbound,
			COALESCE(SUM(CASE 
				WHEN action IN ('SALE') THEN ABS(quantity_change)
				ELSE 0 
			END), 0) as outbound,
			COUNT(*) as transaction_count
		FROM inventory_audit_log
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(created_at)
		ORDER BY period DESC
	`

	rows, err := r.pool.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query movement trends: %w", err)
	}
	defer rows.Close()

	trends := make([]model.MovementTrend, 0)
	for rows.Next() {
		var trend model.MovementTrend
		var period time.Time
		err := rows.Scan(
			&period,
			&trend.Inbound,
			&trend.Outbound,
			&trend.TransactionCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}

		trend.Period = period.Format("2006-01-02")
		trend.NetMovement = trend.Inbound - trend.Outbound

		trends = append(trends, trend)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trends: %w", err)
	}

	return trends, nil
}

// GetReservationMetrics implements Repository.GetReservationMetrics
func (r *postgresRepository) GetReservationMetrics(ctx context.Context) (*model.ReservationMetrics, error) {
	// Summary query
	summaryQuery := `
		SELECT 
			COALESCE(SUM(reserved), 0) as total_reserved,
			COALESCE(SUM(quantity), 0) as total_quantity
		FROM warehouse_inventory
	`

	var metrics model.ReservationMetrics
	var totalQuantity int

	err := r.pool.QueryRow(ctx, summaryQuery).Scan(&metrics.TotalReserved, &totalQuantity)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get reservation summary: %w", err)
	}

	// Calculate reservation rate
	if totalQuantity > 0 {
		metrics.ReservationRate = (float64(metrics.TotalReserved) / float64(totalQuantity)) * 100
	}

	// Get by warehouse breakdown
	warehouseQuery := `
		SELECT 
			w.name,
			COALESCE(SUM(wi.reserved), 0) as reserved
		FROM warehouse_inventory wi
		INNER JOIN warehouses w ON wi.warehouse_id = w.id
		WHERE w.deleted_at IS NULL
		GROUP BY w.name
		ORDER BY w.name ASC
	`

	rows, err := r.pool.Query(ctx, warehouseQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouse reservations: %w", err)
	}
	defer rows.Close()

	metrics.ByWarehouse = make(map[string]int)
	for rows.Next() {
		var warehouseName string
		var reserved int
		err := rows.Scan(&warehouseName, &reserved)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse reservation: %w", err)
		}
		metrics.ByWarehouse[warehouseName] = reserved
	}

	// Calculate average reservation duration from audit log
	durationQuery := `
		SELECT 
			AVG(EXTRACT(EPOCH FROM (release.created_at - reserve.created_at)) / 60) as avg_minutes
		FROM inventory_audit_log reserve
		INNER JOIN inventory_audit_log release 
			ON reserve.warehouse_id = release.warehouse_id
			AND reserve.book_id = release.book_id
			AND reserve.action = 'RESERVE'
			AND release.action = 'RELEASE'
			AND release.created_at > reserve.created_at
			AND release.created_at <= reserve.created_at + INTERVAL '1 hour'
		WHERE reserve.created_at >= NOW() - INTERVAL '7 days'
	`

	var avgMinutes *float64
	err = r.pool.QueryRow(ctx, durationQuery).Scan(&avgMinutes)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to calculate avg duration: %w", err)
	}

	if avgMinutes != nil {
		metrics.AvgDurationMinutes = int(*avgMinutes)
	}

	// Calculate conversion rate (reserved → sale)
	conversionQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE action = 'RESERVE') as total_reserves,
			COUNT(*) FILTER (WHERE action = 'SALE') as total_sales
		FROM inventory_audit_log
		WHERE created_at >= NOW() - INTERVAL '7 days'
	`

	var totalReserves, totalSales int
	err = r.pool.QueryRow(ctx, conversionQuery).Scan(&totalReserves, &totalSales)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to calculate conversion rate: %w", err)
	}

	if totalReserves > 0 {
		metrics.ConversionRate = (float64(totalSales) / float64(totalReserves)) * 100
	}

	return &metrics, nil
}
func (r *postgresRepository) ReserveStockWithTx(
	ctx context.Context,
	tx pgx.Tx,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userID *uuid.UUID,
) error {
	query := `SELECT reserve_stock($1, $2, $3, $4)`

	var success bool
	err := tx.QueryRow(ctx, query, warehouseID, bookID, quantity, userID).Scan(&success)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Custom error code BIZ01 = insufficient stock
			if pgErr.Code == "BIZ01" {
				return model.NewInsufficientStockError(quantity, 0) // Available sẽ ở trong message
			}
			// 40001 = serialization_failure (concurrent modification)
			if pgErr.Code == "40001" {
				return model.ErrOptimisticLockFailed
			}
		}
		// Check for "does not exist" error
		if errors.Is(err, pgx.ErrNoRows) || (err != nil && errors.As(err, &pgErr) && pgErr.Message == "Inventory record not found") {
			return model.NewInventoryNotFoundByBookError(bookID, warehouseID.String())
		}
		return fmt.Errorf("failed to reserve stock: %w", err)
	}

	// Fetch updated inventory
	return nil
}

// ReleaseStockWithTx releases stock using provided transaction
func (r *postgresRepository) ReleaseStockWithTx(
	ctx context.Context,
	tx pgx.Tx,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userID *uuid.UUID,
) error {
	query := `SELECT release_stock($1, $2, $3, $4)`

	var success bool
	err := tx.QueryRow(ctx, query, warehouseID, bookID, quantity, userID).Scan(&success)

	if err != nil {
		return fmt.Errorf("failed to release stock: %w", err)
	}
	if !success {
		return fmt.Errorf("release_stock returned false for warehouse=%s, book=%s", warehouseID, bookID)
	}
	return nil
}

// GetAvailableQuantity returns available quantity for a book at a warehouse
func (r *postgresRepository) GetAvailableQuantity(
	ctx context.Context,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
) (int, error) {
	query := `
		SELECT available_quantity
		FROM inventories
		WHERE warehouse_id = $1 AND book_id = $2
	`

	var availableQty int
	err := r.pool.QueryRow(ctx, query, warehouseID, bookID).Scan(&availableQty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, model.NewInventoryNotFoundError(bookID)
		}
		return 0, fmt.Errorf("failed to get available quantity: %w", err)
	}

	return availableQty, nil
}

// GetBookTotalStock implements RepositoryInterface.
// Đọc tổng tồn cho 1 book từ view books_total_stock.
// Nếu không có row (book hiện không có inventory ở bất kỳ kho nào), trả về (nil, nil).
func (r *postgresRepository) GetBookTotalStock(ctx context.Context, bookID string) (*model.BookTotalStock, error) {
	query := `
        SELECT 
            book_id,
            total_quantity,
            total_reserved,
            available,
            warehouse_count,
            warehouses_with_stock
        FROM books_total_stock
        WHERE book_id = $1
    `

	row := r.pool.QueryRow(ctx, query, bookID)

	var result model.BookTotalStock
	var warehouses []string

	// warehouses_with_stock là mảng UUID → scan vào []string
	err := row.Scan(
		&result.BookID,
		&result.TotalQuantity,
		&result.TotalReserved,
		&result.Available,
		&result.WarehouseCount,
		&warehouses,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Không có stock record nào cho book này → để handler hiểu là stock = 0
			return nil, nil
		}
		return nil, fmt.Errorf("GetBookTotalStock query error: %w", err)
	}

	result.WarehousesWithStock = warehouses
	return &result, nil
}
