-- ================================================
-- Migration Rollback: Remove Version Control
-- Purpose: Undo version column additions
-- Description: Remove version columns and indexes
-- ================================================

-- Drop indexes first (indexes are linked to columns)
DROP INDEX IF EXISTS idx_inventories_version CASCADE;
DROP INDEX IF EXISTS idx_orders_version CASCADE;
DROP INDEX IF EXISTS idx_carts_version CASCADE;
DROP INDEX IF EXISTS idx_cart_items_version CASCADE;
DROP INDEX IF EXISTS idx_books_version CASCADE;
DROP INDEX IF EXISTS idx_promotions_version CASCADE;
DROP INDEX IF EXISTS idx_promotion_usage_version CASCADE;

-- Drop columns
ALTER TABLE inventories DROP COLUMN IF EXISTS version CASCADE;
ALTER TABLE orders DROP COLUMN IF EXISTS version CASCADE;
ALTER TABLE carts DROP COLUMN IF EXISTS version CASCADE;
ALTER TABLE cart_items DROP COLUMN IF EXISTS version CASCADE;
ALTER TABLE books DROP COLUMN IF EXISTS version CASCADE;
ALTER TABLE promotions DROP COLUMN IF EXISTS version CASCADE;
ALTER TABLE promotion_usage DROP COLUMN IF EXISTS version CASCADE;

-- Verification
SELECT 
    table_name,
    column_name
FROM information_schema.columns 
WHERE table_name IN (
    'inventories', 'orders', 'carts', 'cart_items', 
    'books', 'promotions', 'promotion_usage'
)
AND column_name = 'version';
-- Should return: (empty result - all removed)
