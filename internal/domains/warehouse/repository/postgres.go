package repository

import (
	"bookstore-backend/internal/domains/warehouse/model"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) CreateWarehouse(ctx context.Context, req model.CreateWarehouseRequest) (*model.Warehouse, error) {
	// Generate unique code (simple approach: uppercase name + timestamp, real deploy: use Postgres sequence for true unique code)
	code := fmt.Sprintf("WARE_%d", time.Now().UnixNano())
	query := `INSERT INTO warehouses (name, code, address, province, latitude, longitude, is_active)
    VALUES ($1, $2, $3, $4, $5, $6, TRUE)
    RETURNING id, version, created_at, updated_at`
	var warehouse model.Warehouse
	err := r.pool.QueryRow(ctx, query, req.Name, code, req.Address, req.Province, req.Latitude, req.Longitude).
		Scan(&warehouse.ID, &warehouse.Version, &warehouse.CreatedAt, &warehouse.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create warehouse: %w", err)
	}
	warehouse.Name = req.Name
	warehouse.Address = req.Address
	warehouse.Province = req.Province
	warehouse.Latitude = req.Latitude
	warehouse.Longitude = req.Longitude
	warehouse.Code = code
	warehouse.IsActive = true
	return &warehouse, nil
}

func (r *postgresRepository) UpdateWarehouse(ctx context.Context, id uuid.UUID, req model.UpdateWarehouseRequest) (*model.Warehouse, error) {
	setClauses := []string{}
	args := []interface{}{id}
	idx := 2
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name=$%d", idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Address != nil {
		setClauses = append(setClauses, fmt.Sprintf("address=$%d", idx))
		args = append(args, *req.Address)
		idx++
	}
	if req.Province != nil {
		setClauses = append(setClauses, fmt.Sprintf("province=$%d", idx))
		args = append(args, *req.Province)
		idx++
	}
	if req.Latitude != nil {
		setClauses = append(setClauses, fmt.Sprintf("latitude=$%d", idx))
		args = append(args, *req.Latitude)
		idx++
	}
	if req.Longitude != nil {
		setClauses = append(setClauses, fmt.Sprintf("longitude=$%d", idx))
		args = append(args, *req.Longitude)
		idx++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active=$%d", idx))
		args = append(args, *req.IsActive)
		idx++
	}
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no field to update")
	}
	setClause := strings.Join(setClauses, ", ")
	query := fmt.Sprintf(`UPDATE warehouses SET %s, updated_at=NOW(), version=version+1 WHERE id=$1 AND deleted_at IS NULL RETURNING name, code, address, province, latitude, longitude, is_active, version, created_at, updated_at`, setClause)
	var wh model.Warehouse
	wh.ID = id
	err := r.pool.QueryRow(ctx, query, args...).Scan(&wh.Name, &wh.Code, &wh.Address, &wh.Province, &wh.Latitude, &wh.Longitude, &wh.IsActive, &wh.Version, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update warehouse: %w", err)
	}
	return &wh, nil
}

func (r *postgresRepository) SoftDeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE warehouses SET deleted_at=NOW(), is_active=FALSE WHERE id=$1 AND deleted_at IS NULL`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete warehouse: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("warehouse not found or already deleted")
	}
	return nil
}

func (r *postgresRepository) GetWarehouseByID(ctx context.Context, id uuid.UUID) (*model.Warehouse, error) {
	query := `SELECT id, name, code, address, province, latitude, longitude, is_active, version, created_at, updated_at, deleted_at
            FROM warehouses WHERE id = $1 AND deleted_at IS NULL`
	var wh model.Warehouse
	err := r.pool.QueryRow(ctx, query, id).Scan(&wh.ID, &wh.Name, &wh.Code, &wh.Address, &wh.Province, &wh.Latitude, &wh.Longitude, &wh.IsActive, &wh.Version, &wh.CreatedAt, &wh.UpdatedAt, &wh.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("warehouse not found: %w", err)
	}
	return &wh, nil
}

func (r *postgresRepository) GetWarehouseByCode(ctx context.Context, code string) (*model.Warehouse, error) {
	query := `SELECT id, name, code, address, province, latitude, longitude, is_active, version, created_at, updated_at, deleted_at
            FROM warehouses WHERE code = $1 AND deleted_at IS NULL`
	var wh model.Warehouse
	err := r.pool.QueryRow(ctx, query, code).Scan(&wh.ID, &wh.Name, &wh.Code, &wh.Address, &wh.Province, &wh.Latitude, &wh.Longitude, &wh.IsActive, &wh.Version, &wh.CreatedAt, &wh.UpdatedAt, &wh.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("warehouse code not found: %w", err)
	}
	return &wh, nil
}

// List warehouses with filter + paging
func (r *postgresRepository) ListWarehouses(ctx context.Context, filter model.ListWarehouseFilter) ([]model.Warehouse, error) {
	where := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	idx := 1
	if filter.Keyword != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d OR address ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+filter.Keyword+"%")
		idx++
	}
	if filter.Province != "" {
		where = append(where, fmt.Sprintf("province = $%d", idx))
		args = append(args, filter.Province)
		idx++
	}
	if filter.IsActive != nil {
		where = append(where, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *filter.IsActive)
		idx++
	}
	whereStr := strings.Join(where, " AND ")
	query := fmt.Sprintf(`SELECT id, name, code, address, province, latitude, longitude, is_active, version, created_at, updated_at, deleted_at
 FROM warehouses WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereStr, idx, idx+1)
	args = append(args, filter.Limit)
	args = append(args, filter.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w", err)
	}
	defer rows.Close()
	var result []model.Warehouse
	for rows.Next() {
		var wh model.Warehouse
		err := rows.Scan(&wh.ID, &wh.Name, &wh.Code, &wh.Address, &wh.Province, &wh.Latitude, &wh.Longitude, &wh.IsActive, &wh.Version, &wh.CreatedAt, &wh.UpdatedAt, &wh.DeletedAt)
		if err != nil {
			continue
		}
		result = append(result, wh)
	}
	return result, nil
}

