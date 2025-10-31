<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# TODO LIST CHI TIẾT CHO BACKEND DEVELOPER - SPRINT 15-16: MULTI-WAREHOUSE INVENTORY

Dựa trên URD, dưới đây là danh sách công việc chi tiết và đầy đủ cho backend developer trong Sprint 15-16 (Phase 3, 2 tuần - 10 ngày làm việc).[^1]

## 1. Warehouses + Warehouse_Inventory Tables (P3-T001)

### Mô tả

Tạo database schema cho hệ thống multi-warehouse inventory management.[^1]

### Database Schema

#### 1.1 Warehouses Table Migration

Tạo file `migrations/000025_create_warehouses_tables.up.sql`:[^1]

```sql
CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT UNIQUE NOT NULL, -- e.g., HN-01, HCM-02, DN-01
    
    -- Address
    address TEXT NOT NULL,
    province TEXT NOT NULL,
    district TEXT,
    ward TEXT,
    
    -- Geolocation for nearest warehouse calculation
    latitude DECIMAL(9,6),
    longitude DECIMAL(9,6),
    
    -- Contact
    phone TEXT,
    email TEXT,
    manager_name TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_warehouses_code ON warehouses(code);
CREATE INDEX idx_warehouses_active ON warehouses(is_active) WHERE is_active = true;
CREATE INDEX idx_warehouses_province ON warehouses(province);
CREATE INDEX idx_warehouses_location ON warehouses(latitude, longitude) WHERE latitude IS NOT NULL;

-- Trigger auto update updated_at
CREATE TRIGGER update_warehouses_updated_at
    BEFORE UPDATE ON warehouses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```


#### 1.2 Warehouse_Inventory Table Migration

```sql
CREATE TABLE warehouse_inventory (
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    
    -- Stock tracking
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved INT NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    
    -- Alerts
    alert_threshold INT DEFAULT 10,
    
    -- Metadata
    last_restocked_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (warehouse_id, book_id),
    
    -- Constraint: available stock must be non-negative
    CONSTRAINT available_stock CHECK (quantity >= reserved)
);

-- Indexes
CREATE INDEX idx_inventory_book ON warehouse_inventory(book_id);
CREATE INDEX idx_inventory_warehouse ON warehouse_inventory(warehouse_id);
CREATE INDEX idx_inventory_low_stock ON warehouse_inventory(warehouse_id, book_id, quantity) 
    WHERE quantity <= alert_threshold;
CREATE INDEX idx_inventory_available ON warehouse_inventory(book_id, quantity, reserved) 
    WHERE quantity > reserved;

-- Trigger auto update updated_at
CREATE TRIGGER update_warehouse_inventory_updated_at
    BEFORE UPDATE ON warehouse_inventory
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```


#### 1.3 Views for Aggregated Data

```sql
-- Total stock across all warehouses
CREATE VIEW books_total_stock AS
SELECT 
    book_id,
    SUM(quantity) as total_quantity,
    SUM(reserved) as total_reserved,
    SUM(quantity - reserved) as available
FROM warehouse_inventory
GROUP BY book_id;

-- Warehouse stock summary
CREATE VIEW warehouse_stock_summary AS
SELECT 
    w.id as warehouse_id,
    w.name as warehouse_name,
    w.code as warehouse_code,
    COUNT(DISTINCT wi.book_id) as unique_books,
    SUM(wi.quantity) as total_quantity,
    SUM(wi.reserved) as total_reserved,
    SUM(wi.quantity - wi.reserved) as available
FROM warehouses w
LEFT JOIN warehouse_inventory wi ON w.id = wi.warehouse_id
GROUP BY w.id, w.name, w.code;

-- Low stock items
CREATE VIEW low_stock_items AS
SELECT 
    w.name as warehouse_name,
    w.code as warehouse_code,
    b.title as book_title,
    b.id as book_id,
    wi.quantity,
    wi.reserved,
    wi.alert_threshold,
    (wi.quantity - wi.reserved) as available
FROM warehouse_inventory wi
JOIN warehouses w ON wi.warehouse_id = w.id
JOIN books b ON wi.book_id = b.id
WHERE wi.quantity <= wi.alert_threshold
ORDER BY wi.quantity ASC;
```


#### 1.4 Update Orders Table

```sql
-- Add warehouse_id to orders table
ALTER TABLE orders 
ADD COLUMN IF NOT EXISTS warehouse_id UUID REFERENCES warehouses(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_orders_warehouse ON orders(warehouse_id) 
    WHERE warehouse_id IS NOT NULL;
```


### Công việc cụ thể

#### 1.5 Domain Models

Tạo file `internal/domains/warehouse/model/warehouse.go`:[^1]

```go
package model

import "time"

type Warehouse struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Code        string    `json:"code"`
    Address     string    `json:"address"`
    Province    string    `json:"province"`
    District    *string   `json:"district,omitempty"`
    Ward        *string   `json:"ward,omitempty"`
    Latitude    *float64  `json:"latitude,omitempty"`
    Longitude   *float64  `json:"longitude,omitempty"`
    Phone       *string   `json:"phone,omitempty"`
    Email       *string   `json:"email,omitempty"`
    ManagerName *string   `json:"manager_name,omitempty"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type WarehouseInventory struct {
    WarehouseID     string     `json:"warehouse_id"`
    BookID          string     `json:"book_id"`
    Quantity        int        `json:"quantity"`
    Reserved        int        `json:"reserved"`
    Available       int        `json:"available"` // Calculated: quantity - reserved
    AlertThreshold  int        `json:"alert_threshold"`
    LastRestockedAt *time.Time `json:"last_restocked_at,omitempty"`
    UpdatedAt       time.Time  `json:"updated_at"`
    
    // Joined fields
    WarehouseName   string     `json:"warehouse_name,omitempty"`
    BookTitle       string     `json:"book_title,omitempty"`
}

type InventoryAdjustment struct {
    ID           string    `json:"id"`
    WarehouseID  string    `json:"warehouse_id"`
    BookID       string    `json:"book_id"`
    OldQuantity  int       `json:"old_quantity"`
    NewQuantity  int       `json:"new_quantity"`
    Difference   int       `json:"difference"`
    Reason       string    `json:"reason"`
    AdjustedBy   string    `json:"adjusted_by"` // User ID
    CreatedAt    time.Time `json:"created_at"`
}
```


#### 1.6 Seed Data

Tạo file `seeds/004_warehouses_seed.sql`:[^1]

```sql
-- Insert warehouses
INSERT INTO warehouses (code, name, address, province, latitude, longitude, phone, is_active)
VALUES
-- Hanoi warehouse
('HN-01', 'Kho Hà Nội - Trung Tâm', 
 '123 Đường Láng, Đống Đa, Hà Nội', 
 'Hà Nội',
 21.0285, 105.8542,
 '0243-123-4567',
 true),

-- Ho Chi Minh warehouse
('HCM-01', 'Kho TP.HCM - Quận 1',
 '456 Nguyễn Huệ, Quận 1, TP. Hồ Chí Minh',
 'Hồ Chí Minh',
 10.7769, 106.7009,
 '028-987-6543',
 true),

-- Da Nang warehouse
('DN-01', 'Kho Đà Nẵng',
 '789 Trần Phú, Hải Châu, Đà Nẵng',
 'Đà Nẵng',
 16.0544, 108.2022,
 '0236-555-0123',
 true),

-- Can Tho warehouse
('CT-01', 'Kho Cần Thơ - Miền Tây',
 '321 Mậu Thân, Ninh Kiều, Cần Thơ',
 'Cần Thơ',
 10.0452, 105.7469,
 '0292-333-4444',
 true);

