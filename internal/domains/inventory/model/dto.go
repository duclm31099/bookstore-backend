// internal/domains/inventory/model/dto.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// ========================================
// INVENTORY REQUESTS
// ========================================

type CreateInventoryRequest struct {
	BookID                 uuid.UUID  `json:"book_id" validate:"required"`
	WarehouseID            *uuid.UUID `json:"warehouse_id,omitempty"` // Nil = create for all
	CreateForAllWarehouses bool       `json:"create_for_all_warehouses"`
	Quantity               int        `json:"quantity" validate:"required,gte=0"`
	AlertThreshold         *int       `json:"alert_threshold,omitempty" validate:"omitempty,gte=0"`
	UpdatedBy              *uuid.UUID `json:"updated_by,omitempty"`
}

type UpdateInventoryRequest struct {
	Quantity       *int       `json:"quantity,omitempty" validate:"omitempty,gte=0"`
	Reserved       *int       `json:"reserved,omitempty" validate:"omitempty,gte=0"`
	AlertThreshold *int       `json:"alert_threshold,omitempty" validate:"omitempty,gte=0"`
	Version        int        `json:"version" validate:"required"` // Optimistic lock
	UpdatedBy      *uuid.UUID `json:"updated_by,omitempty"`
}

type ListInventoryRequest struct {
	BookID            *string `form:"book_id" json:"book_id,omitempty"`
	WarehouseID       *string `form:"warehouse_id" json:"warehouse_id,omitempty"`
	IsLowStock        *bool   `form:"is_low_stock" json:"is_low_stock,omitempty"`
	HasAvailableStock *bool   `form:"has_available_stock" json:"has_available_stock,omitempty"`
	Page              int     `form:"page" json:"page" binding:"required,gte=1"`
	Limit             int     `form:"limit" json:"limit" binding:"required,gte=1,lte=100"`
}

// ========================================
// STOCK OPERATION REQUESTS
// ========================================

type ReserveStockRequest struct {
	BookID      uuid.UUID  `json:"book_id" validate:"required"`
	WarehouseID *uuid.UUID `json:"warehouse_id,omitempty"` // Nil = auto-select nearest
	Quantity    int        `json:"quantity" validate:"required,gte=1"`
	ReferenceID uuid.UUID  `json:"reference_id" validate:"required"` // Order ID
	UserID      *uuid.UUID `json:"user_id,omitempty"`

	// For auto warehouse selection
	CustomerLatitude  *float64 `json:"customer_latitude,omitempty"`
	CustomerLongitude *float64 `json:"customer_longitude,omitempty"`
}

type ReleaseStockRequest struct {
	WarehouseID uuid.UUID  `json:"warehouse_id" validate:"required"`
	BookID      uuid.UUID  `json:"book_id" validate:"required"`
	Quantity    int        `json:"quantity" validate:"required,gte=1"`
	ReferenceID uuid.UUID  `json:"reference_id" validate:"required"` // Order ID
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Reason      *string    `json:"reason,omitempty"` // "timeout", "cancelled", "payment_failed"
}

type CompleteSaleRequest struct {
	WarehouseID uuid.UUID  `json:"warehouse_id" validate:"required"`
	BookID      uuid.UUID  `json:"book_id" validate:"required"`
	Quantity    int        `json:"quantity" validate:"required,gte=1"`
	ReferenceID uuid.UUID  `json:"reference_id" validate:"required"` // Order ID
	UserID      *uuid.UUID `json:"user_id,omitempty"`
}

type AdjustStockRequest struct {
	WarehouseID uuid.UUID `json:"warehouse_id" validate:"required"`
	BookID      uuid.UUID `json:"book_id" validate:"required"`
	NewQuantity int       `json:"new_quantity" validate:"required,gte=0"`
	Reason      string    `json:"reason" validate:"required,min=10"` // Mandatory for audit
	Version     int       `json:"version" validate:"required"`
	ChangedBy   uuid.UUID `json:"changed_by" validate:"required"` // Admin user
	IPAddress   *string   `json:"ip_address,omitempty"`
}

type RestockRequest struct {
	WarehouseID   uuid.UUID  `json:"warehouse_id" validate:"required"`
	BookID        uuid.UUID  `json:"book_id" validate:"required"`
	QuantityToAdd int        `json:"quantity_to_add" validate:"required,gte=1"`
	Reason        *string    `json:"reason,omitempty"`
	UpdatedBy     *uuid.UUID `json:"updated_by,omitempty"`
}

// ========================================
// WAREHOUSE SELECTION REQUESTS
// ========================================

