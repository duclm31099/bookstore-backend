package model

import (
	"time"

	"github.com/google/uuid"
)

// Entity kho (map bảng warehouses)
type Warehouse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Code      string     `json:"code"`
	Address   string     `json:"address"`
	Province  string     `json:"province"`
	Latitude  *float64   `json:"latitude,omitempty"`
	Longitude *float64   `json:"longitude,omitempty"`
	IsActive  bool       `json:"is_active"`
	Version   int        `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Khi lookup inventory cho book tại các kho
// (map chức năng FindWarehousesWithStockByDistance)
type WarehouseWithInventory struct {
	Warehouse
	AvailableQuantity int     `json:"available_quantity"`
	DistanceKm        float64 `json:"distance_km"`
}

// DTO cho tạo kho
// Code sẽ được service/repo tự sinh, không truyền từ client
type CreateWarehouseRequest struct {
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	Province  string   `json:"province"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type UpdateWarehouseRequest struct {
	Name      *string  `json:"name,omitempty"`
	Address   *string  `json:"address,omitempty"`
	Province  *string  `json:"province,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
}
type ListWarehouseFilter struct {
	Keyword  string
	Province string
	IsActive *bool
	Offset   int
	Limit    int
}
