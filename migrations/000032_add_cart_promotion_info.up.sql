-- Migration: Add promotion support to carts table
-- Version: 20251117_add_promotion_to_carts
-- Environment: DEV (single file, safe for re-run)

-- ================================================
-- STEP 1: Add columns with safe defaults
-- ================================================
ALTER TABLE carts
ADD COLUMN IF NOT EXISTS promo_code TEXT DEFAULT NULL,
ADD COLUMN IF NOT EXISTS discount NUMERIC(12,2) DEFAULT 0 NOT NULL,
ADD COLUMN IF NOT EXISTS total NUMERIC(12,2) DEFAULT 0 NOT NULL,
ADD COLUMN IF NOT EXISTS version INT DEFAULT 1 NOT NULL,
ADD COLUMN IF NOT EXISTS promo_metadata JSONB;

-- ================================================
-- STEP 2: Backfill existing data
-- ================================================
-- Set total = subtotal for existing carts (no discount applied yet)
UPDATE carts SET total = subtotal WHERE total = 0;

-- ================================================
-- STEP 3: Create indexes (drop if exists first)
-- ================================================
DROP INDEX IF EXISTS idx_carts_promo_code;
CREATE INDEX idx_carts_promo_code ON carts(promo_code) WHERE promo_code IS NOT NULL;

DROP INDEX IF EXISTS idx_carts_version;
CREATE INDEX idx_carts_version ON carts(version);

-- ================================================
-- STEP 4: Create trigger function
-- ================================================
-- Drop existing trigger and function first (safe for re-run)
DROP TRIGGER IF EXISTS trigger_update_cart_total ON carts;
DROP FUNCTION IF EXISTS update_cart_total();

-- Create function to auto-calculate total
CREATE OR REPLACE FUNCTION update_cart_total()
RETURNS TRIGGER AS $$
BEGIN
    NEW.total = NEW.subtotal - COALESCE(NEW.discount, 0);
    
    -- Ensure total never goes negative
    IF NEW.total < 0 THEN
        NEW.total = 0;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
CREATE TRIGGER trigger_update_cart_total
    BEFORE INSERT OR UPDATE OF subtotal, discount ON carts
    FOR EACH ROW
    EXECUTE FUNCTION update_cart_total();

-- ================================================
-- STEP 5: Add comments (documentation)
-- ================================================
COMMENT ON COLUMN carts.promo_code IS 'Applied promotion code';
COMMENT ON COLUMN carts.discount IS 'Discount amount from promotion (VND)';
COMMENT ON COLUMN carts.total IS 'Final total = subtotal - discount (auto-calculated by trigger)';
COMMENT ON COLUMN carts.version IS 'Version number for optimistic locking (prevent concurrent updates)';
COMMENT ON COLUMN carts.promo_metadata IS 'Full promotion details as JSON (type, value, applied_at, etc)';

-- ================================================
-- VERIFICATION (optional, for dev)
-- ================================================
-- Check if columns exist
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'carts' AND column_name = 'promo_code'
    ) THEN
        RAISE NOTICE 'Migration successful: promo_code column exists';
    ELSE
        RAISE EXCEPTION 'Migration failed: promo_code column not found';
    END IF;
END $$;