type FindWarehouseRequest struct {
	BookID            uuid.UUID `json:"book_id" validate:"required"`
	RequiredQuantity  int       `json:"required_quantity" validate:"required,gte=1"`
	CustomerLatitude  float64   `json:"latitude" validate:"required"`
	CustomerLongitude float64   `json:"longitude" validate:"required"`
}

type CheckAvailabilityRequest struct {
	Items                []CheckAvailabilityItem `json:"items" validate:"required,dive"`
	PreferredWarehouseID *uuid.UUID              `json:"preferred_warehouse_id,omitempty"`
	CustomerLatitude     *string                 `json:"latitude" validate:"required"`
	CustomerLongitude    *string                 `json:"longitude" validate:"required"`
}

type CheckAvailabilityItem struct {
	BookID   uuid.UUID `json:"book_id" validate:"required"`
	Quantity int       `json:"quantity" validate:"required,gte=1"`
}

// ========================================
// BULK OPERATIONS
// ========================================

type BulkUpdateRequest struct {
	CSVPath    string    `json:"csv_path" validate:"required"`
	UploadedBy uuid.UUID `json:"uploaded_by" validate:"required"`
}

// ========================================
// AUDIT & REPORTING REQUESTS
// ========================================

type AuditTrailRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id,omitempty" form:"warehouse_id"`
	BookID      *uuid.UUID `json:"book_id,omitempty" form:"book_id"`
	Action      *string    `json:"action,omitempty" form:"action"` // RESTOCK, RESERVE, etc.
	ChangedBy   *uuid.UUID `json:"changed_by,omitempty" form:"changed_by"`
	StartDate   *time.Time `json:"start_date,omitempty" form:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty" form:"end_date"`
	Page        int        `json:"page" validate:"required,gte=1" form:"page"`
	Limit       int        `json:"limit" validate:"required,gte=1,lte=100" form:"limit"`
}

type ExportAuditRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id,omitempty"`
	StartDate   time.Time  `json:"start_date" validate:"required"`
	EndDate     time.Time  `json:"end_date" validate:"required"`
	Format      string     `json:"format" validate:"required,oneof=csv xlsx"`
}

// ========================================
// WAREHOUSE REQUESTS
// ========================================

type CreateWarehouseRequest struct {
	Name      string   `json:"name" validate:"required,min=3,max=100"`
	Code      string   `json:"code" validate:"required,min=2,max=20"`
	Address   string   `json:"address" validate:"required"`
	Province  string   `json:"province" validate:"required"`
	Latitude  *float64 `json:"latitude,omitempty" validate:"omitempty,gte=-90,lte=90"`
	Longitude *float64 `json:"longitude,omitempty" validate:"omitempty,gte=-180,lte=180"`
}

type UpdateWarehouseRequest struct {
	Name      *string  `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	Address   *string  `json:"address,omitempty"`
	Province  *string  `json:"province,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty" validate:"omitempty,gte=-90,lte=90"`
	Longitude *float64 `json:"longitude,omitempty" validate:"omitempty,gte=-180,lte=180"`
	IsActive  *bool    `json:"is_active,omitempty"`
	Version   int      `json:"version" validate:"required"`
}

type ListWarehousesRequest struct {
	IsActive *bool   `json:"is_active,omitempty"`
	Province *string `json:"province,omitempty"`
}

// ========================================
// RESPONSES
// ========================================

type InventoryResponse struct {
	WarehouseID    uuid.UUID  `json:"warehouse_id"`
	WarehouseName  string     `json:"warehouse_name"`
	BookID         uuid.UUID  `json:"book_id"`
	Quantity       int        `json:"quantity"`
	Reserved       int        `json:"reserved"`
	Available      int        `json:"available"`
	AlertThreshold int        `json:"alert_threshold"`
	IsLowStock     bool       `json:"is_low_stock"`
	Version        int        `json:"version"`
	LastRestockAt  *time.Time `json:"last_restocked_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ListInventoryResponse struct {
	Items      []InventoryResponse `json:"items"`
	TotalItems int                 `json:"total_items"`
	TotalPages int                 `json:"total_pages"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
}

type ReserveStockResponse struct {
	Success           bool      `json:"success"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseName     string    `json:"warehouse_name"`
	BookID            uuid.UUID `json:"book_id"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	ExpiresAt         time.Time `json:"expires_at"` // Now + 15 minutes
	Message           string    `json:"message"`
}

type ReleaseStockResponse struct {
	Success           bool      `json:"success"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	BookID            uuid.UUID `json:"book_id"`
	ReleasedQuantity  int       `json:"released_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	Message           string    `json:"message"`
}

type CompleteSaleResponse struct {
	Success      bool      `json:"success"`
	WarehouseID  uuid.UUID `json:"warehouse_id"`
	BookID       uuid.UUID `json:"book_id"`
	SoldQuantity int       `json:"sold_quantity"`
	Remaining    int       `json:"remaining_quantity"`
	Message      string    `json:"message"`
}

type AdjustStockResponse struct {
	Success        bool      `json:"success"`
	WarehouseID    uuid.UUID `json:"warehouse_id"`
	BookID         uuid.UUID `json:"book_id"`
	OldQuantity    int       `json:"old_quantity"`
	NewQuantity    int       `json:"new_quantity"`
	QuantityChange int       `json:"quantity_change"`
	AuditLogID     uuid.UUID `json:"audit_log_id"`
	Message        string    `json:"message"`
}

type RestockResponse struct {
	Success       bool      `json:"success"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	BookID        uuid.UUID `json:"book_id"`
	QuantityAdded int       `json:"quantity_added"`
	NewQuantity   int       `json:"new_quantity"`
	LastRestockAt time.Time `json:"last_restocked_at"`
	Message       string    `json:"message"`
}