-- Sample inventory distribution
-- (Will be populated by migration script P3-T002)
```


### Acceptance Criteria

- Migration tạo bảng warehouses và warehouse_inventory thành công[^1]
- Views tạo đúng để aggregate data[^1]
- Constraints đảm bảo quantity >= reserved[^1]
- Indexes optimize queries[^1]
- Seed data insert 4 warehouses ở các khu vực chính[^1]


### Dependencies

- P1-T002: Database setup[^1]
- P1-T003: Core tables (books, orders)[^1]


### Effort

1 ngày[^1]

***

## 2. Migrate Existing Stock to Warehouse Model (P3-T002)

### Mô tả

Migration script để migrate existing inventory data từ bảng cũ (nếu có) sang warehouse_inventory.[^1]

### Giả định

- Hiện tại có bảng `inventory` hoặc field `stock` trong `books` table[^1]
- Tất cả stock sẽ được assign vào 1 warehouse mặc định (HN-01)[^1]


### Công việc cụ thể

#### 2.1 Migration Script

Tạo file `migrations/000026_migrate_stock_to_warehouse.up.sql`:[^1]

```sql
-- Step 1: Get default warehouse (HN-01)
DO $$
DECLARE
    default_warehouse_id UUID;
BEGIN
    SELECT id INTO default_warehouse_id 
    FROM warehouses 
    WHERE code = 'HN-01' 
    LIMIT 1;
    
    IF default_warehouse_id IS NULL THEN
        RAISE EXCEPTION 'Default warehouse HN-01 not found. Please run warehouses seed first.';
    END IF;
    
    -- Step 2: Migrate from old inventory table (if exists)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'inventory') THEN
        INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold)
        SELECT 
            default_warehouse_id,
            book_id,
            COALESCE(quantity, 0),
            COALESCE(reserved, 0),
            10 -- Default alert threshold
        FROM inventory
        WHERE quantity > 0
        ON CONFLICT (warehouse_id, book_id) DO NOTHING;
        
        RAISE NOTICE 'Migrated inventory from inventory table';
    END IF;
    
    -- Step 3: Migrate from books.stock field (if exists)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'books' AND column_name = 'stock'
    ) THEN
        INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold)
        SELECT 
            default_warehouse_id,
            id,
            COALESCE(stock, 0),
            0, -- No reserved initially
            10
        FROM books
        WHERE COALESCE(stock, 0) > 0
        ON CONFLICT (warehouse_id, book_id) DO NOTHING;
        
        RAISE NOTICE 'Migrated stock from books table';
    END IF;
    
    -- Step 4: For books without inventory, create zero stock records
    INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold)
    SELECT 
        default_warehouse_id,
        b.id,
        0,
        0,
        10
    FROM books b
    WHERE NOT EXISTS (
        SELECT 1 FROM warehouse_inventory wi 
        WHERE wi.book_id = b.id AND wi.warehouse_id = default_warehouse_id
    )
    AND b.is_active = true
    AND b.deleted_at IS NULL;
    
    RAISE NOTICE 'Created zero stock records for remaining books';
END $$;

-- Step 5: Drop old inventory structures (optional, comment out if keeping for backup)
-- DROP TABLE IF EXISTS inventory;
-- ALTER TABLE books DROP COLUMN IF EXISTS stock;
```


#### 2.2 Verification Script

Tạo file `scripts/verify_inventory_migration.sql`:[^1]

```sql
-- Check total books vs inventory records
SELECT 
    'Total active books' as metric,
    COUNT(*) as count
FROM books 
WHERE is_active = true AND deleted_at IS NULL
UNION ALL
SELECT 
    'Books with inventory',
    COUNT(DISTINCT book_id)
FROM warehouse_inventory
UNION ALL
SELECT
    'Total inventory records',
    COUNT(*)
FROM warehouse_inventory
UNION ALL
SELECT
    'Total stock quantity',
    COALESCE(SUM(quantity), 0)
FROM warehouse_inventory;

-- Check for negative stocks (should be 0)
SELECT COUNT(*) as negative_stock_count
FROM warehouse_inventory
WHERE quantity < 0 OR reserved < 0 OR quantity < reserved;

-- Low stock items
SELECT 
    w.name as warehouse,
    b.title as book,
    wi.quantity,
    wi.reserved,
    wi.quantity - wi.reserved as available
FROM warehouse_inventory wi
JOIN warehouses w ON wi.warehouse_id = w.id
JOIN books b ON wi.book_id = b.id
WHERE wi.quantity <= wi.alert_threshold
ORDER BY wi.quantity ASC
LIMIT 20;
```


#### 2.3 Go Migration Runner

Tạo file `cmd/migrate/migrate_inventory.go`:[^1]

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "database/sql"
)

type InventoryMigration struct {
    db *sql.DB
}

func (m *InventoryMigration) Run(ctx context.Context) error {
    log.Println("Starting inventory migration...")
    
    // 1. Validate warehouses exist
    var warehouseCount int
    err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM warehouses WHERE is_active = true").Scan(&warehouseCount)
    if err != nil {
        return fmt.Errorf("failed to count warehouses: %w", err)
    }
    
    if warehouseCount == 0 {
        return fmt.Errorf("no active warehouses found. Please seed warehouses first")
    }
    
    log.Printf("Found %d active warehouse(s)", warehouseCount)
    
    // 2. Get statistics before migration
    var oldStockCount int
    err = m.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(stock), 0) FROM books 
        WHERE stock IS NOT NULL AND is_active = true
    `).Scan(&oldStockCount)
    
    log.Printf("Old stock count: %d", oldStockCount)
    
    // 3. Run migration
    _, err = m.db.ExecContext(ctx, migrationSQL)
    if err != nil {
        return fmt.Errorf("migration failed: %w", err)
    }
    
    // 4. Verify results
    var newStockCount int
    err = m.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(quantity), 0) FROM warehouse_inventory
    `).Scan(&newStockCount)
    
    log.Printf("New stock count: %d", newStockCount)
    
    if oldStockCount != newStockCount {
        log.Printf("WARNING: Stock mismatch. Old: %d, New: %d", oldStockCount, newStockCount)
    } else {
        log.Println("✓ Stock counts match")
    }
    
    // 5. Check constraints
    var violationCount int
    err = m.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM warehouse_inventory 
        WHERE quantity < reserved OR quantity < 0 OR reserved < 0
    `).Scan(&violationCount)
    
    if violationCount > 0 {
        return fmt.Errorf("found %d constraint violations", violationCount)
    }
    
    log.Println("✓ All constraints satisfied")
    log.Println("Migration completed successfully!")
    
    return nil
}
```


### Acceptance Criteria

- Migration script chạy thành công[^1]
- Tất cả existing stock được migrate vào warehouse_inventory[^1]
- Total stock count trước và sau migration bằng nhau[^1]
- Không có constraint violations[^1]
- Books không có stock được tạo record với quantity = 0[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]


### Effort

1 ngày[^1]

***

## 3. Warehouse Selection Algorithm (Nearest) (P3-T003)

### Mô tả

Implement thuật toán chọn warehouse gần nhất dựa trên delivery address để optimize shipping.[^1]

### Business Logic

1. Tính khoảng cách từ delivery address tới tất cả warehouses có stock[^1]
2. Chọn warehouse gần nhất có đủ stock[^1]
3. Nếu không có warehouse nào có đủ, split order hoặc chọn warehouse có stock nhiều nhất[^1]

### Công việc cụ thể

#### 3.1 Geocoding Service (Get Coordinates from Address)

Tạo file `internal/infrastructure/geocoding/service.go`:[^1]

```go
package geocoding

import (
    "context"
    "fmt"
)

type GeocodingService struct {
    // Can use external APIs: Google Maps, OpenStreetMap, etc.
    // For MVP, use hardcoded province coordinates
}

// Province center coordinates (Vietnam)
var ProvinceCoordinates = map[string]struct{ Lat, Lng float64 }{
    "Hà Nội":        {21.0285, 105.8542},
    "Hồ Chí Minh":   {10.7769, 106.7009},
    "Đà Nẵng":       {16.0544, 108.2022},
    "Hải Phòng":     {20.8449, 106.6881},
    "Cần Thơ":       {10.0452, 105.7469},
    "An Giang":      {10.5215, 105.1258},
    "Bà Rịa-Vũng Tàu": {10.5417, 107.2429},
    // ... add more provinces
}

func (s *GeocodingService) GetCoordinates(ctx context.Context, province string) (lat, lng float64, err error) {
    coords, ok := ProvinceCoordinates[province]
    if !ok {
        return 0, 0, fmt.Errorf("province not found: %s", province)
    }
    
    return coords.Lat, coords.Lng, nil
}

