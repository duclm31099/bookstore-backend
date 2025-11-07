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
func (r *postgresRepository) Create(ctx context.Context, inventory *model.Inventory) error {
	query := `
		INSERT INTO inventories (
			id, book_id, warehouse_location, quantity, reserved_quantity,
			low_stock_threshold, version, last_restock_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING available_quantity, is_low_stock
	`

	err := r.pool.QueryRow(ctx, query,
		inventory.ID,
		inventory.BookID,
		inventory.WarehouseLocation,
		inventory.Quantity,
		inventory.ReservedQuantity,
		inventory.LowStockThreshold,
		inventory.Version,
		inventory.LastRestockAt,
		inventory.UpdatedAt,
	).Scan(&inventory.AvailableQuantity, &inventory.IsLowStock)

	if err != nil {
		// Handle unique constraint violation (duplicate book_id + warehouse_location)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return model.ErrInventoryAlreadyExists
			}
			if pgErr.Code == "23503" { // foreign_key_violation
				return bookModel.ErrBookNotFound
			}
		}
		return fmt.Errorf("failed to insert inventory: %w", err)
	}

	return nil
}

// CreateBatch implements Repository.CreateBatch using pgx CopyFrom
func (r *postgresRepository) CreateBatch(ctx context.Context, inventories []model.Inventory) error {
	// Prepare data for CopyFrom
	columns := []string{
		"id", "book_id", "warehouse_location", "quantity", "reserved_quantity",
		"low_stock_threshold", "version", "last_restock_at", "updated_at",
	}

	rows := make([][]interface{}, len(inventories))
	for i, inv := range inventories {
		rows[i] = []interface{}{
			inv.ID,
			inv.BookID,
			inv.WarehouseLocation,
			inv.Quantity,
			inv.ReservedQuantity,
			inv.LowStockThreshold,
			inv.Version,
			inv.LastRestockAt,
			inv.UpdatedAt,
		}
	}

	// Execute batch insert using COPY protocol
	copyCount, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"inventories"},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return model.ErrInventoryAlreadyExists
			}
			if pgErr.Code == "23503" { // foreign_key_violation
				return model.ErrBookNotFound
			}
		}
		return fmt.Errorf("failed to batch insert inventories: %w", err)
	}

	if copyCount != int64(len(inventories)) {
		return fmt.Errorf("expected to insert %d rows, but inserted %d", len(inventories), copyCount)
	}

	// Note: CopyFrom doesn't support RETURNING clause, so generated columns
	// (available_quantity, is_low_stock) won't be populated automatically.
	// They will be calculated by PostgreSQL triggers/constraints.
	// If you need these values immediately, fetch them after insert.

	return nil
}

// GetByID implements Repository.GetByID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Inventory, error) {
	query := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE id = $1
	`

	var inventory model.Inventory
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inventory.ID,
		&inventory.BookID,
		&inventory.WarehouseLocation,
		&inventory.Quantity,
		&inventory.ReservedQuantity,
		&inventory.AvailableQuantity,
		&inventory.LowStockThreshold,
		&inventory.IsLowStock,
		&inventory.Version,
		&inventory.LastRestockAt,
		&inventory.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewInventoryNotFoundError(id)
		}
		return nil, fmt.Errorf("failed to get inventory by id: %w", err)
	}

	return &inventory, nil
}

// GetByBookAndWarehouse implements Repository.GetByBookAndWarehouse
func (r *postgresRepository) GetByBookAndWarehouse(ctx context.Context, bookID uuid.UUID, warehouse string) (*model.Inventory, error) {
	query := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE book_id = $1 AND warehouse_location = $2
	`

	var inventory model.Inventory
	err := r.pool.QueryRow(ctx, query, bookID, warehouse).Scan(
		&inventory.ID,
		&inventory.BookID,
		&inventory.WarehouseLocation,
		&inventory.Quantity,
		&inventory.ReservedQuantity,
		&inventory.AvailableQuantity,
		&inventory.LowStockThreshold,
		&inventory.IsLowStock,
		&inventory.Version,
		&inventory.LastRestockAt,
		&inventory.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewInventoryNotFoundByBookError(bookID, warehouse)
		}
		return nil, fmt.Errorf("failed to get inventory by book and warehouse: %w", err)
	}

	return &inventory, nil
}

