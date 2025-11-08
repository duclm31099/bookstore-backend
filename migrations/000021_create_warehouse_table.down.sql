-- Migration Version: 20251108_001_create_warehouses_and_inventory.down.sql

DROP FUNCTION IF EXISTS find_nearest_warehouse(UUID, DECIMAL, DECIMAL, INT);
DROP FUNCTION IF EXISTS complete_sale(UUID, UUID, INT, UUID);
DROP FUNCTION IF EXISTS release_stock(UUID, UUID, INT, UUID);
DROP FUNCTION IF EXISTS reserve_stock(UUID, UUID, INT, UUID);
DROP FUNCTION IF EXISTS check_low_stock();
DROP FUNCTION IF EXISTS log_inventory_change();
DROP FUNCTION IF EXISTS update_warehouse_inventory_version();

DROP VIEW IF EXISTS books_total_stock;

DROP TABLE IF EXISTS low_stock_alerts;
DROP TABLE IF EXISTS inventory_audit_log_2026_q2;
DROP TABLE IF EXISTS inventory_audit_log_2026_q1;
DROP TABLE IF EXISTS inventory_audit_log_2025_q4;
DROP TABLE IF EXISTS inventory_audit_log;
DROP TABLE IF EXISTS warehouse_inventory;
DROP TABLE IF EXISTS warehouses;
