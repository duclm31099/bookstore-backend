package model

import (
	"time"

	"github.com/google/uuid"
)

// ===================================
// REQUEST DTOs
// ===================================

// CreateInventoryRequest represents the request payload for creating inventory
type CreateInventoryRequest struct {
	BookID            uuid.UUID `json:"book_id" binding:"required"`
	WarehouseLocation string    `json:"warehouse_location" binding:"required,oneof=HN HCM DN CT ALL"` // ALL = create for all warehouses
	Quantity          int       `json:"quantity" binding:"required,min=0"`
	LowStockThreshold *int      `json:"low_stock_threshold" binding:"omitempty,min=0"` // Optional, default 10
}

// UpdateInventoryRequest represents the request payload for updating inventory
type UpdateInventoryRequest struct {
	Quantity          *int `json:"quantity" binding:"omitempty,min=0"`
	ReservedQuantity  *int `json:"reserved_quantity" binding:"omitempty,min=0"`
	LowStockThreshold *int `json:"low_stock_threshold" binding:"omitempty,min=0"`
	Version           int  `json:"version" binding:"required,min=1"` // Optimistic locking
}

// ListInventoryRequest represents query parameters for listing inventories
type ListInventoryRequest struct {
	BookID            *uuid.UUID `form:"book_id"`
	WarehouseLocation *string    `form:"warehouse_location" binding:"omitempty,oneof=HN HCM DN CT"`
	IsLowStock        *bool      `form:"is_low_stock"`
	Page              int        `form:"page" binding:"required,min=1"`
	Limit             int        `form:"limit" binding:"required,min=1,max=100"`
}

// SearchInventoryRequest represents query for searching by book_id + warehouse
type SearchInventoryRequest struct {
	BookID            uuid.UUID `form:"book_id" binding:"required"`
	WarehouseLocation string    `form:"warehouse_location" binding:"required,oneof=HN HCM DN CT"`
}

// ===================================
// RESPONSE DTOs
// ===================================

// InventoryResponse represents the response payload for inventory
type InventoryResponse struct {
	ID                uuid.UUID  `json:"id"`
	BookID            uuid.UUID  `json:"book_id"`
	WarehouseLocation string     `json:"warehouse_location"`
	Quantity          int        `json:"quantity"`
	ReservedQuantity  int        `json:"reserved_quantity"`
	AvailableQuantity int        `json:"available_quantity"` // GENERATED column
	LowStockThreshold int        `json:"low_stock_threshold"`
	IsLowStock        bool       `json:"is_low_stock"` // GENERATED column
	Version           int        `json:"version"`
	LastRestockAt     *time.Time `json:"last_restock_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ListInventoryResponse represents paginated list response
type ListInventoryResponse struct {
	Items      []InventoryResponse `json:"items"`
	TotalItems int                 `json:"total_items"`
	TotalPages int                 `json:"total_pages"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
}

// ===================================
// MAPPERS (Model <-> DTO)
// ===================================

