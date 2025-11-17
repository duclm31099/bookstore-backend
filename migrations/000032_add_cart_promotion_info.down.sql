-- Version: 20251117_add_promotion_to_carts.down.sql

-- Remove trigger
DROP TRIGGER IF EXISTS trigger_update_cart_total ON carts;

-- Remove function
DROP FUNCTION IF EXISTS update_cart_total();

-- Remove indexes
DROP INDEX IF EXISTS idx_carts_promo_code;
DROP INDEX IF EXISTS idx_carts_version;

-- Remove columns (⚠️ Data loss!)
ALTER TABLE carts
DROP COLUMN IF EXISTS promo_code,
DROP COLUMN IF EXISTS discount,
DROP COLUMN IF EXISTS total,
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS promo_metadata;