type WarehouseRecommendation struct {
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseName     string    `json:"warehouse_name"`
	DistanceKM        float64   `json:"distance_km"`
	AvailableQuantity int       `json:"available_quantity"`
	EstimatedDelivery string    `json:"estimated_delivery"` // "1-2 days", "3-5 days"
}

type CheckAvailabilityResponse struct {
	Overall              bool                            `json:"overall_fulfillable"`
	Items                []CheckAvailabilityItemResponse `json:"items"`
	RecommendedWarehouse *WarehouseRecommendation        `json:"recommended_warehouse,omitempty"`
	RequiresSplit        bool                            `json:"requires_split"` // Multiple warehouses needed
}

type CheckAvailabilityItemResponse struct {
	BookID            uuid.UUID              `json:"book_id"`
	RequestedQuantity int                    `json:"requested_quantity"`
	TotalAvailable    int                    `json:"total_available"`
	Fulfillable       bool                   `json:"fulfillable"`
	WarehouseDetails  []WarehouseStockDetail `json:"warehouse_details"`
	Recommendation    string                 `json:"recommendation,omitempty"`
}

type WarehouseStockDetail struct {
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	Available     int       `json:"available"`
	CanFulfill    bool      `json:"can_fulfill"`
	DistanceKM    *float64  `json:"distance_km,omitempty"`
}

type StockSummaryResponse struct {
	BookID         uuid.UUID               `json:"book_id"`
	TotalQuantity  int                     `json:"total_quantity"`
	TotalReserved  int                     `json:"total_reserved"`
	TotalAvailable int                     `json:"total_available"`
	WarehouseCount int                     `json:"warehouse_count"`
	ByWarehouse    []WarehouseStockSummary `json:"by_warehouse"`
}

type WarehouseStockSummary struct {
	WarehouseID    uuid.UUID `json:"warehouse_id"`
	WarehouseName  string    `json:"warehouse_name"`
	Quantity       int       `json:"quantity"`
	Reserved       int       `json:"reserved"`
	Available      int       `json:"available"`
	IsLowStock     bool      `json:"is_low_stock"`
	AlertThreshold int       `json:"alert_threshold"`
}

type BulkUpdateJobResponse struct {
	JobID     uuid.UUID `json:"job_id"`
	Status    string    `json:"status"` // "queued", "processing"
	TotalRows int       `json:"total_rows"`
	Message   string    `json:"message"`
}

type BulkUpdateStatusResponse struct {
	JobID         uuid.UUID         `json:"job_id"`
	Status        string            `json:"status"` // "processing", "completed", "failed"
	TotalRows     int               `json:"total_rows"`
	ProcessedRows int               `json:"processed_rows"`
	SuccessRows   int               `json:"success_rows"`
	ErrorRows     int               `json:"error_rows"`
	Errors        []BulkUpdateError `json:"errors,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
}

type BulkUpdateError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type AuditTrailResponse struct {
	Items      []AuditLogEntry `json:"items"`
	TotalItems int             `json:"total_items"`
	TotalPages int             `json:"total_pages"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
}

type InventoryHistoryResponse struct {
	WarehouseID uuid.UUID       `json:"warehouse_id"`
	BookID      uuid.UUID       `json:"book_id"`
	History     []AuditLogEntry `json:"history"`
	TotalItems  int             `json:"total_items"`
	Page        int             `json:"page"`
	Limit       int             `json:"limit"`
}