// List implements Repository.List
func (r *postgresRepository) List(ctx context.Context, filter model.ListInventoryRequest) ([]model.Inventory, int, error) {
	// Build dynamic query with filters
	queryBuilder := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM inventories WHERE 1=1"

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filter.BookID != nil {
		queryBuilder += fmt.Sprintf(" AND book_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND book_id = $%d", argCount)
		args = append(args, *filter.BookID)
		argCount++
	}

	if filter.WarehouseLocation != nil {
		queryBuilder += fmt.Sprintf(" AND warehouse_location = $%d", argCount)
		countQuery += fmt.Sprintf(" AND warehouse_location = $%d", argCount)
		args = append(args, *filter.WarehouseLocation)
		argCount++
	}

	if filter.IsLowStock != nil {
		queryBuilder += fmt.Sprintf(" AND is_low_stock = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_low_stock = $%d", argCount)
		args = append(args, *filter.IsLowStock)
		argCount++
	}

	// Get total count
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count inventories: %w", err)
	}

	// Add ordering, pagination
	queryBuilder += " ORDER BY updated_at DESC, id ASC" // Consistent ordering
	offset := (filter.Page - 1) * filter.Limit
	queryBuilder += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list inventories: %w", err)
	}
	defer rows.Close()

	// Scan results
	inventories := make([]model.Inventory, 0, filter.Limit)
	for rows.Next() {
		var inv model.Inventory
		err := rows.Scan(
			&inv.ID,
			&inv.BookID,
			&inv.WarehouseLocation,
			&inv.Quantity,
			&inv.ReservedQuantity,
			&inv.AvailableQuantity,
			&inv.LowStockThreshold,
			&inv.IsLowStock,
			&inv.Version,
			&inv.LastRestockAt,
			&inv.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan inventory row: %w", err)
		}
		inventories = append(inventories, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, totalCount, nil
}

// ExistsByBookID implements Repository.ExistsByBookID
func (r *postgresRepository) ExistsByBookID(ctx context.Context, bookID uuid.UUID) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM inventories WHERE book_id = $1)"

	var exists bool
	err := r.pool.QueryRow(ctx, query, bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check inventory existence: %w", err)
	}

	return exists, nil
}

