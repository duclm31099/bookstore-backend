package model

import (
	"time"

	"github.com/google/uuid"
)

// New structs for new schema
type NearestWarehouse struct {
	BookID            uuid.UUID `json:"book_id"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseName     string    `json:"warehouse_name"`
	AvailableQuantity int       `json:"available_quantity"`
	DistanceKM        float64   `json:"distance_km"`
}

type TotalStockResponse struct {
	BookID         uuid.UUID `json:"book_id"`
	TotalQuantity  int       `json:"total_quantity"`
	TotalReserved  int       `json:"total_reserved"`
	TotalAvailable int       `json:"total_available"`
	WarehouseCount int       `json:"warehouse_count"`
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

// Inventory represents warehouse_inventory table
type Inventory struct {
	ID uuid.UUID `json:"id" db:"id"`
	// Composite Primary Key
	WarehouseID uuid.UUID `json:"warehouse_id" db:"warehouse_id"`
	BookID      uuid.UUID `json:"book_id" db:"book_id"`

	// Stock columns
	Quantity       int  `json:"quantity" db:"quantity"`
	Reserved       int  `json:"reserved" db:"reserved"`
	AlertThreshold int  `json:"alert_threshold" db:"alert_threshold"`
	IsLowStock     bool `json:"is_low_stock"`
	// Metadata
	Version       int        `json:"version" db:"version"`
	LastRestockAt *time.Time `json:"last_restocked_at,omitempty" db:"last_restocked_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	UpdatedBy     *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`

	// Computed fields (not in DB)
	AvailableQuantity int    `json:"available" db:"-"`
	WarehouseName     string `json:"warehouse_name,omitempty" db:"-"` // Join field
}

// Warehouse represents warehouses table
type Warehouse struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Code      string     `json:"code" db:"code"`
	Address   string     `json:"address" db:"address"`
	Province  string     `json:"province" db:"province"`
	Latitude  *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude *float64   `json:"longitude,omitempty" db:"longitude"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	Version   int        `json:"version" db:"version"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// LowStockAlert represents low_stock_alerts table
type LowStockAlert struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	WarehouseID     uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	BookID          uuid.UUID  `json:"book_id" db:"book_id"`
	CurrentQuantity int        `json:"current_quantity" db:"current_quantity"`
	AlertThreshold  int        `json:"alert_threshold" db:"alert_threshold"`
	IsResolved      bool       `json:"is_resolved" db:"is_resolved"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`

	// Join fields
	WarehouseName string `json:"warehouse_name,omitempty" db:"-"`
	BookTitle     string `json:"book_title,omitempty" db:"-"`
	Priority      string `json:"priority" db:"-"` // critical/high/medium
}

// AuditLogEntry represents inventory_audit_log table
type AuditLogEntry struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	WarehouseID    uuid.UUID  `json:"warehouse_id" db:"warehouse_id"`
	BookID         uuid.UUID  `json:"book_id" db:"book_id"`
	Action         string     `json:"action" db:"action"` // RESTOCK, RESERVE, RELEASE, ADJUSTMENT, SALE
	OldQuantity    int        `json:"old_quantity" db:"old_quantity"`
	NewQuantity    int        `json:"new_quantity" db:"new_quantity"`
	OldReserved    int        `json:"old_reserved" db:"old_reserved"`
	NewReserved    int        `json:"new_reserved" db:"new_reserved"`
	QuantityChange int        `json:"quantity_change" db:"quantity_change"`
	Reason         *string    `json:"reason,omitempty" db:"reason"`
	ChangedBy      *uuid.UUID `json:"changed_by,omitempty" db:"changed_by"`
	IPAddress      *string    `json:"ip_address,omitempty" db:"ip_address"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}
