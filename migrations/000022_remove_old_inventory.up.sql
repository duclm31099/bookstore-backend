-- Migration: 20251108_003_drop_old_inventory_schema
-- Description: Drop old inventories + inventory_movements schema (no data migration needed)
-- Author: System
-- Date: 2025-11-08
-- Reason: Clean up old schema - new warehouse_inventory schema is primary

-- =====================================================
-- DROP OLD TRIGGERS
-- =====================================================

DROP TRIGGER IF EXISTS trigger_log_inventory_changes ON inventories CASCADE;
DROP TRIGGER IF EXISTS update_inventories_updated_at ON inventories CASCADE;

-- =====================================================
-- DROP OLD FUNCTIONS
-- =====================================================

DROP FUNCTION IF EXISTS log_inventory_movement() CASCADE;
DROP FUNCTION IF EXISTS get_total_available_stock(UUID) CASCADE;
DROP FUNCTION IF EXISTS reserve_inventory(UUID, INT, TEXT) CASCADE;

-- =====================================================
-- DROP OLD TABLES (CASCADE drops indexes automatically)
-- =====================================================

-- Drop inventory_movements first (has FK to inventories)
DROP TABLE IF EXISTS inventory_movements CASCADE;

-- Drop inventories table
DROP TABLE IF EXISTS inventories CASCADE;

-- =====================================================
-- VERIFICATION
-- =====================================================

-- Verify tables are gone
DO $$
BEGIN
    IF EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_name IN ('inventories', 'inventory_movements')
    ) THEN
        RAISE EXCEPTION 'Old inventory tables still exist after DROP';
    END IF;
    
    -- Verify new tables exist
    IF NOT EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_name IN ('warehouses', 'warehouse_inventory', 'inventory_audit_log')
    ) THEN
        RAISE EXCEPTION 'New warehouse tables do not exist - verify migration order';
    END IF;
    
    RAISE NOTICE 'Migration successful: old inventory schema dropped, new warehouse schema active';
END $$;