// Calculate Haversine distance (in kilometers)
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
    const earthRadius = 6371.0 // km
    
    dLat := toRadians(lat2 - lat1)
    dLng := toRadians(lng2 - lng1)
    
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
         math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
         math.Sin(dLng/2)*math.Sin(dLng/2)
    
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    
    return earthRadius * c
}

func toRadians(deg float64) float64 {
    return deg * math.Pi / 180
}
```


#### 3.2 Warehouse Selection Service

Tạo file `internal/domains/warehouse/service/selection_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
    "sort"
)

type WarehouseSelectionService struct {
    warehouseRepo  *repository.WarehouseRepository
    inventoryRepo  *repository.InventoryRepository
    geocodingService *geocoding.GeocodingService
}

type WarehouseCandidate struct {
    Warehouse *Warehouse
    Distance  float64
    HasStock  map[string]int // book_id -> available quantity
}

func (s *WarehouseSelectionService) SelectWarehouse(
    ctx context.Context, 
    deliveryProvince string,
    items []OrderItem, // {BookID, Quantity}
) (*Warehouse, error) {
    
    // 1. Get delivery coordinates
    deliveryLat, deliveryLng, err := s.geocodingService.GetCoordinates(ctx, deliveryProvince)
    if err != nil {
        return nil, fmt.Errorf("failed to get delivery coordinates: %w", err)
    }
    
    // 2. Get all active warehouses
    warehouses, err := s.warehouseRepo.FindAllActive(ctx)
    if err != nil {
        return nil, err
    }
    
    if len(warehouses) == 0 {
        return nil, fmt.Errorf("no active warehouses available")
    }
    
    // 3. Get book IDs from items
    bookIDs := make([]string, len(items))
    requiredQty := make(map[string]int)
    for i, item := range items {
        bookIDs[i] = item.BookID
        requiredQty[item.BookID] = item.Quantity
    }
    
    // 4. Check inventory for each warehouse
    candidates := []WarehouseCandidate{}
    
    for _, warehouse := range warehouses {
        if warehouse.Latitude == nil || warehouse.Longitude == nil {
            continue // Skip warehouses without coordinates
        }
        
        // Calculate distance
        distance := geocoding.HaversineDistance(
            deliveryLat, deliveryLng,
            *warehouse.Latitude, *warehouse.Longitude,
        )
        
        // Check stock availability
        inventory, err := s.inventoryRepo.GetByWarehouseAndBooks(ctx, warehouse.ID, bookIDs)
        if err != nil {
            continue
        }
        
        hasStock := make(map[string]int)
        canFulfill := true
        
        for _, item := range items {
            inv, ok := inventory[item.BookID]
            if !ok || inv.Available < item.Quantity {
                canFulfill = false
                break
            }
            hasStock[item.BookID] = inv.Available
        }
        
        if canFulfill {
            candidates = append(candidates, WarehouseCandidate{
                Warehouse: warehouse,
                Distance:  distance,
                HasStock:  hasStock,
            })
        }
    }
    
    // 5. No warehouse can fulfill all items
    if len(candidates) == 0 {
        return nil, fmt.Errorf("insufficient stock across all warehouses")
    }
    
    // 6. Sort by distance (ascending)
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Distance < candidates[j].Distance
    })
    
    // 7. Return nearest warehouse
    selected := candidates[^0]
    
    log.Info("Selected warehouse",
        "warehouse", selected.Warehouse.Code,
        "distance_km", fmt.Sprintf("%.2f", selected.Distance),
        "delivery_province", deliveryProvince,
    )
    
    return selected.Warehouse, nil
}

// Alternative: Select multiple warehouses for split fulfillment
func (s *WarehouseSelectionService) SelectMultipleWarehouses(
    ctx context.Context,
    deliveryProvince string,
    items []OrderItem,
) ([]WarehouseFulfillment, error) {
    // For Phase 4: Split order across warehouses
    // Not implemented in MVP
    return nil, fmt.Errorf("not implemented")
}
```


#### 3.3 Repository Methods

```go
func (r *WarehouseRepository) FindAllActive(ctx context.Context) ([]Warehouse, error) {
    query := `
        SELECT id, name, code, address, province, latitude, longitude, is_active
        FROM warehouses
        WHERE is_active = true
        ORDER BY name
    `
    
    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    warehouses := []Warehouse{}
    for rows.Next() {
        var w Warehouse
        err := rows.Scan(
            &w.ID, &w.Name, &w.Code, &w.Address, &w.Province,
            &w.Latitude, &w.Longitude, &w.IsActive,
        )
        if err != nil {
            return nil, err
        }
        warehouses = append(warehouses, w)
    }
    
    return warehouses, nil
}

func (r *InventoryRepository) GetByWarehouseAndBooks(
    ctx context.Context,
    warehouseID string,
    bookIDs []string,
) (map[string]*WarehouseInventory, error) {
    query := `
        SELECT 
            book_id,
            quantity,
            reserved,
            (quantity - reserved) as available
        FROM warehouse_inventory
        WHERE warehouse_id = $1
        AND book_id = ANY($2)
    `
    
    rows, err := r.db.QueryContext(ctx, query, warehouseID, pq.Array(bookIDs))
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    inventory := make(map[string]*WarehouseInventory)
    for rows.Next() {
        var inv WarehouseInventory
        err := rows.Scan(&inv.BookID, &inv.Quantity, &inv.Reserved, &inv.Available)
        if err != nil {
            return nil, err
        }
        inventory[inv.BookID] = &inv
    }
    
    return inventory, nil
}
```


#### 3.4 Unit Tests

```go
func TestWarehouseSelection(t *testing.T) {
    service := NewWarehouseSelectionService(...)
    
    // Test case 1: Single warehouse can fulfill
    items := []OrderItem{
        {BookID: "book1", Quantity: 2},
        {BookID: "book2", Quantity: 1},
    }
    
    warehouse, err := service.SelectWarehouse(context.Background(), "Hà Nội", items)
    assert.NoError(t, err)
    assert.NotNil(t, warehouse)
    assert.Equal(t, "HN-01", warehouse.Code)
    
    // Test case 2: Choose nearest warehouse
    warehouse, err = service.SelectWarehouse(context.Background(), "Đà Nẵng", items)
    assert.NoError(t, err)
    assert.Equal(t, "DN-01", warehouse.Code) // Đà Nẵng warehouse should be nearest
    
    // Test case 3: Insufficient stock
    largeItems := []OrderItem{
        {BookID: "book1", Quantity: 1000},
    }
    
    _, err = service.SelectWarehouse(context.Background(), "Hà Nội", largeItems)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "insufficient stock")
}
```


### Acceptance Criteria

- Algorithm chọn được warehouse gần nhất có stock[^1]
- Distance calculation chính xác (Haversine formula)[^1]
- Fallback khi không có warehouse nào có đủ stock[^1]
- Performance: < 100ms để select warehouse[^1]
- Unit tests coverage > 80%[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]
- P3-T002: Inventory migration[^1]


### Effort

2 ngày[^1]

***

## 4. Update Order Flow với Warehouse_ID (P3-T004)

### Mô tả

Update order creation flow để include warehouse selection và assign warehouse_id.[^1]

### Changes to Order Flow

1. **Before**: Order → Reserve inventory (generic)[^1]
2. **After**: Order → Select warehouse → Reserve inventory at warehouse → Assign warehouse_id[^1]

### Công việc cụ thể

#### 4.1 Update Order Service

```go
func (s *OrderService) CreateOrder(ctx context.Context, params CreateOrderParams) (*Order, error) {
    // ... existing validation
    
    // NEW: Get delivery address
    address, err := s.addressRepo.FindByID(ctx, params.AddressID)
    if err != nil {
        return nil, fmt.Errorf("address not found")
    }
    
    // NEW: Select warehouse based on address
    warehouse, err := s.warehouseSelectionService.SelectWarehouse(
        ctx,
        address.Province,
        cartItems, // Convert from cart items
    )
    if err != nil {
        return nil, fmt.Errorf("warehouse selection failed: %w", err)
    }
    
    log.Info("Selected warehouse for order",
        "warehouse_code", warehouse.Code,
        "province", address.Province,
    )
    
    // Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // Create order WITH warehouse_id
    order := &Order{
        UserID:        params.UserID,
        AddressID:     params.AddressID,
        WarehouseID:   &warehouse.ID, // NEW
        Subtotal:      subtotal,
        Total:         total,
        Status:        "pending",
        PaymentMethod: params.PaymentMethod,
        PaymentStatus: "pending",
    }
    
    err = s.orderRepo.CreateTx(ctx, tx, order)
    if err != nil {
        return nil, err
    }
    
    // Reserve inventory AT THE SELECTED WAREHOUSE
    err = s.inventoryService.ReserveInventoryTx(
        ctx, tx,
        warehouse.ID, // NEW: warehouse-specific
        order.ID,
        cartItems,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to reserve inventory: %w", err)
    }
    
    // ... rest of order creation
    
    return order, tx.Commit()
}
```


#### 4.2 Update Inventory Service

```go
type InventoryService struct {
    inventoryRepo *repository.InventoryRepository
}

