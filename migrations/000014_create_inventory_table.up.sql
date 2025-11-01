-- ================================================
-- INVENTORIES TABLE
-- Track stock levels per warehouse location
-- ================================================
CREATE TABLE IF NOT EXISTS inventories (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    
    -- Location (Vietnam warehouses)
    warehouse_location TEXT NOT NULL CHECK (warehouse_location IN ('HN', 'HCM', 'DN', 'CT')),
    
    -- Stock levels
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    available_quantity INT GENERATED ALWAYS AS (quantity - reserved_quantity) STORED,
    
    -- Alerts
    low_stock_threshold INT DEFAULT 10 CHECK (low_stock_threshold >= 0),
    is_low_stock BOOLEAN GENERATED ALWAYS AS (quantity - reserved_quantity <= low_stock_threshold) STORED,
    
    -- Timestamps
    last_restock_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- One inventory record per book per location
    UNIQUE(book_id, warehouse_location),
    
    -- Reserved cannot exceed total
    CHECK (reserved_quantity <= quantity)
);

-- ================================================
-- INVENTORY MOVEMENTS TABLE
-- Audit trail for all stock changes
-- ================================================
CREATE TABLE IF NOT EXISTS inventory_movements (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    inventory_id UUID NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
    
    -- Movement details
    movement_type TEXT NOT NULL CHECK (
        movement_type IN ('inbound', 'outbound', 'adjustment', 'return', 'reserve', 'release')
    ),
    quantity INT NOT NULL, -- Can be negative for outbound
    
    -- Before/After snapshot
    quantity_before INT NOT NULL,
    quantity_after INT NOT NULL,
    
    -- Reference to related entity
    reference_type TEXT CHECK (reference_type IN ('order', 'purchase', 'manual', 'return')),
    reference_id UUID, -- order_id, purchase_id, etc.
    
    -- Additional info
    notes TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL, -- Warehouse staff
    
    -- Timestamp
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================
-- INDEXES
-- ================================================
CREATE INDEX idx_inventories_book ON inventories(book_id);
CREATE INDEX idx_inventories_location ON inventories(warehouse_location);
CREATE INDEX idx_inventories_low_stock ON inventories(warehouse_location) WHERE is_low_stock = true;

CREATE INDEX idx_inventory_movements_inventory ON inventory_movements(inventory_id, created_at DESC);
CREATE INDEX idx_inventory_movements_reference ON inventory_movements(reference_type, reference_id);
CREATE INDEX idx_inventory_movements_created_by ON inventory_movements(created_by);

-- ================================================
-- TRIGGERS
-- ================================================

-- Auto-update updated_at on inventory changes
CREATE TRIGGER update_inventories_updated_at
BEFORE UPDATE ON inventories
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Auto-log all inventory changes
CREATE OR REPLACE FUNCTION log_inventory_movement()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if quantity changes
    IF (TG_OP = 'UPDATE' AND OLD.quantity != NEW.quantity) THEN
        INSERT INTO inventory_movements (
            inventory_id,
            movement_type,
            quantity,
            quantity_before,
            quantity_after,
            notes
        ) VALUES (
            NEW.id,
            'adjustment', -- Default type, will be overridden by application
            NEW.quantity - OLD.quantity,
            OLD.quantity,
            NEW.quantity,
            'Auto-tracked change'
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_inventory_changes
AFTER UPDATE OF quantity ON inventories
FOR EACH ROW
EXECUTE FUNCTION log_inventory_movement();

-- ================================================
-- HELPER FUNCTION: Get total stock across all warehouses
-- ================================================
CREATE OR REPLACE FUNCTION get_total_available_stock(p_book_id UUID)
RETURNS INT AS $$
BEGIN
    RETURN (
        SELECT COALESCE(SUM(available_quantity), 0)
        FROM inventories
        WHERE book_id = p_book_id
    );
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- HELPER FUNCTION: Reserve stock for order
-- ================================================
CREATE OR REPLACE FUNCTION reserve_inventory(
    p_book_id UUID,
    p_quantity INT,
    p_warehouse_location TEXT DEFAULT 'HN'
)
RETURNS BOOLEAN AS $$
DECLARE
    v_available INT;
BEGIN
    -- Check available stock
    SELECT available_quantity INTO v_available
    FROM inventories
    WHERE book_id = p_book_id 
      AND warehouse_location = p_warehouse_location
    FOR UPDATE; -- Lock row
    
    IF v_available IS NULL OR v_available < p_quantity THEN
        RETURN FALSE;
    END IF;
    
    -- Reserve stock
    UPDATE inventories
    SET reserved_quantity = reserved_quantity + p_quantity
    WHERE book_id = p_book_id 
      AND warehouse_location = p_warehouse_location;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE inventories IS 'Stock levels per book per warehouse';
COMMENT ON COLUMN inventories.available_quantity IS 'Auto-calculated: quantity - reserved_quantity';
COMMENT ON COLUMN inventories.is_low_stock IS 'Auto-calculated: available <= threshold';

COMMENT ON TABLE inventory_movements IS 'Audit trail for all inventory changes';
COMMENT ON COLUMN inventory_movements.movement_type IS 'inbound=received, outbound=shipped, reserve=order pending';
