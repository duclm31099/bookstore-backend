-- Migration Version: 20251108_001_create_warehouses_and_inventory.up.sql
-- Description: Create warehouses and warehouse_inventory tables with version control and auditing
-- Author: System Generated
-- Date: 2025-11-08

-- =====================================================
-- TABLE: warehouses
-- =====================================================
CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT UNIQUE NOT NULL,
    address TEXT NOT NULL,
    province TEXT NOT NULL,
    latitude DECIMAL(9,6),
    longitude DECIMAL(9,6),
    is_active BOOLEAN DEFAULT true,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_warehouses_active ON warehouses(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_warehouses_code ON warehouses(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_warehouses_location ON warehouses(latitude, longitude) WHERE is_active = true AND deleted_at IS NULL;
CREATE INDEX idx_warehouses_province ON warehouses(province) WHERE is_active = true;

-- =====================================================
-- TABLE: warehouse_inventory
-- =====================================================
CREATE TABLE warehouse_inventory (
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE RESTRICT,
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved INT NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    alert_threshold INT DEFAULT 10,
    version INT NOT NULL DEFAULT 1,
    last_restocked_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),
    PRIMARY KEY (warehouse_id, book_id),
    CONSTRAINT available_stock CHECK (quantity >= reserved)
);

CREATE INDEX idx_inventory_book ON warehouse_inventory(book_id) WHERE quantity > 0;
CREATE INDEX idx_inventory_warehouse ON warehouse_inventory(warehouse_id);
CREATE INDEX idx_inventory_low_stock ON warehouse_inventory(warehouse_id, book_id) 
    WHERE quantity <= alert_threshold AND quantity > 0;
CREATE INDEX idx_inventory_available ON warehouse_inventory(book_id, warehouse_id) 
    WHERE (quantity - reserved) > 0;
CREATE INDEX idx_inventory_updated_at ON warehouse_inventory(updated_at DESC);

-- =====================================================
-- TABLE: inventory_audit_log (FR-INV-005)
-- =====================================================
CREATE TABLE inventory_audit_log (
    id UUID DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    book_id UUID NOT NULL REFERENCES books(id),
    action TEXT NOT NULL CHECK (action IN ('RESTOCK', 'RESERVE', 'RELEASE', 'ADJUSTMENT', 'SALE')),
    old_quantity INT NOT NULL,
    new_quantity INT NOT NULL,
    old_reserved INT NOT NULL,
    new_reserved INT NOT NULL,
    quantity_change INT NOT NULL,
    reason TEXT,
    changed_by UUID REFERENCES users(id),
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE inventory_audit_log_2025_q4 PARTITION OF inventory_audit_log
    FOR VALUES FROM ('2025-10-01') TO ('2026-01-01');
CREATE TABLE inventory_audit_log_2026_q1 PARTITION OF inventory_audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');
CREATE TABLE inventory_audit_log_2026_q2 PARTITION OF inventory_audit_log
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');

CREATE INDEX idx_audit_log_warehouse_book ON inventory_audit_log(warehouse_id, book_id, created_at DESC);
CREATE INDEX idx_audit_log_changed_by ON inventory_audit_log(changed_by, created_at DESC);
CREATE INDEX idx_audit_log_action ON inventory_audit_log(action, created_at DESC);
CREATE INDEX idx_audit_log_id ON inventory_audit_log(id);

-- =====================================================
-- TRIGGER: Update version and timestamp
-- =====================================================
CREATE OR REPLACE FUNCTION update_warehouse_inventory_version()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.version IS NOT NULL AND NEW.version <> OLD.version THEN
        RAISE EXCEPTION 'Concurrent modification detected. Please retry the operation.'
            USING ERRCODE = '40001';
    END IF;
    
    NEW.version = OLD.version + 1;
    NEW.updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_warehouse_inventory_version
    BEFORE UPDATE ON warehouse_inventory
    FOR EACH ROW
    EXECUTE FUNCTION update_warehouse_inventory_version();

-- =====================================================
-- TRIGGER: Auto-create audit log (FR-INV-005)
-- =====================================================
CREATE OR REPLACE FUNCTION log_inventory_change()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE') THEN
        IF OLD.quantity <> NEW.quantity OR OLD.reserved <> NEW.reserved THEN
            INSERT INTO inventory_audit_log (
                warehouse_id,
                book_id,
                action,
                old_quantity,
                new_quantity,
                old_reserved,
                new_reserved,
                quantity_change,
                changed_by,
                created_at
            ) VALUES (
                NEW.warehouse_id,
                NEW.book_id,
                CASE
                    WHEN NEW.quantity > OLD.quantity THEN 'RESTOCK'
                    WHEN NEW.reserved > OLD.reserved THEN 'RESERVE'
                    WHEN NEW.reserved < OLD.reserved THEN 'RELEASE'
                    ELSE 'ADJUSTMENT'
                END,
                OLD.quantity,
                NEW.quantity,
                OLD.reserved,
                NEW.reserved,
                NEW.quantity - OLD.quantity,
                NEW.updated_by,
                NOW()
            );
        END IF;
    ELSIF (TG_OP = 'INSERT') THEN
        INSERT INTO inventory_audit_log (
            warehouse_id,
            book_id,
            action,
            old_quantity,
            new_quantity,
            old_reserved,
            new_reserved,
            quantity_change,
            changed_by,
            created_at
        ) VALUES (
            NEW.warehouse_id,
            NEW.book_id,
            'RESTOCK',
            0,
            NEW.quantity,
            0,
            NEW.reserved,
            NEW.quantity,
            NEW.updated_by,
            NOW()
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_inventory_audit_log
    AFTER INSERT OR UPDATE ON warehouse_inventory
    FOR EACH ROW
    EXECUTE FUNCTION log_inventory_change();

-- =====================================================
-- TABLE: low_stock_alerts (FR-INV-004)
-- =====================================================
CREATE TABLE low_stock_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    book_id UUID NOT NULL REFERENCES books(id),
    current_quantity INT NOT NULL,
    alert_threshold INT NOT NULL,
    is_resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_low_stock_alerts_active ON low_stock_alerts(warehouse_id, book_id) 
    WHERE is_resolved = false;

CREATE INDEX idx_low_stock_alerts_unresolved ON low_stock_alerts(is_resolved, created_at DESC) 
    WHERE is_resolved = false;

CREATE OR REPLACE FUNCTION check_low_stock()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.quantity < NEW.alert_threshold AND 
       (TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.quantity >= OLD.alert_threshold)) THEN
        INSERT INTO low_stock_alerts (
            warehouse_id,
            book_id,
            current_quantity,
            alert_threshold
        ) VALUES (
            NEW.warehouse_id,
            NEW.book_id,
            NEW.quantity,
            NEW.alert_threshold
        )
        ON CONFLICT (warehouse_id, book_id) 
        WHERE is_resolved = false
        DO UPDATE SET 
            current_quantity = EXCLUDED.current_quantity,
            created_at = NOW();
    END IF;
    
    IF TG_OP = 'UPDATE' AND NEW.quantity >= NEW.alert_threshold AND OLD.quantity < OLD.alert_threshold THEN
        UPDATE low_stock_alerts
        SET is_resolved = true,
            resolved_at = NOW()
        WHERE warehouse_id = NEW.warehouse_id
          AND book_id = NEW.book_id
          AND is_resolved = false;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_low_stock_alert
    AFTER INSERT OR UPDATE ON warehouse_inventory
    FOR EACH ROW
    EXECUTE FUNCTION check_low_stock();

-- =====================================================
-- VIEW: Total stock across all warehouses
-- =====================================================
CREATE OR REPLACE VIEW books_total_stock AS
SELECT 
    book_id,
    SUM(quantity) as total_quantity,
    SUM(reserved) as total_reserved,
    SUM(quantity - reserved) as available,
    COUNT(DISTINCT warehouse_id) as warehouse_count,
    ARRAY_AGG(
        DISTINCT warehouse_id 
        ORDER BY warehouse_id
    ) FILTER (WHERE quantity > reserved) as warehouses_with_stock
FROM warehouse_inventory
GROUP BY book_id;

-- =====================================================
-- FUNCTION: Reserve stock with pessimistic lock (FR-INV-003)
-- =====================================================
CREATE OR REPLACE FUNCTION reserve_stock(
    p_warehouse_id UUID,
    p_book_id UUID,
    p_quantity INT,
    p_user_id UUID DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_available INT;
    v_current_version INT;
BEGIN
    SELECT quantity - reserved, version
    INTO v_available, v_current_version
    FROM warehouse_inventory
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
    FOR UPDATE NOWAIT;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Inventory record not found for warehouse % and book %', p_warehouse_id, p_book_id;
    END IF;
    
    IF v_available < p_quantity THEN
        RAISE EXCEPTION 'Insufficient stock. Available: %, Requested: %', v_available, p_quantity
            USING ERRCODE = 'BIZ01';
    END IF;
    
    UPDATE warehouse_inventory
    SET reserved = reserved + p_quantity,
        updated_by = p_user_id,
        version = v_current_version + 1,
        updated_at = NOW()
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
      AND version = v_current_version;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Concurrent modification detected during reservation';
    END IF;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- FUNCTION: Release reserved stock
-- =====================================================
CREATE OR REPLACE FUNCTION release_stock(
    p_warehouse_id UUID,
    p_book_id UUID,
    p_quantity INT,
    p_user_id UUID DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_version INT;
BEGIN
    SELECT version
    INTO v_current_version
    FROM warehouse_inventory
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
    FOR UPDATE NOWAIT;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Inventory record not found';
    END IF;
    
    UPDATE warehouse_inventory
    SET reserved = GREATEST(reserved - p_quantity, 0),
        updated_by = p_user_id,
        version = v_current_version + 1,
        updated_at = NOW()
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
      AND version = v_current_version;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- FUNCTION: Complete sale (reduce quantity and reserved)
-- =====================================================
CREATE OR REPLACE FUNCTION complete_sale(
    p_warehouse_id UUID,
    p_book_id UUID,
    p_quantity INT,
    p_user_id UUID DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_version INT;
BEGIN
    SELECT version
    INTO v_current_version
    FROM warehouse_inventory
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
    FOR UPDATE NOWAIT;
    
    UPDATE warehouse_inventory
    SET quantity = quantity - p_quantity,
        reserved = GREATEST(reserved - p_quantity, 0),
        updated_by = p_user_id,
        version = v_current_version + 1,
        updated_at = NOW()
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
      AND version = v_current_version
      AND quantity >= p_quantity;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Insufficient quantity for sale';
    END IF;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- FUNCTION: Find nearest warehouse with stock (FR-INV-002)
-- =====================================================
CREATE OR REPLACE FUNCTION find_nearest_warehouse(
    p_book_id UUID,
    p_latitude DECIMAL(9,6),
    p_longitude DECIMAL(9,6),
    p_required_quantity INT DEFAULT 1
)
RETURNS TABLE (
    warehouse_id UUID,
    warehouse_name TEXT,
    available_quantity INT,
    distance_km DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        w.id,
        w.name,
        (wi.quantity - wi.reserved)::INT,
        (6371 * acos(
            cos(radians(p_latitude)) * 
            cos(radians(w.latitude)) * 
            cos(radians(w.longitude) - radians(p_longitude)) + 
            sin(radians(p_latitude)) * 
            sin(radians(w.latitude))
        ))::DECIMAL as distance_km
    FROM warehouses w
    INNER JOIN warehouse_inventory wi 
        ON w.id = wi.warehouse_id
    WHERE w.is_active = true
      AND w.deleted_at IS NULL
      AND w.latitude IS NOT NULL
      AND w.longitude IS NOT NULL
      AND wi.book_id = p_book_id
      AND (wi.quantity - wi.reserved) >= p_required_quantity
    ORDER BY distance_km ASC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS
-- =====================================================
COMMENT ON TABLE warehouses IS 'Stores warehouse locations and information';
COMMENT ON TABLE warehouse_inventory IS 'Tracks inventory levels per warehouse with version control to prevent race conditions';
COMMENT ON TABLE inventory_audit_log IS 'Audit trail for all inventory changes (partitioned by created_at for performance)';
COMMENT ON TABLE low_stock_alerts IS 'Alerts for low stock items per warehouse';

COMMENT ON COLUMN warehouse_inventory.version IS 'Optimistic locking version to prevent concurrent update conflicts';
COMMENT ON COLUMN warehouse_inventory.reserved IS 'Quantity reserved for pending orders (checkout timeout 15m)';
COMMENT ON COLUMN warehouse_inventory.alert_threshold IS 'Triggers email alert to admin when stock falls below this value';
COMMENT ON COLUMN inventory_audit_log.created_at IS 'Partition key - included in PRIMARY KEY for partitioned table';

-- =====================================================
-- SEED DATA (Optional - for testing)
-- =====================================================
INSERT INTO warehouses (id, name, code, address, province, latitude, longitude) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'Kho Hà Nội', 'HN-01', '123 Đường Láng, Đống Đa', 'Hà Nội', 21.028511, 105.804817),
    ('550e8400-e29b-41d4-a716-446655440002', 'Kho TP.HCM', 'HCM-01', '456 Nguyễn Trãi, Quận 1', 'TP. Hồ Chí Minh', 10.762622, 106.660172),
    ('550e8400-e29b-41d4-a716-446655440003', 'Kho Đà Nẵng', 'DN-01', '789 Hải Phòng, Sơn Trà', 'Đà Nẵng', 16.047079, 108.206230);
