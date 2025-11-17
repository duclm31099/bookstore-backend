package service

import (
	"bookstore-backend/internal/domains/warehouse/model"
	"bookstore-backend/internal/domains/warehouse/repository"
	"context"
	"errors"

	"github.com/google/uuid"
)

type warehouseService struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) Service {
	return &warehouseService{repo: repo}
}

func (s *warehouseService) CreateWarehouse(ctx context.Context, req model.CreateWarehouseRequest) (*model.Warehouse, error) {
	return s.repo.CreateWarehouse(ctx, req)
}

func (s *warehouseService) UpdateWarehouse(ctx context.Context, id uuid.UUID, req model.UpdateWarehouseRequest) (*model.Warehouse, error) {
	return s.repo.UpdateWarehouse(ctx, id, req)
}

func (s *warehouseService) SoftDeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	hasInv, err := s.repo.HasInventory(ctx, id)
	if err != nil {
		return err
	}
	if hasInv {
		return errors.New("cannot delete warehouse with existing inventory")
	}
	return s.repo.SoftDeleteWarehouse(ctx, id)
}

func (s *warehouseService) GetWarehouseByID(ctx context.Context, id uuid.UUID) (*model.Warehouse, error) {
	return s.repo.GetWarehouseByID(ctx, id)
}

func (s *warehouseService) GetWarehouseByCode(ctx context.Context, code string) (*model.Warehouse, error) {
	return s.repo.GetWarehouseByCode(ctx, code)
}

func (s *warehouseService) ListWarehouses(ctx context.Context, filter model.ListWarehouseFilter) ([]model.Warehouse, error) {
	return s.repo.ListWarehouses(ctx, filter)
}

func (s *warehouseService) ListActiveWarehouses(ctx context.Context) ([]model.Warehouse, error) {
	return s.repo.ListActiveWarehouses(ctx)
}

// Trả về kho gần nhất còn hàng
func (s *warehouseService) FindNearestWarehouseWithStock(ctx context.Context, bookID uuid.UUID, lat float64, lon float64, requiredQty int) (*model.WarehouseWithInventory, error) {
	list, err := s.repo.FindWarehousesWithStockByDistance(ctx, bookID, lat, lon, requiredQty)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("no warehouse with sufficient stock found")
	}
	return &list[0], nil
}

func (s *warehouseService) ValidateWarehouseHasStock(ctx context.Context, warehouseID, bookID uuid.UUID, requiredQty int) (bool, error) {
	// Đơn giản: query toàn bộ kho còn hàng, check xem warehouseID có nằm trong này không
	// (có thể optimize riêng nhưng cách này đủ chắc)
	list, err := s.repo.FindWarehousesWithStockByDistance(ctx, bookID, 0, 0, requiredQty)
	if err != nil {
		return false, err
	}
	for _, w := range list {
		if w.ID == warehouseID && w.AvailableQuantity >= requiredQty {
			return true, nil
		}
	}
	return false, nil
}
