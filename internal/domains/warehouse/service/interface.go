package service

import (
	"bookstore-backend/internal/domains/warehouse/model"
	"context"

	"github.com/google/uuid"
)

type Service interface {
	// CRUD
	CreateWarehouse(ctx context.Context, req model.CreateWarehouseRequest) (*model.Warehouse, error)
	UpdateWarehouse(ctx context.Context, id uuid.UUID, req model.UpdateWarehouseRequest) (*model.Warehouse, error)
	SoftDeleteWarehouse(ctx context.Context, id uuid.UUID) error
	GetWarehouseByID(ctx context.Context, id uuid.UUID) (*model.Warehouse, error)
	GetWarehouseByCode(ctx context.Context, code string) (*model.Warehouse, error)
	ListWarehouses(ctx context.Context, filter model.ListWarehouseFilter) ([]model.Warehouse, error)
	ListActiveWarehouses(ctx context.Context) ([]model.Warehouse, error)
	// Lookup kho gần nhất có hàng
	FindNearestWarehouseWithStock(ctx context.Context, bookID uuid.UUID, lat float64, lon float64, requiredQty int) (*model.WarehouseWithInventory, error)
	// Validate kho cho order
	ValidateWarehouseHasStock(ctx context.Context, warehouseID, bookID uuid.UUID, requiredQty int) (bool, error)
}
