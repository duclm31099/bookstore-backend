package model

import (
	"time"

	"github.com/google/uuid"
)

// Inventory represents the database entity for inventories table
// Maps directly to PostgreSQL schema
type Inventory struct {
	// Identity
	ID     uuid.UUID `db:"id"`
	BookID uuid.UUID `db:"book_id"`

	// Location (Vietnam warehouses: HN, HCM, DN, CT)
	WarehouseLocation string `db:"warehouse_location"`

	// Stock levels
	Quantity          int `db:"quantity"`
	ReservedQuantity  int `db:"reserved_quantity"`
	AvailableQuantity int `db:"available_quantity"` // GENERATED ALWAYS AS (quantity - reserved_quantity) STORED

	// Alerts
	LowStockThreshold int  `db:"low_stock_threshold"`
	IsLowStock        bool `db:"is_low_stock"` // GENERATED ALWAYS AS (quantity - reserved_quantity <= low_stock_threshold) STORED

	// Optimistic locking
	Version int `db:"version"`

	// Timestamps
	LastRestockAt *time.Time `db:"last_restock_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

// ValidWarehouseLocations defines allowed warehouse locations in Vietnam
var ValidWarehouseLocations = []string{"HN", "HCM", "DN", "CT"}

// IsValidWarehouse checks if warehouse location is valid
func IsValidWarehouse(location string) bool {
	for _, valid := range ValidWarehouseLocations {
		if valid == location {
			return true
		}
	}
	return false
}

// InventoryMovement represents inventory movement audit trail record
type InventoryMovement struct {
	ID             uuid.UUID
	InventoryID    uuid.UUID
	MovementType   string // inbound, outbound, adjustment, return, reserve, release
	Quantity       int    // Can be negative
	QuantityBefore int
	QuantityAfter  int
	ReferenceType  *string
	ReferenceID    *uuid.UUID
	Notes          *string
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
}