func (s *InventoryService) ReserveInventoryTx(
    ctx context.Context,
    tx *sql.Tx,
    warehouseID string,
    orderID string,
    items []OrderItem,
) error {
    
    for _, item := range items {
        // Lock row for update (pessimistic lock)
        query := `
            SELECT quantity, reserved
            FROM warehouse_inventory
            WHERE warehouse_id = $1 AND book_id = $2
            FOR UPDATE
        `
        
        var quantity, reserved int
        err := tx.QueryRowContext(ctx, query, warehouseID, item.BookID).Scan(&quantity, &reserved)
        if err != nil {
            return fmt.Errorf("inventory not found for book %s", item.BookID)
        }
        
        // Check available stock
        available := quantity - reserved
        if available < item.Quantity {
            return fmt.Errorf("insufficient stock for book %s: need %d, available %d",
                item.BookID, item.Quantity, available)
        }
        
        // Update reserved quantity
        updateQuery := `
            UPDATE warehouse_inventory
            SET reserved = reserved + $1,
                updated_at = NOW()
            WHERE warehouse_id = $2 AND book_id = $3
        `
        
        _, err = tx.ExecContext(ctx, updateQuery, item.Quantity, warehouseID, item.BookID)
        if err != nil {
            return err
        }
        
        log.Info("Reserved inventory",
            "warehouse_id", warehouseID,
            "book_id", item.BookID,
            "quantity", item.Quantity,
        )
    }
    
    return nil
}

func (s *InventoryService) ReleaseReservationTx(
    ctx context.Context,
    tx *sql.Tx,
    orderID string,
) error {
    // Get order warehouse and items
    order, err := s.orderRepo.FindByIDWithItemsTx(ctx, tx, orderID)
    if err != nil {
        return err
    }
    
    if order.WarehouseID == nil {
        return fmt.Errorf("order has no warehouse assigned")
    }
    
    // Release reserved stock
    for _, item := range order.Items {
        query := `
            UPDATE warehouse_inventory
            SET reserved = GREATEST(reserved - $1, 0),
                updated_at = NOW()
            WHERE warehouse_id = $2 AND book_id = $3
        `
        
        _, err := tx.ExecContext(ctx, query, item.Quantity, *order.WarehouseID, item.BookID)
        if err != nil {
            return err
        }
    }
    
    log.Info("Released inventory reservation",
        "order_id", orderID,
        "warehouse_id", *order.WarehouseID,
    )
    
    return nil
}

func (s *InventoryService) ConfirmInventoryTx(
    ctx context.Context,
    tx *sql.Tx,
    orderID string,
) error {
    // Get order
    order, err := s.orderRepo.FindByIDWithItemsTx(ctx, tx, orderID)
    if err != nil {
        return err
    }
    
    if order.WarehouseID == nil {
        return fmt.Errorf("order has no warehouse assigned")
    }
    
    // Deduct actual quantity and release reservation
    for _, item := range order.Items {
        query := `
            UPDATE warehouse_inventory
            SET quantity = quantity - $1,
                reserved = reserved - $1,
                updated_at = NOW()
            WHERE warehouse_id = $2 AND book_id = $3
        `
        
        result, err := tx.ExecContext(ctx, query, item.Quantity, *order.WarehouseID, item.BookID)
        if err != nil {
            return err
        }
        
        rowsAffected, _ := result.RowsAffected()
        if rowsAffected == 0 {
            return fmt.Errorf("failed to deduct inventory for book %s", item.BookID)
        }
    }
    
    log.Info("Confirmed inventory deduction",
        "order_id", orderID,
        "warehouse_id", *order.WarehouseID,
    )
    
    return nil
}
```


#### 4.3 Update Order Status Handlers

```go
// When admin confirms order
func (h *AdminOrderHandler) ConfirmOrder(c *gin.Context) {
    orderID := c.Param("id")
    
    err := h.orderService.ConfirmOrder(c.Request.Context(), orderID)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Order confirmed, inventory deducted"})
}