// Update implements Repository.Update with optimistic locking
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, inventory *model.Inventory) error {
	query := `
		UPDATE inventories
		SET 
			quantity = $2,
			reserved_quantity = $3,
			low_stock_threshold = $4,
			last_restock_at = $5,
			version = version + 1,  -- Increment version atomically
			updated_at = NOW()
		WHERE id = $1 AND version = $6  -- Check version for optimistic locking
		RETURNING 
			available_quantity, 
			is_low_stock, 
			version, 
			updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		id,
		inventory.Quantity,
		inventory.ReservedQuantity,
		inventory.LowStockThreshold,
		inventory.LastRestockAt,
		inventory.Version, // Current version from request
	).Scan(
		&inventory.AvailableQuantity,
		&inventory.IsLowStock,
		&inventory.Version, // New version after increment
		&inventory.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows affected means either:
			// 1. Record doesn't exist
			// 2. Version mismatch (optimistic lock failed)

			// Check if record exists
			var exists bool
			checkQuery := "SELECT EXISTS(SELECT 1 FROM inventories WHERE id = $1)"
			checkErr := r.pool.QueryRow(ctx, checkQuery, id).Scan(&exists)

			if checkErr != nil {
				return fmt.Errorf("failed to check inventory existence: %w", checkErr)
			}

			if !exists {
				return model.NewInventoryNotFoundError(id)
			}

			// Record exists but version mismatch - optimistic lock failed
			return model.ErrOptimisticLockFailed
		}
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	return nil
}

// Delete implements Repository.Delete
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM inventories
		WHERE id = $1
		  AND quantity = 0
		  AND reserved_quantity = 0
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	// Check if row was actually deleted
	if result.RowsAffected() == 0 {
		// Check if record exists to provide better error message
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM inventories WHERE id = $1)"
		checkErr := r.pool.QueryRow(ctx, checkQuery, id).Scan(&exists)

		if checkErr != nil {
			return fmt.Errorf("failed to check inventory existence: %w", checkErr)
		}

		if !exists {
			return model.NewInventoryNotFoundError(id)
		}

		// Record exists but has stock, validation should catch this
		// but if it reaches here, something went wrong
		return model.ErrCannotDeleteNonEmptyInventory
	}

	return nil
}

// ReserveStock implements Repository.ReserveStock with transaction and row-level locking
func (r *postgresRepository) ReserveStock(
	ctx context.Context,
	bookID uuid.UUID,
	warehouse string,
	quantity int,
	referenceType string,
	referenceID uuid.UUID,
) (*model.Inventory, error) {
	// Start transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Auto-rollback if not committed

	// Lock row and check available stock (FOR UPDATE prevents concurrent modifications)
	checkQuery := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE book_id = $1 AND warehouse_location = $2
		FOR UPDATE  -- Pessimistic lock for concurrent safety
	`

	var inventory model.Inventory
	err = tx.QueryRow(ctx, checkQuery, bookID, warehouse).Scan(
		&inventory.ID,
		&inventory.BookID,
		&inventory.WarehouseLocation,
		&inventory.Quantity,
		&inventory.ReservedQuantity,
		&inventory.AvailableQuantity,
		&inventory.LowStockThreshold,
		&inventory.IsLowStock,
		&inventory.Version,
		&inventory.LastRestockAt,
		&inventory.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewInventoryNotFoundByBookError(bookID, warehouse)
		}
		return nil, fmt.Errorf("failed to lock inventory: %w", err)
	}

	// Check if enough available stock
	if inventory.AvailableQuantity < quantity {
		return nil, model.NewInsufficientStockError(quantity, inventory.AvailableQuantity)
	}

	// Update reserved quantity
	updateQuery := `
		UPDATE inventories
		SET 
			reserved_quantity = reserved_quantity + $3,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND version = $2
		RETURNING 
			reserved_quantity,
			available_quantity,
			is_low_stock,
			version,
			updated_at
	`

	err = tx.QueryRow(ctx, updateQuery,
		inventory.ID,
		inventory.Version,
		quantity,
	).Scan(
		&inventory.ReservedQuantity,
		&inventory.AvailableQuantity,
		&inventory.IsLowStock,
		&inventory.Version,
		&inventory.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to reserve stock: %w", err)
	}

	// Create inventory movement record for audit trail
	movementQuery := `
		INSERT INTO inventory_movements (
			inventory_id, movement_type, quantity,
			quantity_before, quantity_after,
			reference_type, reference_id,
			notes, created_at
		) VALUES (
			$1, 'reserve', $2, $3, $4, $5, $6, $7, NOW()
		)
	`

	quantityBefore := inventory.ReservedQuantity - quantity
	_, err = tx.Exec(ctx, movementQuery,
		inventory.ID,
		quantity,
		quantityBefore,
		inventory.ReservedQuantity,
		referenceType,
		referenceID,
		fmt.Sprintf("Reserved %d units for %s", quantity, referenceType),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to log inventory movement: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &inventory, nil
}