type ExportResponse struct {
	FileName  string    `json:"file_name"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	FileSize  int64     `json:"file_size_bytes"`
}

type DashboardSummaryResponse struct {
	Summary         DashboardSummary   `json:"summary"`
	ByWarehouse     []WarehouseMetrics `json:"by_warehouse"`
	LowStockAlerts  []LowStockAlert    `json:"low_stock_alerts"`
	RecentMovements []AuditLogEntry    `json:"recent_movements"`
	Timestamp       time.Time          `json:"timestamp"`
}

type DashboardSummary struct {
	TotalBooks      int     `json:"total_books"`
	TotalQuantity   int     `json:"total_quantity"`
	TotalReserved   int     `json:"total_reserved"`
	TotalAvailable  int     `json:"total_available"`
	LowStockCount   int     `json:"low_stock_count"`
	OutOfStockCount int     `json:"out_of_stock_count"`
	HealthScore     float64 `json:"health_score"`  // 0-100
	HealthStatus    string  `json:"health_status"` // "healthy", "warning", "critical"
}

type WarehouseMetrics struct {
	WarehouseID     uuid.UUID  `json:"warehouse_id"`
	WarehouseName   string     `json:"warehouse_name"`
	BookCount       int        `json:"book_count"`
	TotalQuantity   int        `json:"total_quantity"`
	TotalReserved   int        `json:"total_reserved"`
	TotalAvailable  int        `json:"total_available"`
	LowStockCount   int        `json:"low_stock_count"`
	OutOfStockCount int        `json:"out_of_stock_count"`
	Utilization     float64    `json:"utilization_percent"`      // quantity / capacity
	ReservationRate float64    `json:"reservation_rate_percent"` // reserved / quantity
	HealthScore     float64    `json:"health_score"`
	LastMovement    *time.Time `json:"last_movement,omitempty"`
}

type WarehousePerformanceResponse struct {
	WarehouseID    uuid.UUID        `json:"warehouse_id"`
	WarehouseName  string           `json:"warehouse_name"`
	Metrics        WarehouseMetrics `json:"metrics"`
	MovementTrends []MovementTrend  `json:"movement_trends"`
}

type MovementTrend struct {
	Period           string `json:"period"` // "2025-11-08"
	Inbound          int    `json:"inbound"`
	Outbound         int    `json:"outbound"`
	NetMovement      int    `json:"net_movement"`
	TransactionCount int    `json:"transaction_count"`
}

type InventoryValueResponse struct {
	TotalValue   float64            `json:"total_value"`
	ByWarehouse  map[string]float64 `json:"by_warehouse"`
	ByCategory   map[string]float64 `json:"by_category,omitempty"`
	Currency     string             `json:"currency"` // "VND"
	CalculatedAt time.Time          `json:"calculated_at"`
}

type ReservationAnalysisResponse struct {
	TotalReserved      int            `json:"total_reserved"`
	ReservationRate    float64        `json:"reservation_rate_percent"`
	ByWarehouse        map[string]int `json:"by_warehouse"`
	AvgDurationMinutes int            `json:"avg_duration_minutes"`
	ConversionRate     float64        `json:"conversion_rate_percent"` // reserved → sale
}

type WarehouseResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Address   string    `json:"address"`
	Province  string    `json:"province"`
	Latitude  *float64  `json:"latitude,omitempty"`
	Longitude *float64  `json:"longitude,omitempty"`
	IsActive  bool      `json:"is_active"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OutOfStockItem struct {
	BookID            uuid.UUID  `json:"book_id"`
	WarehouseID       uuid.UUID  `json:"warehouse_id"`
	WarehouseName     string     `json:"warehouse_name"`
	ReservedStock     int        `json:"reserved_stock"`
	LastRestockDate   *time.Time `json:"last_restock_date,omitempty"`
	DaysSinceStockout int        `json:"days_since_stockout"`
}

// internal/domains/inventory/model/dto.go

// ReservationMetrics for reservation analysis
type ReservationMetrics struct {
	TotalReserved      int            `json:"total_reserved"`
	ReservationRate    float64        `json:"reservation_rate_percent"`
	ByWarehouse        map[string]int `json:"by_warehouse"`
	AvgDurationMinutes int            `json:"avg_duration_minutes"`
	ConversionRate     float64        `json:"conversion_rate_percent"` // reserved → sale
}

// BookTotalStock đại diện cho 1 row trong view books_total_stock
type BookTotalStock struct {
	BookID              string   `json:"book_id"`
	TotalQuantity       int      `json:"total_quantity"`
	TotalReserved       int      `json:"total_reserved"`
	Available           int      `json:"available"`
	WarehouseCount      int      `json:"warehouse_count"`
	WarehousesWithStock []string `json:"warehouses_with_stock"`
}