// When order cancelled
func (h *OrderHandler) CancelOrder(c *gin.Context) {
    orderID := c.Param("id")
    userID := c.GetString("user_id")
    
    err := h.orderService.CancelOrder(c.Request.Context(), orderID, userID)
    if err != nil {
        c.JSON(400, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "Order cancelled, inventory released"})
}
```


#### 4.4 Integration Test

```go
func TestOrderFlowWithWarehouse(t *testing.T) {
    env := SetupTestEnv(t)
    defer env.Teardown(t)
    
    // Seed data
    user := SeedUser(env.DB)
    warehouse := SeedWarehouse(env.DB, "HN-01", "Hà Nội", 21.0285, 105.8542)
    book := SeedBook(env.DB, "Book 1")
    SeedInventory(env.DB, warehouse.ID, book.ID, 10) // 10 copies in stock
    address := SeedAddress(env.DB, user.ID, "Hà Nội")
    
    orderService := service.NewOrderService(...)
    
    ctx := context.Background()
    
    // Create order
    order, err := orderService.CreateOrder(ctx, CreateOrderParams{
        UserID:        user.ID,
        AddressID:     address.ID,
        PaymentMethod: "cod",
        Items: []OrderItem{
            {BookID: book.ID, Quantity: 2},
        },
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, order.WarehouseID)
    assert.Equal(t, warehouse.ID, *order.WarehouseID)
    
    // Verify inventory reserved
    inv, _ := inventoryService.GetInventory(ctx, warehouse.ID, book.ID)
    assert.Equal(t, 10, inv.Quantity)
    assert.Equal(t, 2, inv.Reserved)
    assert.Equal(t, 8, inv.Available)
    
    // Confirm order
    err = orderService.ConfirmOrder(ctx, order.ID)
    assert.NoError(t, err)
    
    // Verify inventory deducted
    inv, _ = inventoryService.GetInventory(ctx, warehouse.ID, book.ID)
    assert.Equal(t, 8, inv.Quantity)
    assert.Equal(t, 0, inv.Reserved)
    assert.Equal(t, 8, inv.Available)
}
```


### Acceptance Criteria

- Order creation tự động select warehouse gần nhất[^1]
- Warehouse_id được lưu trong orders table[^1]
- Inventory reservation specific to warehouse[^1]
- Transaction rollback nếu inventory không đủ[^1]
- Release reservation khi order cancelled[^1]
- Confirm deduction khi order confirmed[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]
- P3-T003: Warehouse selection algorithm[^1]
- P1-T025: Order creation API[^1]


### Effort

2 ngày[^1]

***

## 5. Admin: Warehouse CRUD (P3-T005)

### Mô tả

Admin panel APIs để quản lý warehouses.[^1]

### API Endpoints

- `GET /v1/admin/warehouses` - List warehouses[^1]
- `GET /v1/admin/warehouses/:id` - Get warehouse detail[^1]
- `POST /v1/admin/warehouses` - Create warehouse[^1]
- `PUT /v1/admin/warehouses/:id` - Update warehouse[^1]
- `DELETE /v1/admin/warehouses/:id` - Deactivate warehouse[^1]
- `GET /v1/admin/warehouses/:id/stats` - Warehouse statistics[^1]


### Công việc cụ thể

#### 5.1 Create Warehouse API

**Request Body**:

```json
{
  "name": "Kho Hải Phòng",
  "code": "HP-01",
  "address": "789 Lạch Tray, Ngô Quyền, Hải Phòng",
  "province": "Hải Phòng",
  "district": "Ngô Quyền",
  "latitude": 20.8449,
  "longitude": 106.6881,
  "phone": "0225-123-4567",
  "email": "warehouse.hp@bookstore.com",
  "manager_name": "Nguyễn Văn A"
}
```

**Service**:

```go
func (s *WarehouseService) CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (*Warehouse, error) {
    // 1. Validate code uniqueness
    exists, err := s.warehouseRepo.ExistsByCode(ctx, req.Code)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, fmt.Errorf("warehouse code already exists: %s", req.Code)
    }
    
    // 2. Validate coordinates (if provided)
    if req.Latitude != nil && req.Longitude != nil {
        if *req.Latitude < -90 || *req.Latitude > 90 {
            return nil, fmt.Errorf("invalid latitude: must be between -90 and 90")
        }
        if *req.Longitude < -180 || *req.Longitude > 180 {
            return nil, fmt.Errorf("invalid longitude: must be between -180 and 180")
        }
    }
    
    // 3. Create warehouse
    warehouse := &Warehouse{
        Name:        req.Name,
        Code:        req.Code,
        Address:     req.Address,
        Province:    req.Province,
        District:    req.District,
        Ward:        req.Ward,
        Latitude:    req.Latitude,
        Longitude:   req.Longitude,
        Phone:       req.Phone,
        Email:       req.Email,
        ManagerName: req.ManagerName,
        IsActive:    true,
    }
    
    err = s.warehouseRepo.Create(ctx, warehouse)
    if err != nil {
        return nil, err
    }
    
    log.Info("Warehouse created",
        "code", warehouse.Code,
        "name", warehouse.Name,
    )
    
    return warehouse, nil
}
```

**Validation**:

```go
func (r CreateWarehouseRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Name, 
            validation.Required,
            validation.Length(1, 200)),
        validation.Field(&r.Code,
            validation.Required,
            validation.Match(regexp.MustCompile(`^[A-Z]{2,3}-\d{2}$`))), // e.g., HN-01, HCM-01
        validation.Field(&r.Address,
            validation.Required,
            validation.Length(1, 500)),
        validation.Field(&r.Province,
            validation.Required),
        validation.Field(&r.Latitude,
            validation.When(r.Latitude != nil, 
                validation.Min(-90.0),
                validation.Max(90.0))),
        validation.Field(&r.Longitude,
            validation.When(r.Longitude != nil,
                validation.Min(-180.0),
                validation.Max(180.0))),
    )
}
```


#### 5.2 List Warehouses API

```go
func (s *WarehouseService) ListWarehouses(ctx context.Context, filters WarehouseFilters) (*WarehouseListResult, error) {
    query := `
        SELECT 
            w.*,
            COUNT(DISTINCT wi.book_id) as unique_books,
            COALESCE(SUM(wi.quantity), 0) as total_quantity
        FROM warehouses w
        LEFT JOIN warehouse_inventory wi ON w.id = wi.warehouse_id
        WHERE 1=1
    `
    
    args := []interface{}{}
    argPos := 1
    
    if filters.Province != nil {
        query += fmt.Sprintf(" AND w.province = $%d", argPos)
        args = append(args, *filters.Province)
        argPos++
    }
    
    if filters.IsActive != nil {
        query += fmt.Sprintf(" AND w.is_active = $%d", argPos)
        args = append(args, *filters.IsActive)
        argPos++
    }
    
    query += " GROUP BY w.id ORDER BY w.name"
    
    // Pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
    args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
    
    // Execute...
    rows, err := s.db.QueryContext(ctx, query, args...)
    // ... scan and return
    
    return &WarehouseListResult{Warehouses: warehouses, Total: total}, nil
}
```


#### 5.3 Warehouse Statistics API

`GET /v1/admin/warehouses/:id/stats`

```go
func (s *WarehouseService) GetWarehouseStats(ctx context.Context, warehouseID string) (*WarehouseStats, error) {
    stats := &WarehouseStats{WarehouseID: warehouseID}
    
    // Total unique books
    err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(DISTINCT book_id) FROM warehouse_inventory WHERE warehouse_id = $1
    `, warehouseID).Scan(&stats.UniqueBooks)
    
    // Total quantity
    err = s.db.QueryRowContext(ctx, `
        SELECT 
            COALESCE(SUM(quantity), 0),
            COALESCE(SUM(reserved), 0),
            COALESCE(SUM(quantity - reserved), 0)
        FROM warehouse_inventory 
        WHERE warehouse_id = $1
    `, warehouseID).Scan(&stats.TotalQuantity, &stats.TotalReserved, &stats.TotalAvailable)
    
    // Low stock items
    err = s.db.QueryRowContext(ctx, `
        SELECT COUNT(*) 
        FROM warehouse_inventory 
        WHERE warehouse_id = $1 AND quantity <= alert_threshold
    `, warehouseID).Scan(&stats.LowStockItems)
    
    // Orders fulfilled from this warehouse (last 30 days)
    err = s.db.QueryRowContext(ctx, `
        SELECT COUNT(*) 
        FROM orders 
        WHERE warehouse_id = $1 
        AND created_at >= NOW() - INTERVAL '30 days'
    `, warehouseID).Scan(&stats.OrdersLast30Days)
    
    return stats, nil
}
```

**Response**:

```json
{
  "success": true,
  "data": {
    "warehouse_id": "uuid",
    "unique_books": 1250,
    "total_quantity": 15000,
    "total_reserved": 450,
    "total_available": 14550,
    "low_stock_items": 23,
    "orders_last_30_days": 456
  }
}
```


### Acceptance Criteria

- Admin tạo được warehouse với validation[^1]
- Code format validation (e.g., HN-01)[^1]
- List warehouses với pagination[^1]
- Warehouse stats API trả về metrics[^1]
- Update và deactivate warehouse[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]
- P1-T029: RBAC middleware[^1]


### Effort

2 ngày[^1]

***

## 6. Admin: Inventory Management Per Warehouse (P3-T006)

### Mô tả

Admin APIs để quản lý inventory cho từng warehouse.[^1]

### API Endpoints

- `GET /v1/admin/inventory?warehouse_id=xxx&book_id=xxx` - List inventory[^1]
- `GET /v1/admin/inventory/warehouse/:warehouse_id/book/:book_id` - Get specific inventory[^1]
- `PATCH /v1/admin/inventory/warehouse/:warehouse_id/book/:book_id` - Adjust inventory[^1]
- `POST /v1/admin/inventory/bulk-update` - Bulk adjustment (CSV)[^1]
- `GET /v1/admin/inventory/low-stock` - Low stock alerts[^1]
- `GET /v1/admin/inventory/movements/:book_id` - Inventory movement history[^1]


### Công việc cụ thể

#### 6.1 Adjust Inventory API

`PATCH /v1/admin/inventory/warehouse/:warehouse_id/book/:book_id`

**Request Body**:

```json
{
  "adjustment_type": "add", // or "set", "subtract"
  "quantity": 50,
  "reason": "Restocking from supplier"
}
```

**Service**:

```go
func (s *InventoryService) AdjustInventory(
    ctx context.Context,
    warehouseID string,
    bookID string,
    adjustment InventoryAdjustment,
    adjustedBy string,
) error {
    
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Lock inventory row
    var oldQuantity, reserved int
    err = tx.QueryRowContext(ctx, `
        SELECT quantity, reserved
        FROM warehouse_inventory
        WHERE warehouse_id = $1 AND book_id = $2
        FOR UPDATE
    `, warehouseID, bookID).Scan(&oldQuantity, &reserved)
    
    if err == sql.ErrNoRows {
        // Create new inventory record if not exists
        oldQuantity = 0
        reserved = 0
    } else if err != nil {
        return err
    }
    
    // Calculate new quantity
    var newQuantity int
    switch adjustment.AdjustmentType {
    case "add":
        newQuantity = oldQuantity + adjustment.Quantity
    case "subtract":
        newQuantity = oldQuantity - adjustment.Quantity
        if newQuantity < reserved {
            return fmt.Errorf("cannot subtract: new quantity (%d) would be less than reserved (%d)",
                newQuantity, reserved)
        }
    case "set":
        newQuantity = adjustment.Quantity
        if newQuantity < reserved {
            return fmt.Errorf("cannot set quantity: new quantity (%d) would be less than reserved (%d)",
                newQuantity, reserved)
        }
    default:
        return fmt.Errorf("invalid adjustment type: %s", adjustment.AdjustmentType)
    }
    
    if newQuantity < 0 {
        return fmt.Errorf("quantity cannot be negative")
    }
    
    // Update inventory
    _, err = tx.ExecContext(ctx, `
        INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold, updated_at)
        VALUES ($1, $2, $3, $4, 10, NOW())
        ON CONFLICT (warehouse_id, book_id) 
        DO UPDATE SET 
            quantity = $3,
            last_restocked_at = CASE WHEN $3 > warehouse_inventory.quantity THEN NOW() ELSE warehouse_inventory.last_restocked_at END,
            updated_at = NOW()
    `, warehouseID, bookID, newQuantity, reserved)
    
    if err != nil {
        return err
    }
    
    // Log adjustment
    _, err = tx.ExecContext(ctx, `
        INSERT INTO inventory_adjustments 
        (warehouse_id, book_id, old_quantity, new_quantity, difference, reason, adjusted_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, warehouseID, bookID, oldQuantity, newQuantity, newQuantity-oldQuantity, adjustment.Reason, adjustedBy)
    
    if err != nil {
        return err
    }
    
    log.Info("Inventory adjusted",
        "warehouse_id", warehouseID,
        "book_id", bookID,
        "old_quantity", oldQuantity,
        "new_quantity", newQuantity,
        "reason", adjustment.Reason,
    )
    
    return tx.Commit()
}
```


#### 6.2 Bulk Inventory Update (CSV Import)

`POST /v1/admin/inventory/bulk-update`

**CSV Format**:

```csv
warehouse_code,book_isbn,quantity,reason
HN-01,978-604-2-29886-0,100,Initial stock
HN-01,978-604-2-10231-6,50,Restocking
HCM-01,978-604-2-29886-0,75,Transfer from HN
```

**Service**:

```go
func (s *InventoryService) BulkUpdate(ctx context.Context, file io.Reader, adjustedBy string) (*BulkUpdateResult, error) {
    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("failed to parse CSV: %w", err)
    }
    
    result := &BulkUpdateResult{
        TotalRows:    len(records) - 1, // Exclude header
        SuccessCount: 0,
        Errors:       []string{},
    }
    
    // Skip header
    for i, record := range records[1:] {
        lineNum := i + 2 // 1-indexed + header
        
        if len(record) < 4 {
            result.Errors = append(result.Errors, fmt.Sprintf("Line %d: insufficient columns", lineNum))
            continue
        }
        
        warehouseCode := strings.TrimSpace(record[^0])
        bookISBN := strings.TrimSpace(record[^1])
        quantityStr := strings.TrimSpace(record[^2])
        reason := strings.TrimSpace(record[^3])
        
        // Parse quantity
        quantity, err := strconv.Atoi(quantityStr)
        if err != nil {
            result.Errors = append(result.Errors, fmt.Sprintf("Line %d: invalid quantity", lineNum))
            continue
        }
        
        // Lookup warehouse ID
        warehouseID, err := s.warehouseRepo.FindIDByCode(ctx, warehouseCode)
        if err != nil {
            result.Errors = append(result.Errors, fmt.Sprintf("Line %d: warehouse not found (%s)", lineNum, warehouseCode))
            continue
        }
        
        // Lookup book ID
        bookID, err := s.bookRepo.FindIDByISBN(ctx, bookISBN)
        if err != nil {
            result.Errors = append(result.Errors, fmt.Sprintf("Line %d: book not found (%s)", lineNum, bookISBN))
            continue
        }
        
        // Adjust inventory
        err = s.AdjustInventory(ctx, warehouseID, bookID, InventoryAdjustment{
            AdjustmentType: "set",
            Quantity:       quantity,
            Reason:         reason,
        }, adjustedBy)
        
        if err != nil {
            result.Errors = append(result.Errors, fmt.Sprintf("Line %d: %s", lineNum, err.Error()))
            continue
        }
        
        result.SuccessCount++
    }
    
    log.Info("Bulk inventory update completed",
        "total", result.TotalRows,
        "success", result.SuccessCount,
        "errors", len(result.Errors),
    )
    
    return result, nil
}
```


#### 6.3 Low Stock Alerts API

`GET /v1/admin/inventory/low-stock?warehouse_id=xxx`

```go
func (s *InventoryService) GetLowStockItems(ctx context.Context, warehouseID *string) ([]LowStockItem, error) {
    query := `
        SELECT 
            w.name as warehouse_name,
            w.code as warehouse_code,
            b.title as book_title,
            b.isbn,
            wi.quantity,
            wi.reserved,
            (wi.quantity - wi.reserved) as available,
            wi.alert_threshold
        FROM warehouse_inventory wi
        JOIN warehouses w ON wi.warehouse_id = w.id
        JOIN books b ON wi.book_id = b.id
        WHERE wi.quantity <= wi.alert_threshold
    `
    
    args := []interface{}{}
    if warehouseID != nil {
        query += " AND wi.warehouse_id = $1"
        args = append(args, *warehouseID)
    }
    
    query += " ORDER BY wi.quantity ASC, b.title"
    
    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    items := []LowStockItem{}
    for rows.Next() {
        var item LowStockItem
        err := rows.Scan(
            &item.WarehouseName, &item.WarehouseCode,
            &item.BookTitle, &item.ISBN,
            &item.Quantity, &item.Reserved, &item.Available,
            &item.AlertThreshold,
        )
        if err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    
    return items, nil
}
```


#### 6.4 Inventory Movement History

`GET /v1/admin/inventory/movements/:book_id?warehouse_id=xxx&from=2025-10-01&to=2025-10-31`

```sql
CREATE TABLE inventory_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    book_id UUID NOT NULL REFERENCES books(id),
    old_quantity INT NOT NULL,
    new_quantity INT NOT NULL,
    difference INT NOT NULL, -- new - old
    reason TEXT NOT NULL,
    adjusted_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_adjustments_book ON inventory_adjustments(book_id, created_at DESC);
CREATE INDEX idx_adjustments_warehouse ON inventory_adjustments(warehouse_id, created_at DESC);
```

```go
func (s *InventoryService) GetMovementHistory(
    ctx context.Context,
    bookID string,
    filters MovementFilters,
) ([]InventoryMovement, error) {
    query := `
        SELECT 
            ia.id,
            w.name as warehouse_name,
            ia.old_quantity,
            ia.new_quantity,
            ia.difference,
            ia.reason,
            u.fullname as adjusted_by_name,
            ia.created_at
        FROM inventory_adjustments ia
        JOIN warehouses w ON ia.warehouse_id = w.id
        LEFT JOIN users u ON ia.adjusted_by = u.id
        WHERE ia.book_id = $1
    `
    
    args := []interface{}{bookID}
    argPos := 2
    
    if filters.WarehouseID != nil {
        query += fmt.Sprintf(" AND ia.warehouse_id = $%d", argPos)
        args = append(args, *filters.WarehouseID)
        argPos++
    }
    
    if filters.From != nil {
        query += fmt.Sprintf(" AND ia.created_at >= $%d", argPos)
        args = append(args, *filters.From)
        argPos++
    }
    
    if filters.To != nil {
        query += fmt.Sprintf(" AND ia.created_at <= $%d", argPos)
        args = append(args, *filters.To)
        argPos++
    }
    
    query += " ORDER BY ia.created_at DESC LIMIT 100"
    
    // Execute and return...
}
```


### Acceptance Criteria

- Admin adjust được inventory per warehouse[^1]
- Validation: cannot set quantity < reserved[^1]
- Bulk CSV import hoạt động, báo lỗi từng dòng[^1]
- Low stock alerts list đúng items[^1]
- Movement history track đầy đủ thay đổi[^1]
- Audit trail với adjusted_by[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]
- P1-T029: RBAC middleware[^1]


### Effort

2 ngày[^1]

***

## 7. Low Stock Alert Job (P3-T007)

### Mô tả

Background job gửi email alert cho admin khi stock dưới threshold.[^1]

### Business Logic

- Chạy daily vào 8 AM[^1]
- Check tất cả warehouses[^1]
- Email admin list low stock items[^1]
- Group by warehouse[^1]


### Công việc cụ thể

#### 7.1 Asynq Job Definition

```go
const TypeLowStockAlert = "inventory:low_stock_alert"

type LowStockAlertPayload struct {
    // Empty - runs for all warehouses
}
```


#### 7.2 Job Handler

Tạo file `internal/domains/warehouse/jobs/low_stock_alert.go`:[^1]

```go
func (h *InventoryJobHandler) SendLowStockAlert(ctx context.Context, task *asynq.Task) error {
    log.Info("Running low stock alert job")
    
    // 1. Get all low stock items
    items, err := h.inventoryService.GetLowStockItems(ctx, nil) // All warehouses
    if err != nil {
        return fmt.Errorf("failed to get low stock items: %w", err)
    }
    
    if len(items) == 0 {
        log.Info("No low stock items found")
        return nil
    }
    
    log.Info("Found low stock items", "count", len(items))
    
    // 2. Group by warehouse
    byWarehouse := make(map[string][]LowStockItem)
    for _, item := range items {
        byWarehouse[item.WarehouseCode] = append(byWarehouse[item.WarehouseCode], item)
    }
    
    // 3. Generate email HTML
    emailHTML := h.generateLowStockEmailHTML(byWarehouse, items)
    
    // 4. Get admin emails
    adminEmails, err := h.userRepo.FindAdminEmails(ctx)
    if err != nil {
        return fmt.Errorf("failed to get admin emails: %w", err)
    }
    
    if len(adminEmails) == 0 {
        log.Warn("No admin emails found")
        return nil
    }
    
    // 5. Send email
    err = h.emailService.SendLowStockAlert(LowStockAlertEmail{
        To:      adminEmails,
        Items:   items,
        HTMLBody: emailHTML,
    })
    
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }
    
    log.Info("Low stock alert email sent", "recipients", len(adminEmails), "items_count", len(items))
    
    return nil
}