// ReleaseStock implements Repository.ReleaseStock with transaction
func (r *postgresRepository) ReleaseStock(
	ctx context.Context,
	bookID uuid.UUID,
	warehouse string,
	quantity int,
	referenceID uuid.UUID,
) (*model.Inventory, error) {
	// Start transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock row and get current state
	checkQuery := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE book_id = $1 AND warehouse_location = $2
		FOR UPDATE
	`

	var inventory model.Inventory
	err = tx.QueryRow(ctx, checkQuery, bookID, warehouse).Scan(
		&inventory.ID,
		&inventory.BookID,
		&inventory.WarehouseLocation,
		&inventory.Quantity,
		&inventory.ReservedQuantity,
		&inventory.AvailableQuantity,
		&inventory.LowStockThreshold,
		&inventory.IsLowStock,
		&inventory.Version,
		&inventory.LastRestockAt,
		&inventory.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewInventoryNotFoundByBookError(bookID, warehouse)
		}
		return nil, fmt.Errorf("failed to lock inventory: %w", err)
	}

	// Check if enough reserved quantity to release
	if inventory.ReservedQuantity < quantity {
		return nil, fmt.Errorf("%w: trying to release %d, but only %d reserved",
			model.ErrInvalidReleaseQuantity, quantity, inventory.ReservedQuantity)
	}

	// Update reserved quantity
	updateQuery := `
		UPDATE inventories
		SET 
			reserved_quantity = reserved_quantity - $3,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND version = $2
		RETURNING 
			reserved_quantity,
			available_quantity,
			is_low_stock,
			version,
			updated_at
	`

	err = tx.QueryRow(ctx, updateQuery,
		inventory.ID,
		inventory.Version,
		quantity,
	).Scan(
		&inventory.ReservedQuantity,
		&inventory.AvailableQuantity,
		&inventory.IsLowStock,
		&inventory.Version,
		&inventory.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to release stock: %w", err)
	}

	// Create inventory movement record
	movementQuery := `
		INSERT INTO inventory_movements (
			inventory_id, movement_type, quantity,
			quantity_before, quantity_after,
			reference_type, reference_id,
			notes, created_at
		) VALUES (
			$1, 'release', $2, $3, $4, 'order', $5, $6, NOW()
		)
	`

	quantityBefore := inventory.ReservedQuantity + quantity
	_, err = tx.Exec(ctx, movementQuery,
		inventory.ID,
		-quantity, // Negative for release
		quantityBefore,
		inventory.ReservedQuantity,
		referenceID,
		fmt.Sprintf("Released %d reserved units", quantity),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to log inventory movement: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &inventory, nil
}

// GetInventoriesByBook implements Repository.GetInventoriesByBook
func (r *postgresRepository) GetInventoriesByBook(ctx context.Context, bookID uuid.UUID) ([]model.Inventory, error) {
	query := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE book_id = $1
		ORDER BY warehouse_location ASC
	`

	rows, err := r.pool.Query(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventories by book: %w", err)
	}
	defer rows.Close()

	inventories := make([]model.Inventory, 0, 4) // Expected 4 warehouses max
	for rows.Next() {
		var inv model.Inventory
		err := rows.Scan(
			&inv.ID,
			&inv.BookID,
			&inv.WarehouseLocation,
			&inv.Quantity,
			&inv.ReservedQuantity,
			&inv.AvailableQuantity,
			&inv.LowStockThreshold,
			&inv.IsLowStock,
			&inv.Version,
			&inv.LastRestockAt,
			&inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory: %w", err)
		}
		inventories = append(inventories, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return inventories, nil
}

// GetInventoriesByBooks implements Repository.GetInventoriesByBooks
// Optimized to fetch multiple books in single query
func (r *postgresRepository) GetInventoriesByBooks(ctx context.Context, bookIDs []uuid.UUID) (map[uuid.UUID][]model.Inventory, error) {
	if len(bookIDs) == 0 {
		return make(map[uuid.UUID][]model.Inventory), nil
	}

	query := `
		SELECT 
			id, book_id, warehouse_location, quantity, reserved_quantity,
			available_quantity, low_stock_threshold, is_low_stock,
			version, last_restock_at, updated_at
		FROM inventories
		WHERE book_id = ANY($1)
		ORDER BY book_id ASC, warehouse_location ASC
	`

	rows, err := r.pool.Query(ctx, query, bookIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventories by books: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.Inventory)
	for rows.Next() {
		var inv model.Inventory
		err := rows.Scan(
			&inv.ID,
			&inv.BookID,
			&inv.WarehouseLocation,
			&inv.Quantity,
			&inv.ReservedQuantity,
			&inv.AvailableQuantity,
			&inv.LowStockThreshold,
			&inv.IsLowStock,
			&inv.Version,
			&inv.LastRestockAt,
			&inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory: %w", err)
		}
		result[inv.BookID] = append(result[inv.BookID], inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory rows: %w", err)
	}

	return result, nil
}

// CreateMovement implements Repository.CreateMovement
func (r *postgresRepository) CreateMovement(ctx context.Context, movement *model.InventoryMovement) error {
	query := `
		INSERT INTO inventory_movements (
			id, inventory_id, movement_type, quantity,
			quantity_before, quantity_after,
			reference_type, reference_id,
			notes, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.pool.Exec(ctx, query,
		movement.ID,
		movement.InventoryID,
		movement.MovementType,
		movement.Quantity,
		movement.QuantityBefore,
		movement.QuantityAfter,
		movement.ReferenceType,
		movement.ReferenceID,
		movement.Notes,
		movement.CreatedBy,
		movement.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert movement: %w", err)
	}

	return nil
}

// ListMovements implements Repository.ListMovements
func (r *postgresRepository) ListMovements(ctx context.Context, filter model.ListMovementsRequest) ([]model.InventoryMovement, int, error) {
	queryBuilder := `
		SELECT 
			im.id, im.inventory_id, im.movement_type, im.quantity,
			im.quantity_before, im.quantity_after,
			im.reference_type, im.reference_id,
			im.notes, im.created_by, im.created_at
		FROM inventory_movements im
		WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM inventory_movements WHERE 1=1"

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filter.InventoryID != nil {
		queryBuilder += fmt.Sprintf(" AND im.inventory_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND inventory_id = $%d", argCount)
		args = append(args, *filter.InventoryID)
		argCount++
	}

	if filter.MovementType != nil {
		queryBuilder += fmt.Sprintf(" AND im.movement_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND movement_type = $%d", argCount)
		args = append(args, *filter.MovementType)
		argCount++
	}

	if filter.ReferenceType != nil {
		queryBuilder += fmt.Sprintf(" AND im.reference_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND reference_type = $%d", argCount)
		args = append(args, *filter.ReferenceType)
		argCount++
	}

	// Get total count
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count movements: %w", err)
	}

	// Add ordering and pagination
	queryBuilder += " ORDER BY im.created_at DESC, im.id ASC"
	offset := (filter.Page - 1) * filter.Limit
	queryBuilder += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, queryBuilder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list movements: %w", err)
	}
	defer rows.Close()

	movements := make([]model.InventoryMovement, 0, filter.Limit)
	for rows.Next() {
		var m model.InventoryMovement
		err := rows.Scan(
			&m.ID,
			&m.InventoryID,
			&m.MovementType,
			&m.Quantity,
			&m.QuantityBefore,
			&m.QuantityAfter,
			&m.ReferenceType,
			&m.ReferenceID,
			&m.Notes,
			&m.CreatedBy,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan movement: %w", err)
		}
		movements = append(movements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating movements: %w", err)
	}

	return movements, totalCount, nil
}

// GetMovementStatsForBook implements Repository.GetMovementStatsForBook
func (r *postgresRepository) GetMovementStatsForBook(ctx context.Context, bookID uuid.UUID) (*model.MovementStatsResponse, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN im.movement_type = 'inbound' THEN im.quantity ELSE 0 END), 0) as total_inbound,
			COALESCE(SUM(CASE WHEN im.movement_type = 'outbound' THEN im.quantity ELSE 0 END), 0) as total_outbound,
			COALESCE(SUM(CASE WHEN im.movement_type = 'reserve' THEN im.quantity ELSE 0 END), 0) as total_reserved,
			COALESCE(SUM(CASE WHEN im.movement_type = 'release' THEN im.quantity ELSE 0 END), 0) as total_released,
			MAX(im.created_at) as last_movement
		FROM inventory_movements im
		JOIN inventories i ON im.inventory_id = i.id
		WHERE i.book_id = $1
	`

	var totalInbound, totalOutbound, totalReserved, totalReleased int
	var lastMovement *time.Time

	err := r.pool.QueryRow(ctx, query, bookID).Scan(
		&totalInbound,
		&totalOutbound,
		&totalReserved,
		&totalReleased,
		&lastMovement,
	)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get movement stats: %w", err)
	}

	// Get by warehouse breakdown
	warehouseQuery := `
		SELECT 
			i.warehouse_location,
			COALESCE(SUM(CASE WHEN im.movement_type = 'inbound' THEN im.quantity ELSE 0 END), 0) as inbound
		FROM inventory_movements im
		JOIN inventories i ON im.inventory_id = i.id
		WHERE i.book_id = $1
		GROUP BY i.warehouse_location
		ORDER BY i.warehouse_location ASC
	`

	rows, err := r.pool.Query(ctx, warehouseQuery, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse breakdown: %w", err)
	}
	defer rows.Close()

	byWarehouse := make(map[string]int)
	for rows.Next() {
		var warehouse string
		var inbound int
		err := rows.Scan(&warehouse, &inbound)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse: %w", err)
		}
		byWarehouse[warehouse] = inbound
	}

	// Get by movement type breakdown
	typeQuery := `
		SELECT 
			im.movement_type,
			COUNT(*) as count
		FROM inventory_movements im
		JOIN inventories i ON im.inventory_id = i.id
		WHERE i.book_id = $1
		GROUP BY im.movement_type
		ORDER BY im.movement_type ASC
	`

	rows, err = r.pool.Query(ctx, typeQuery, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get type breakdown: %w", err)
	}
	defer rows.Close()

	byMovementType := make(map[string]int)
	for rows.Next() {
		var movementType string
		var count int
		err := rows.Scan(&movementType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan type: %w", err)
		}
		byMovementType[movementType] = count
	}

	netMovement := totalInbound - totalOutbound

	return &model.MovementStatsResponse{
		BookID:         bookID,
		TotalInbound:   totalInbound,
		TotalOutbound:  totalOutbound,
		TotalReserved:  totalReserved,
		TotalReleased:  totalReleased,
		NetMovement:    netMovement,
		ByWarehouse:    byWarehouse,
		ByMovementType: byMovementType,
		LastMovement:   lastMovement,
	}, nil
}

// GetDashboardMetrics implements Repository.GetDashboardMetrics
func (r *postgresRepository) GetDashboardMetrics(ctx context.Context) (*model.DashboardSummary, error) {
	query := `
		SELECT 
			COUNT(DISTINCT book_id) as total_books,
			COALESCE(SUM(quantity), 0) as total_quantity,
			COALESCE(SUM(reserved_quantity), 0) as total_reserved,
			COALESCE(SUM(available_quantity), 0) as total_available,
			SUM(CASE WHEN is_low_stock THEN 1 ELSE 0 END) as low_stock_count,
			SUM(CASE WHEN quantity = 0 THEN 1 ELSE 0 END) as out_of_stock_count
		FROM inventories
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

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	// Calculate health score (0-100)
	if summary.TotalQuantity > 0 {
		// Health = (available / total) * 100 - (low_stock_count / total_books) * 10
		availabilityScore := (float64(summary.TotalAvailable) / float64(summary.TotalQuantity)) * 100
		lowStockPenalty := (float64(summary.LowStockCount) / float64(summary.TotalBooks)) * 10
		summary.HealthScore = availabilityScore - lowStockPenalty
		if summary.HealthScore < 0 {
			summary.HealthScore = 0
		}
	}

	// Determine health status
	if summary.HealthScore >= 80 {
		summary.HealthStatus = "healthy"
	} else if summary.HealthScore >= 50 {
		summary.HealthStatus = "warning"
	} else {
		summary.HealthStatus = "critical"
	}

	return &summary, nil
}

