package repository

import (
	"bookstore-backend/internal/domains/warehouse/model"
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// CRUD admin
	CreateWarehouse(ctx context.Context, req model.CreateWarehouseRequest) (*model.Warehouse, error)
	UpdateWarehouse(ctx context.Context, id uuid.UUID, req model.UpdateWarehouseRequest) (*model.Warehouse, error)
	SoftDeleteWarehouse(ctx context.Context, id uuid.UUID) error
	GetWarehouseByID(ctx context.Context, id uuid.UUID) (*model.Warehouse, error)
	GetWarehouseByCode(ctx context.Context, code string) (*model.Warehouse, error)
	ListWarehouses(ctx context.Context, filter model.ListWarehouseFilter) ([]model.Warehouse, error)
	// Advanced filter + paging tuỳ logic filter

	// Validate không còn inventory trước khi xóa
	HasInventory(ctx context.Context, warehouseID uuid.UUID) (bool, error)

	// Public lookup
	FindWarehousesWithStockByDistance(ctx context.Context, bookID uuid.UUID, lat float64, long float64, requiredQty int) ([]model.WarehouseWithInventory, error)
	ListActiveWarehouses(ctx context.Context) ([]model.Warehouse, error)
}
