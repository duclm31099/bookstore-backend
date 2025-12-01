-- ================================================
-- Rollback Migration: Drop Promotion Removal Logs
-- ================================================

-- Drop indexes first (good practice, though CASCADE would handle it)
DROP INDEX IF EXISTS idx_promotion_removal_logs_user;
DROP INDEX IF EXISTS idx_promotion_removal_logs_notified;
DROP INDEX IF EXISTS idx_promotion_removal_logs_cart;
DROP INDEX IF EXISTS idx_promotion_removal_logs_removed_at;

-- Drop table
DROP TABLE IF EXISTS promotion_removal_logs;