// GetWarehouseMetrics implements Repository.GetWarehouseMetrics
func (r *postgresRepository) GetWarehouseMetrics(ctx context.Context) ([]model.WarehouseMetrics, error) {
	query := `
		SELECT 
			warehouse_location,
			COUNT(DISTINCT book_id) as book_count,
			COALESCE(SUM(quantity), 0) as total_quantity,
			COALESCE(SUM(reserved_quantity), 0) as total_reserved,
			COALESCE(SUM(available_quantity), 0) as total_available,
			SUM(CASE WHEN is_low_stock THEN 1 ELSE 0 END) as low_stock_count,
			SUM(CASE WHEN quantity = 0 THEN 1 ELSE 0 END) as out_of_stock_count,
			MAX(updated_at) as last_movement
		FROM inventories
		GROUP BY warehouse_location
		ORDER BY warehouse_location ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouse metrics: %w", err)
	}
	defer rows.Close()

	var metrics []model.WarehouseMetrics
	var totalQuantity int
	var totalReserved int

	for rows.Next() {
		var m model.WarehouseMetrics
		err := rows.Scan(
			&m.Warehouse,
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

		// Calculate utilization (mock: assume max capacity = total_quantity * 1.5)
		if m.TotalQuantity > 0 {
			m.Utilization = (float64(m.TotalQuantity) / float64(m.TotalQuantity*2)) * 100
		}

		// Calculate reservation rate
		if m.TotalQuantity > 0 {
			m.ReservationRate = (float64(m.TotalReserved) / float64(m.TotalQuantity)) * 100
		}

		// Calculate health score
		if m.TotalQuantity > 0 {
			availScore := (float64(m.TotalAvailable) / float64(m.TotalQuantity)) * 100
			lowStockPenalty := (float64(m.LowStockCount) / float64(m.BookCount)) * 10
			m.HealthScore = availScore - lowStockPenalty
			if m.HealthScore < 0 {
				m.HealthScore = 0
			}
		}

		totalQuantity += m.TotalQuantity
		totalReserved += m.TotalReserved
		metrics = append(metrics, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating warehouse metrics: %w", err)
	}

	return metrics, nil
}

// GetLowStockItems implements Repository.GetLowStockItems
func (r *postgresRepository) GetLowStockItems(ctx context.Context) ([]model.LowStockItem, error) {
	query := `
		SELECT 
			i.book_id,
			i.warehouse_location,
			i.quantity,
			i.reserved_quantity,
			i.available_quantity,
			i.low_stock_threshold
		FROM inventories i
		WHERE i.is_low_stock = true
		ORDER BY i.available_quantity ASC, i.warehouse_location ASC
		LIMIT 100
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query low stock items: %w", err)
	}
	defer rows.Close()

	var items []model.LowStockItem
	for rows.Next() {
		var item model.LowStockItem
		err := rows.Scan(
			&item.BookID,
			&item.WarehouseLocation,
			&item.CurrentStock,
			&item.ReservedStock,
			&item.AvailableStock,
			&item.LowStockThreshold,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan low stock item: %w", err)
		}

		// Calculate estimated days until stockout (mock: assume 10 units/day usage)
		if item.AvailableStock > 0 {
			item.DaysUntilStockout = item.AvailableStock / 10
		}

		// Recommend reorder quantity (3x low stock threshold)
		item.RecommendedReorder = item.LowStockThreshold * 3

		// Determine priority
		if item.AvailableStock == 0 {
			item.Priority = "critical"
		} else if item.AvailableStock <= item.LowStockThreshold/2 {
			item.Priority = "high"
		} else {
			item.Priority = "medium"
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating low stock items: %w", err)
	}

	return items, nil
}