func (r *postgresRepository) HasInventory(ctx context.Context, warehouseID uuid.UUID) (bool, error) {
	query := `SELECT 1 FROM warehouse_inventory WHERE warehouse_id=$1 AND (quantity > 0 OR reserved > 0) LIMIT 1`
	row := r.pool.QueryRow(ctx, query, warehouseID)
	var dummy int
	if err := row.Scan(&dummy); err != nil {
		return false, nil // not found nghĩa là KHÔNG có tồn inventory
	}
	return true, nil
}

func (r *postgresRepository) FindWarehousesWithStockByDistance(ctx context.Context, bookID uuid.UUID, lat float64, lon float64, requiredQty int) ([]model.WarehouseWithInventory, error) {
	query := `
    SELECT w.id, w.name, w.code, w.address, w.province, w.latitude, w.longitude,
        w.is_active, w.version, w.created_at, w.updated_at, w.deleted_at,
        (wi.quantity - wi.reserved) AS available_quantity,
        (6371 * acos(
          cos(radians($2)) * cos(radians(w.latitude)) * cos(radians(w.longitude) - radians($3)) +
          sin(radians($2)) * sin(radians(w.latitude))
        )) AS distance_km
    FROM warehouses w
    INNER JOIN warehouse_inventory wi ON w.id = wi.warehouse_id
    WHERE w.is_active AND w.deleted_at IS NULL
      AND w.latitude IS NOT NULL AND w.longitude IS NOT NULL
      AND wi.book_id = $1
      AND (wi.quantity - wi.reserved) >= $4
    ORDER BY distance_km ASC`
	rows, err := r.pool.Query(ctx, query, bookID, lat, lon, requiredQty)
	if err != nil {
		return nil, fmt.Errorf("failed to find warehouse: %w", err)
	}
	defer rows.Close()
	var list []model.WarehouseWithInventory
	for rows.Next() {
		var w model.WarehouseWithInventory
		var deletedAt *time.Time
		err := rows.Scan(
			&w.ID, &w.Name, &w.Code, &w.Address, &w.Province,
			&w.Latitude, &w.Longitude, &w.IsActive, &w.Version, &w.CreatedAt, &w.UpdatedAt, &deletedAt,
			&w.AvailableQuantity, &w.DistanceKm,
		)
		if err != nil {
			continue
		}
		w.DeletedAt = deletedAt
		list = append(list, w)
	}
	return list, nil
}

func (r *postgresRepository) ListActiveWarehouses(ctx context.Context) ([]model.Warehouse, error) {
	query := `SELECT id, name, code, address, province, latitude, longitude, is_active, version, created_at, updated_at, deleted_at
            FROM warehouses WHERE is_active = TRUE AND deleted_at IS NULL`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active warehouses: %w", err)
	}
	defer rows.Close()
	var result []model.Warehouse
	for rows.Next() {
		var wh model.Warehouse
		err := rows.Scan(&wh.ID, &wh.Name, &wh.Code, &wh.Address, &wh.Province, &wh.Latitude, &wh.Longitude, &wh.IsActive, &wh.Version, &wh.CreatedAt, &wh.UpdatedAt, &wh.DeletedAt)
		if err != nil {
			continue
		}
		result = append(result, wh)
	}
	return result, nil
}
