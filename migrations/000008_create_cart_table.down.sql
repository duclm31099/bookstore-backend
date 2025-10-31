DROP TRIGGER IF EXISTS trigger_update_cart_totals_delete ON cart_items;
DROP TRIGGER IF EXISTS trigger_update_cart_totals_update ON cart_items;
DROP TRIGGER IF EXISTS trigger_update_cart_totals_insert ON cart_items;
DROP FUNCTION IF EXISTS update_cart_totals() CASCADE;
DROP TRIGGER IF EXISTS update_cart_items_updated_at ON cart_items;
DROP TRIGGER IF EXISTS update_carts_updated_at ON carts;
DROP TABLE IF EXISTS cart_items CASCADE;
DROP TABLE IF EXISTS carts CASCADE;
