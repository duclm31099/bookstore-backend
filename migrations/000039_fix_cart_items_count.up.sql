-- ================================================
-- FIX: Cart items_count calculation
-- ================================================
-- Problem: items_count was using SUM(quantity) instead of COUNT(*)
-- This caused items_count to show total quantity instead of number of distinct items

CREATE OR REPLACE FUNCTION update_cart_totals()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE carts
    SET 
        items_count = (
            SELECT COALESCE(COUNT(*), 0)  -- ✅ FIX: Count number of items, not sum of quantities
            FROM cart_items
            WHERE cart_id = COALESCE(NEW.cart_id, OLD.cart_id)
        ),
        subtotal = (
            SELECT COALESCE(SUM(quantity * price), 0)
            FROM cart_items
            WHERE cart_id = COALESCE(NEW.cart_id, OLD.cart_id)
        ),
        updated_at = NOW()
    WHERE id = COALESCE(NEW.cart_id, OLD.cart_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Update existing carts to fix incorrect items_count
-- ================================================
UPDATE carts
SET items_count = (
    SELECT COALESCE(COUNT(*), 0)
    FROM cart_items
    WHERE cart_items.cart_id = carts.id
);
