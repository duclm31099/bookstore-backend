-- Drop functions
DROP FUNCTION IF EXISTS reserve_inventory(UUID, INT, TEXT);
DROP FUNCTION IF EXISTS get_total_available_stock(UUID);
DROP FUNCTION IF EXISTS log_inventory_movement();

-- Drop tables (cascade will drop triggers)
DROP TABLE IF EXISTS inventory_movements;
DROP TABLE IF EXISTS inventories;