// GetOutOfStockItems implements Repository.GetOutOfStockItems
func (r *postgresRepository) GetOutOfStockItems(ctx context.Context) ([]model.OutOfStockItem, error) {
	query := `
		SELECT 
			i.book_id,
			i.warehouse_location,
			i.reserved_quantity,
			i.last_restock_at
		FROM inventories i
		WHERE i.quantity = 0
		ORDER BY i.warehouse_location ASC
		LIMIT 100
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query out of stock items: %w", err)
	}
	defer rows.Close()

	var items []model.OutOfStockItem
	now := time.Now()
	for rows.Next() {
		var item model.OutOfStockItem
		err := rows.Scan(
			&item.BookID,
			&item.WarehouseLocation,
			&item.ReservedStock,
			&item.LastRestockDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan out of stock item: %w", err)
		}

		// Calculate days since stockout
		if item.LastRestockDate != nil {
			item.DaysSinceStockout = int(now.Sub(*item.LastRestockDate).Hours() / 24)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating out of stock items: %w", err)
	}

	return items, nil
}

// GetReservedAnalysis implements Repository.GetReservedAnalysis
func (r *postgresRepository) GetReservedAnalysis(ctx context.Context) (*model.ReservedStockAnalysis, error) {
	// Get summary
	summaryQuery := `
		SELECT 
			COALESCE(SUM(reserved_quantity), 0) as total_reserved,
			COALESCE(SUM(available_quantity), 0) as total_available
		FROM inventories
	`

	var analysis model.ReservedStockAnalysis
	err := r.pool.QueryRow(ctx, summaryQuery).Scan(
		&analysis.TotalReserved,
		&analysis.TotalAvailable,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get reserved summary: %w", err)
	}

	// Calculate reservation rate
	if analysis.TotalReserved+analysis.TotalAvailable > 0 {
		analysis.ReservationRate = (float64(analysis.TotalReserved) / float64(analysis.TotalReserved+analysis.TotalAvailable)) * 100
	}

	// Get by warehouse breakdown
	warehouseQuery := `
		SELECT 
			warehouse_location,
			COALESCE(SUM(reserved_quantity), 0) as reserved
		FROM inventories
		GROUP BY warehouse_location
		ORDER BY warehouse_location ASC
	`

	rows, err := r.pool.Query(ctx, warehouseQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query warehouse reserved: %w", err)
	}
	defer rows.Close()

	analysis.ByWarehouse = make(map[string]int)
	for rows.Next() {
		var warehouse string
		var reserved int
		err := rows.Scan(&warehouse, &reserved)
		if err != nil {
			return nil, fmt.Errorf("failed to scan warehouse reserved: %w", err)
		}
		analysis.ByWarehouse[warehouse] = reserved
	}

	return &analysis, nil
}

// GetMovementTrends implements Repository.GetMovementTrends
func (r *postgresRepository) GetMovementTrends(ctx context.Context, days int) ([]model.MovementTrend, error) {
	query := `
		SELECT 
			DATE(im.created_at) as period,
			COALESCE(SUM(CASE WHEN im.movement_type = 'inbound' THEN im.quantity ELSE 0 END), 0) as inbound,
			COALESCE(SUM(CASE WHEN im.movement_type = 'outbound' THEN ABS(im.quantity) ELSE 0 END), 0) as outbound,
			COUNT(*) as transaction_count
		FROM inventory_movements im
		WHERE im.created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(im.created_at)
		ORDER BY period DESC
	`

	rows, err := r.pool.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query movement trends: %w", err)
	}
	defer rows.Close()

	var trends []model.MovementTrend
	for rows.Next() {
		var trend model.MovementTrend
		err := rows.Scan(
			&trend.Period,
			&trend.Inbound,
			&trend.Outbound,
			&trend.TransactionCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}
		trend.NetMovement = trend.Inbound - trend.Outbound
		trends = append(trends, trend)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trends: %w", err)
	}

	return trends, nil
}