// ToResponse converts Inventory model to InventoryResponse DTO
func (i *Inventory) ToResponse() InventoryResponse {
	return InventoryResponse{
		ID:                i.ID,
		BookID:            i.BookID,
		WarehouseLocation: i.WarehouseLocation,
		Quantity:          i.Quantity,
		ReservedQuantity:  i.ReservedQuantity,
		AvailableQuantity: i.AvailableQuantity,
		LowStockThreshold: i.LowStockThreshold,
		IsLowStock:        i.IsLowStock,
		Version:           i.Version,
		LastRestockAt:     i.LastRestockAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

// ToResponseList converts slice of Inventory models to InventoryResponse DTOs
func ToResponseList(inventories []Inventory) []InventoryResponse {
	responses := make([]InventoryResponse, len(inventories))
	for i, inv := range inventories {
		responses[i] = inv.ToResponse()
	}
	return responses
}

// ===================================
// RESERVE/RELEASE STOCK DTOs
// ===================================

// ReserveStockRequest represents request to reserve stock for an order
type ReserveStockRequest struct {
	BookID            uuid.UUID `json:"book_id" binding:"required"`
	WarehouseLocation string    `json:"warehouse_location" binding:"required,oneof=HN HCM DN CT"`
	Quantity          int       `json:"quantity" binding:"required,min=1"`
	ReferenceType     string    `json:"reference_type" binding:"required,oneof=order purchase"`
	ReferenceID       uuid.UUID `json:"reference_id" binding:"required"`
}

// ReleaseStockRequest represents request to release reserved stock
type ReleaseStockRequest struct {
	BookID            uuid.UUID `json:"book_id" binding:"required"`
	WarehouseLocation string    `json:"warehouse_location" binding:"required,oneof=HN HCM DN CT"`
	Quantity          int       `json:"quantity" binding:"required,min=1"`
	ReferenceID       uuid.UUID `json:"reference_id" binding:"required"`
}

// ReserveStockResponse represents response after reserving stock
type ReserveStockResponse struct {
	Success           bool      `json:"success"`
	BookID            uuid.UUID `json:"book_id"`
	WarehouseLocation string    `json:"warehouse_location"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	Message           string    `json:"message"`
}

// ReleaseStockResponse represents response after releasing stock
type ReleaseStockResponse struct {
	Success           bool      `json:"success"`
	BookID            uuid.UUID `json:"book_id"`
	WarehouseLocation string    `json:"warehouse_location"`
	ReleasedQuantity  int       `json:"released_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	Message           string    `json:"message"`
}

// ===================================
// CHECK AVAILABILITY DTOs
// ===================================

// CheckAvailabilityRequest represents request to check stock availability for multiple items
type CheckAvailabilityRequest struct {
	Items              []CheckAvailabilityItem `json:"items" binding:"required,min=1,max=100"`
	PreferredWarehouse *string                 `json:"preferred_warehouse,omitempty" binding:"omitempty,oneof=HN HCM DN CT"`
}

// CheckAvailabilityItem represents single item in availability check
type CheckAvailabilityItem struct {
	BookID   uuid.UUID `json:"book_id" binding:"required"`
	Quantity int       `json:"quantity" binding:"required,min=1"`
}

// CheckAvailabilityResponse represents response for availability check
type CheckAvailabilityResponse struct {
	Overall       bool                            `json:"overall"`                  // All items available?
	Items         []CheckAvailabilityItemResponse `json:"items"`                    // Per-item details
	Fulfillable   bool                            `json:"fulfillable"`              // Can fulfill all items?
	SuggestedFrom string                          `json:"suggested_from,omitempty"` // Best warehouse to fulfill from
}

// CheckAvailabilityItemResponse represents per-item availability status
type CheckAvailabilityItemResponse struct {
	BookID            uuid.UUID               `json:"book_id"`
	RequestedQuantity int                     `json:"requested_quantity"`
	Available         bool                    `json:"available"`                // Enough in any warehouse?
	WarehouseDetails  []WarehouseAvailability `json:"warehouse_details"`        // Details per warehouse
	TotalAvailable    int                     `json:"total_available"`          // Total across all warehouses
	FulfillableFrom   []string                `json:"fulfillable_from"`         // List of warehouses with enough stock
	Recommendation    string                  `json:"recommendation,omitempty"` // Helpful message
}

// WarehouseAvailability represents inventory status in single warehouse
type WarehouseAvailability struct {
	Warehouse         string `json:"warehouse"`
	Quantity          int    `json:"quantity"`
	Reserved          int    `json:"reserved"`
	Available         int    `json:"available"`
	CanFulfill        bool   `json:"can_fulfill"` // Has enough for this item
	IsLowStock        bool   `json:"is_low_stock"`
	LowStockThreshold int    `json:"low_stock_threshold"`
}

// StockSummaryRequest for getting total stock across warehouses
type StockSummaryRequest struct {
	BookID uuid.UUID `form:"book_id" binding:"required"`
}

// StockSummaryResponse for total stock summary
type StockSummaryResponse struct {
	BookID         uuid.UUID               `json:"book_id"`
	TotalQuantity  int                     `json:"total_quantity"`
	TotalReserved  int                     `json:"total_reserved"`
	TotalAvailable int                     `json:"total_available"`
	ByWarehouse    []WarehouseStockSummary `json:"by_warehouse"`
}

// WarehouseStockSummary for per-warehouse summary
type WarehouseStockSummary struct {
	Warehouse         string `json:"warehouse"`
	Quantity          int    `json:"quantity"`
	Reserved          int    `json:"reserved"`
	Available         int    `json:"available"`
	IsLowStock        bool   `json:"is_low_stock"`
	LowStockThreshold int    `json:"low_stock_threshold"`
}

// ===================================
// INVENTORY MOVEMENT DTOs
// ===================================

// CreateMovementRequest represents request to create manual inventory adjustment
type CreateMovementRequest struct {
	InventoryID   uuid.UUID  `json:"inventory_id" binding:"required"`
	MovementType  string     `json:"movement_type" binding:"required,oneof=inbound outbound adjustment return"`
	Quantity      int        `json:"quantity" binding:"required"` // Can be negative for outbound
	ReferenceType *string    `json:"reference_type,omitempty" binding:"omitempty,oneof=order purchase manual return"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty"`
	Notes         string     `json:"notes" binding:"required,max=500"`
}

// ListMovementsRequest represents query for listing movements with pagination
type ListMovementsRequest struct {
	InventoryID   *uuid.UUID `form:"inventory_id"`
	MovementType  *string    `form:"movement_type" binding:"omitempty,oneof=inbound outbound adjustment reserve release return"`
	ReferenceType *string    `form:"reference_type" binding:"omitempty,oneof=order purchase manual return"`
	Page          int        `form:"page" binding:"required,min=1"`
	Limit         int        `form:"limit" binding:"required,min=1,max=100"`
}

// MovementResponse represents inventory movement response
type MovementResponse struct {
	ID                uuid.UUID  `json:"id"`
	InventoryID       uuid.UUID  `json:"inventory_id"`
	BookID            uuid.UUID  `json:"book_id"`
	WarehouseLocation string     `json:"warehouse_location"`
	MovementType      string     `json:"movement_type"` // inbound, outbound, adjustment, reserve, release, return
	Quantity          int        `json:"quantity"`      // Can be negative
	QuantityBefore    int        `json:"quantity_before"`
	QuantityAfter     int        `json:"quantity_after"`
	ReferenceType     *string    `json:"reference_type,omitempty"`
	ReferenceID       *uuid.UUID `json:"reference_id,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// ListMovementsResponse represents paginated movements list
type ListMovementsResponse struct {
	Items      []MovementResponse `json:"items"`
	TotalItems int                `json:"total_items"`
	TotalPages int                `json:"total_pages"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
}

// MovementStatsResponse represents movement statistics for a book
type MovementStatsResponse struct {
	BookID         uuid.UUID      `json:"book_id"`
	TotalInbound   int            `json:"total_inbound"`
	TotalOutbound  int            `json:"total_outbound"`
	TotalReserved  int            `json:"total_reserved"`
	TotalReleased  int            `json:"total_released"`
	NetMovement    int            `json:"net_movement"` // inbound - outbound
	ByWarehouse    map[string]int `json:"by_warehouse"`
	ByMovementType map[string]int `json:"by_movement_type"`
	LastMovement   *time.Time     `json:"last_movement,omitempty"`
}

// ===================================
// STOCK AGGREGATION & SUMMARY DTOs
// ===================================

// DashboardRequest for dashboard metrics
type DashboardRequest struct {
	StartDate *time.Time `form:"start_date,omitempty"` // Optional: filter by date range
	EndDate   *time.Time `form:"end_date,omitempty"`
}

// InventoryDashboardResponse represents inventory dashboard metrics
type InventoryDashboardResponse struct {
	Summary          DashboardSummary      `json:"summary"`
	ByWarehouse      []WarehouseMetrics    `json:"by_warehouse"`
	LowStockItems    []LowStockItem        `json:"low_stock_items"`
	OutOfStockItems  []OutOfStockItem      `json:"out_of_stock_items"`
	ReservedAnalysis ReservedStockAnalysis `json:"reserved_analysis"`
	MovementTrends   []MovementTrend       `json:"movement_trends"`
	WarehouseHealth  map[string]float64    `json:"warehouse_health"` // Utilization %
	Timestamp        time.Time             `json:"timestamp"`
}

// DashboardSummary represents overall inventory summary
type DashboardSummary struct {
	TotalBooks        int     `json:"total_books"`
	TotalQuantity     int     `json:"total_quantity"`
	TotalReserved     int     `json:"total_reserved"`
	TotalAvailable    int     `json:"total_available"`
	LowStockCount     int     `json:"low_stock_count"`
	OutOfStockCount   int     `json:"out_of_stock_count"`
	HealthScore       float64 `json:"health_score"`       // 0-100
	InventoryTurnover float64 `json:"inventory_turnover"` // Times per period
	HealthStatus      string  `json:"health_status"`      // "healthy", "warning", "critical"
}

// WarehouseMetrics represents metrics for single warehouse
type WarehouseMetrics struct {
	Warehouse       string     `json:"warehouse"`
	TotalQuantity   int        `json:"total_quantity"`
	TotalReserved   int        `json:"total_reserved"`
	TotalAvailable  int        `json:"total_available"`
	BookCount       int        `json:"book_count"`
	LowStockCount   int        `json:"low_stock_count"`
	OutOfStockCount int        `json:"out_of_stock_count"`
	Utilization     float64    `json:"utilization"` // % of capacity
	HealthScore     float64    `json:"health_score"`
	ReservationRate float64    `json:"reservation_rate"` // % of total reserved
	LastMovement    *time.Time `json:"last_movement,omitempty"`
}

// LowStockItem represents book with low stock
type LowStockItem struct {
	BookID             uuid.UUID `json:"book_id"`
	BookTitle          string    `json:"book_title,omitempty"`
	WarehouseLocation  string    `json:"warehouse_location"`
	CurrentStock       int       `json:"current_stock"`
	ReservedStock      int       `json:"reserved_stock"`
	AvailableStock     int       `json:"available_stock"`
	LowStockThreshold  int       `json:"low_stock_threshold"`
	DaysUntilStockout  int       `json:"days_until_stockout"` // Estimated
	RecommendedReorder int       `json:"recommended_reorder"` // Quantity
	Priority           string    `json:"priority"`            // "critical", "high", "medium"
}

// OutOfStockItem represents book with zero stock
type OutOfStockItem struct {
	BookID            uuid.UUID  `json:"book_id"`
	BookTitle         string     `json:"book_title,omitempty"`
	WarehouseLocation string     `json:"warehouse_location"`
	ReservedStock     int        `json:"reserved_stock"`
	DaysSinceStockout int        `json:"days_since_stockout"`
	LastRestockDate   *time.Time `json:"last_restock_date,omitempty"`
}

// ReservedStockAnalysis represents analysis of reserved stock
type ReservedStockAnalysis struct {
	TotalReserved      int                 `json:"total_reserved"`
	TotalAvailable     int                 `json:"total_available"`
	ReservationRate    float64             `json:"reservation_rate"` // % of total
	ByWarehouse        map[string]int      `json:"by_warehouse"`
	LongestReserved    *ReservedItemDetail `json:"longest_reserved,omitempty"`
	HighestReservation *ReservedItemDetail `json:"highest_reservation,omitempty"`
}

// ReservedItemDetail represents detail of reserved item
type ReservedItemDetail struct {
	BookID           uuid.UUID `json:"book_id"`
	Warehouse        string    `json:"warehouse"`
	ReservedQuantity int       `json:"reserved_quantity"`
	DateReserved     time.Time `json:"date_reserved"`
	DaysReserved     int       `json:"days_reserved"`
}

// MovementTrend represents movement trend for analytics
type MovementTrend struct {
	Period           string `json:"period"` // e.g., "2025-11-06"
	Inbound          int    `json:"inbound"`
	Outbound         int    `json:"outbound"`
	NetMovement      int    `json:"net_movement"`
	TransactionCount int    `json:"transaction_count"`
}

// InventoryValueResponse represents inventory value calculation
type InventoryValueResponse struct {
	Summary           InventoryValueSummary   `json:"summary"`
	ByWarehouse       []WarehouseValueMetrics `json:"by_warehouse"`
	ByBook            []BookValueMetrics      `json:"by_book"`             // Top 10
	MostValuableItems []BookValueMetrics      `json:"most_valuable_items"` // Top 5
	LeastMovingItems  []BookValueMetrics      `json:"least_moving_items"`  // Bottom 5
}

// InventoryValueSummary represents total inventory value
type InventoryValueSummary struct {
	TotalValue         float64 `json:"total_value"`
	TotalCost          float64 `json:"total_cost"`
	ReservedValue      float64 `json:"reserved_value"`
	AvailableValue     float64 `json:"available_value"`
	AvgUnitValue       float64 `json:"avg_unit_value"`
	ValueConcentration float64 `json:"value_concentration"` // % of top 20% items
}

// WarehouseValueMetrics represents warehouse value breakdown
type WarehouseValueMetrics struct {
	Warehouse      string  `json:"warehouse"`
	TotalValue     float64 `json:"total_value"`
	ReservedValue  float64 `json:"reserved_value"`
	AvailableValue float64 `json:"available_value"`
	ShareOfTotal   float64 `json:"share_of_total"` // %
}

// BookValueMetrics represents per-book value metrics
type BookValueMetrics struct {
	BookID         uuid.UUID `json:"book_id"`
	BookTitle      string    `json:"book_title,omitempty"`
	TotalQuantity  int       `json:"total_quantity"`
	UnitPrice      float64   `json:"unit_price"`
	TotalValue     float64   `json:"total_value"`
	ReservedValue  float64   `json:"reserved_value"`
	AvailableValue float64   `json:"available_value"`
	Turnover       int       `json:"turnover"` // Movement count
	Days           int       `json:"days"`     // Days in inventory
}
