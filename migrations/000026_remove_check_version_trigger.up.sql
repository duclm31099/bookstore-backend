DROP TRIGGER IF EXISTS trg_warehouse_inventory_version ON warehouse_inventory;
DROP FUNCTION IF EXISTS update_warehouse_inventory_version();

CREATE OR REPLACE FUNCTION update_warehouse_inventory_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.version = OLD.version + 1;
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_warehouse_inventory_version
    BEFORE UPDATE ON warehouse_inventory
    FOR EACH ROW
    EXECUTE FUNCTION update_warehouse_inventory_version();