func (h *InventoryJobHandler) generateLowStockEmailHTML(
    byWarehouse map[string][]LowStockItem,
    allItems []LowStockItem,
) string {
    var html strings.Builder
    
    html.WriteString(`
        <html>
        <head><style>
            table { border-collapse: collapse; width: 100%; }
            th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
            th { background-color: #f2f2f2; }
            .warning { color: red; font-weight: bold; }
        </style></head>
        <body>
        <h2>🚨 Low Stock Alert</h2>
        <p>The following items are running low on stock:</p>
    `)
    
    for warehouseCode, items := range byWarehouse {
        html.WriteString(fmt.Sprintf("<h3>Warehouse: %s</h3>", warehouseCode))
        html.WriteString("<table>")
        html.WriteString("<tr><th>Book</th><th>ISBN</th><th>Quantity</th><th>Reserved</th><th>Available</th><th>Threshold</th></tr>")
        
        for _, item := range items {
            html.WriteString(fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td>%s</td>
                    <td class="warning">%d</td>
                    <td>%d</td>
                    <td>%d</td>
                    <td>%d</td>
                </tr>
            `, item.BookTitle, item.ISBN, item.Quantity, item.Reserved, item.Available, item.AlertThreshold))
        }
        
        html.WriteString("</table><br>")
    }
    
    html.WriteString(fmt.Sprintf(`
        <p><strong>Total: %d items need restocking</strong></p>
        <p><a href="%s/admin/inventory/low-stock">View in Admin Panel</a></p>
        </body>
        </html>
    `, len(allItems), os.Getenv("APP_URL")))
    
    return html.String()
}
```


#### 7.3 Schedule Job (Cron)

```go
// In cmd/worker/main.go or scheduler
func scheduleRecurringJobs(scheduler *asynq.Scheduler) {
    // Low stock alert - Daily at 8 AM
    _, err := scheduler.Register(
        "0 8 * * *", // Cron expression
        asynq.NewTask(jobs.TypeLowStockAlert, nil),
        asynq.Queue(queue.QueueDefault),
    )
    
    if err != nil {
        log.Fatal("Failed to schedule low stock alert", "error", err)
    }
    
    log.Info("Scheduled low stock alert job", "cron", "0 8 * * *")
}
```


#### 7.4 Email Template

Tạo file `templates/emails/low_stock_alert.html`:[^1]

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .container { max-width: 800px; margin: 0 auto; padding: 20px; }
        .header { background: #f44336; color: white; padding: 15px; text-align: center; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th { background: #f2f2f2; font-weight: bold; padding: 10px; text-align: left; }
        td { padding: 10px; border-bottom: 1px solid #ddd; }
        .warning { color: #f44336; font-weight: bold; }
        .button { background: #2196F3; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>🚨 Low Stock Alert</h2>
        </div>
        
        <p>Hello Admin,</p>
        <p>The following items are running low on stock and need restocking:</p>
        
        {{range .ByWarehouse}}
        <h3>Warehouse: {{.WarehouseCode}} - {{.WarehouseName}}</h3>
        <table>
            <tr>
                <th>Book Title</th>
                <th>ISBN</th>
                <th>Current Qty</th>
                <th>Reserved</th>
                <th>Available</th>
                <th>Threshold</th>
            </tr>
            {{range .Items}}
            <tr>
                <td>{{.BookTitle}}</td>
                <td>{{.ISBN}}</td>
                <td class="warning">{{.Quantity}}</td>
                <td>{{.Reserved}}</td>
                <td>{{.Available}}</td>
                <td>{{.AlertThreshold}}</td>
            </tr>
            {{end}}
        </table>
        {{end}}
        
        <p><strong>Total: {{.TotalItems}} items need restocking</strong></p>
        
        <p>
            <a href="{{.AdminPanelURL}}" class="button">View in Admin Panel</a>
        </p>
        
        <p>This is an automated alert. Please take action to restock these items.</p>
    </div>
</body>
</html>
```


### Acceptance Criteria

- Job chạy daily vào 8 AM[^1]
- Email gửi đến tất cả admins[^1]
- Email group items by warehouse[^1]
- HTML email đẹp và responsive[^1]
- Link trực tiếp vào admin panel[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]
- P2-T008: Asynq setup[^1]
- P1-T033: Email service[^1]


### Effort

1 ngày[^1]

***

## 8. Inventory Sync Job (P3-T008)

### Mô tả

Background job định kỳ sync và validate inventory data integrity.[^1]

### Job Functions

1. **Detect anomalies**: quantity < reserved[^1]
2. **Cleanup stale reservations**: reservations > 24h old[^1]
3. **Update aggregate views**: Refresh materialized views[^1]
4. **Report discrepancies**: Log và alert nếu có issues[^1]

### Công việc cụ thể

#### 8.1 Job Handler

```go
const TypeInventorySync = "inventory:sync"

func (h *InventoryJobHandler) SyncInventory(ctx context.Context, task *asynq.Task) error {
    log.Info("Running inventory sync job")
    
    result := &InventorySyncResult{
        CheckedItems:        0,
        FixedReservations:   0,
        AnomaliesFound:      []string{},
    }
    
    // 1. Check for constraint violations
    violations, err := h.checkConstraintViolations(ctx)
    if err != nil {
        return fmt.Errorf("failed to check violations: %w", err)
    }
    
    if len(violations) > 0 {
        log.Warn("Found constraint violations", "count", len(violations))
        result.AnomaliesFound = append(result.AnomaliesFound, violations...)
        
        // Alert admins
        h.alertAdmins(ctx, "Inventory Constraint Violations", violations)
    }
    
    // 2. Cleanup stale reservations
    fixed, err := h.cleanupStaleReservations(ctx)
    if err != nil {
        return fmt.Errorf("failed to cleanup reservations: %w", err)
    }
    result.FixedReservations = fixed
    
    // 3. Update aggregate data
    err = h.refreshAggregates(ctx)
    if err != nil {
        log.Error("Failed to refresh aggregates", "error", err)
        // Non-critical, continue
    }
    
    // 4. Count total items checked
    err = h.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM warehouse_inventory
    `).Scan(&result.CheckedItems)
    
    log.Info("Inventory sync completed",
        "checked_items", result.CheckedItems,
        "fixed_reservations", result.FixedReservations,
        "anomalies", len(result.AnomaliesFound),
    )
    
    return nil
}

func (h *InventoryJobHandler) checkConstraintViolations(ctx context.Context) ([]string, error) {
    query := `
        SELECT 
            w.code as warehouse_code,
            b.title as book_title,
            wi.quantity,
            wi.reserved
        FROM warehouse_inventory wi
        JOIN warehouses w ON wi.warehouse_id = w.id
        JOIN books b ON wi.book_id = b.id
        WHERE wi.quantity < wi.reserved
        OR wi.quantity < 0
        OR wi.reserved < 0
    `
    
    rows, err := h.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    violations := []string{}
    for rows.Next() {
        var warehouseCode, bookTitle string
        var quantity, reserved int
        
        rows.Scan(&warehouseCode, &bookTitle, &quantity, &reserved)
        
        msg := fmt.Sprintf("%s - %s: quantity=%d, reserved=%d",
            warehouseCode, bookTitle, quantity, reserved)
        violations = append(violations, msg)
    }
    
    return violations, nil
}

func (h *InventoryJobHandler) cleanupStaleReservations(ctx context.Context) (int, error) {
    // Find orders with reservations older than 24h and still "pending"
    query := `
        WITH stale_orders AS (
            SELECT o.id, o.warehouse_id
            FROM orders o
            WHERE o.status = 'pending'
            AND o.payment_status = 'pending'
            AND o.created_at < NOW() - INTERVAL '24 hours'
        )
        UPDATE warehouse_inventory wi
        SET reserved = GREATEST(reserved - oi.quantity, 0),
            updated_at = NOW()
        FROM order_items oi
        JOIN stale_orders so ON oi.order_id = so.id
        WHERE wi.warehouse_id = so.warehouse_id
        AND wi.book_id = oi.book_id
    `
    
    result, err := h.db.ExecContext(ctx, query)
    if err != nil {
        return 0, err
    }
    
    rowsAffected, _ := result.RowsAffected()
    
    if rowsAffected > 0 {
        log.Info("Cleaned up stale reservations", "count", rowsAffected)
        
        // Also update orders to cancelled
        _, err = h.db.ExecContext(ctx, `
            UPDATE orders
            SET status = 'cancelled',
                cancelled_reason = 'Auto-cancelled: Payment timeout (24h)',
                cancelled_at = NOW()
            WHERE status = 'pending'
            AND payment_status = 'pending'
            AND created_at < NOW() - INTERVAL '24 hours'
        `)
    }
    
    return int(rowsAffected), nil
}

func (h *InventoryJobHandler) refreshAggregates(ctx context.Context) error {
    // Refresh materialized views (if using)
    // Or update cached data
    
    // Update books_total_stock view data in cache
    query := `
        SELECT 
            book_id,
            SUM(quantity) as total_quantity,
            SUM(reserved) as total_reserved,
            SUM(quantity - reserved) as available
        FROM warehouse_inventory
        GROUP BY book_id
    `
    
    rows, err := h.db.QueryContext(ctx, query)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Cache in Redis for fast lookups
    for rows.Next() {
        var bookID string
        var total, reserved, available int
        
        rows.Scan(&bookID, &total, &reserved, &available)
        
        // Store in Redis
        cacheKey := fmt.Sprintf("book:stock:%s", bookID)
        h.redis.HSet(ctx, cacheKey, map[string]interface{}{
            "total":     total,
            "reserved":  reserved,
            "available": available,
        })
        h.redis.Expire(ctx, cacheKey, 1*time.Hour)
    }
    
    log.Info("Refreshed inventory aggregates in cache")
    
    return nil
}
```


#### 8.2 Schedule Job

```go
// Run every 4 hours
_, err := scheduler.Register(
    "0 */4 * * *", // Every 4 hours
    asynq.NewTask(jobs.TypeInventorySync, nil),
    asynq.Queue(queue.QueueDefault),
)
```


### Acceptance Criteria

- Job chạy mỗi 4 giờ[^1]
- Detect được constraint violations[^1]
- Cleanup stale reservations (> 24h)[^1]
- Alert admins nếu có anomalies[^1]
- Refresh cached aggregate data[^1]


### Dependencies

- P3-T001: Warehouses tables[^1]
- P2-T008: Asynq setup[^1]


### Effort

1 ngày[^1]

***

## SUMMARY

### Total Effort Sprint 15-16

| Task ID | Task | Effort (days) |
| :-- | :-- | :-- |
| P3-T001 | Warehouses + warehouse_inventory tables | 1 |
| P3-T002 | Migrate existing stock to warehouse model | 1 |
| P3-T003 | Warehouse selection algorithm (nearest) | 2 |
| P3-T004 | Update order flow với warehouse_id | 2 |
| P3-T005 | Admin: Warehouse CRUD | 2 |
| P3-T006 | Admin: Inventory management per warehouse | 2 |
| P3-T007 | Low stock alert job | 1 |
| P3-T008 | Inventory sync job | 1 |
| **TOTAL** |  | **12 days** |

**Sprint duration**: 2 tuần (10 ngày làm việc)[^1]
**Team size**: 2 backend developers (có thể song song hóa tasks)[^1]

### Parallelization Strategy

**Week 1** (5 ngày):

- **Dev 1**: P3-T001 → P3-T002 → P3-T003 (Database + Algorithm) (1+1+2 = 4 days) + Code review (1 day)[^1]
- **Dev 2**: P3-T005 (Admin Warehouse CRUD) (2 days) + P3-T006 (Admin Inventory Management) (2 days) + Code review (1 day)[^1]

**Week 2** (5 ngày):

- **Dev 1**: P3-T004 (Update order flow) (2 days) + P3-T008 (Inventory sync job) (1 day) + Integration testing (2 days)[^1]
- **Dev 2**: P3-T007 (Low stock alert job) (1 day) + Integration testing + Bug fixes (4 days)[^1]


### Deliverables Checklist Sprint 15-16

- ✅ **Multi-warehouse system** fully functional[^1]
- ✅ **Inventory tracking** per warehouse với reservation logic[^1]
- ✅ **Warehouse selection algorithm** based on distance[^1]
- ✅ **Order flow integration** với automatic warehouse assignment[^1]
- ✅ **Admin inventory management** với adjustment logs[^1]
- ✅ **Bulk CSV import** for inventory updates[^1]
- ✅ **Low stock alerts** automated với email notifications[^1]
- ✅ **Inventory sync job** maintaining data integrity[^1]


### Key Technical Achievements

**Scalability**:

- Multi-warehouse architecture ready for expansion[^1]
- Distance-based warehouse selection optimize shipping costs[^1]
- Partition-ready for high-volume inventory transactions[^1]

**Data Integrity**:

- Pessimistic locking prevents overselling[^1]
- Constraints ensure quantity >= reserved[^1]
- Audit trail tracks all inventory changes[^1]
- Sync job detects and fixes anomalies[^1]

**Business Value**:

- Reduced shipping costs (nearest warehouse)[^1]
- Real-time inventory visibility[^1]
- Proactive low stock alerts[^1]
- Comprehensive audit trail for compliance[^1]


### Database Changes Summary

- ✅ `warehouses` table với geolocation[^1]
- ✅ `warehouse_inventory` table với composite PK[^1]
- ✅ `inventory_adjustments` audit log table[^1]
- ✅ Views: `books_total_stock`, `warehouse_stock_summary`, `low_stock_items`[^1]
- ✅ Migration script từ old inventory model[^1]


### Next Steps (Sprint 17-18)

Phase 3 tiếp theo sẽ focus vào **eBook Management** với file storage, download links, và DRM watermarking.[^1]

<div align="center">⁂</div>

[^1]: USER-REQUIREMENTS-DOCUMENT-URD-PHIEN-BAN-HOA.docx

