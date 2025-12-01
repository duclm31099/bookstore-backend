-- ================================================
-- Migration: Create Promotion Removal Logs Table
-- Purpose: Audit trail for automatic promotion removals
-- Version: 000033
-- ================================================

-- WHY THIS TABLE?
-- 1. Audit Trail: Track all automatic promotion removals for compliance
-- 2. Debugging: Help troubleshoot why promotions were removed
-- 3. Future Notifications: Prepare infrastructure for notifying users
-- 4. Analytics: Understand promotion expiry patterns

CREATE TABLE IF NOT EXISTS promotion_removal_logs (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Foreign keys with CASCADE delete
    -- WHY CASCADE? If cart/user is deleted, we don't need the log anymore
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Promotion details at time of removal
    -- WHY STORE CODE? Promotion might be deleted from promotions table later
    promo_code TEXT NOT NULL,
    discount_amount NUMERIC(12,2) NOT NULL,
    
    -- Removal reason for debugging and user communication
    -- Possible values: 'expired', 'disabled', 'max_uses_reached'
    removal_reason TEXT NOT NULL,
    
    -- Full promotion snapshot as JSON
    -- WHY JSONB? Flexible storage for all promotion details without schema changes
    -- Useful for: showing users what they lost, analytics, debugging
    promo_metadata JSONB,
    
    -- Timestamp of removal
    removed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Flag for future notification system
    -- WHY BOOLEAN? Simple flag to track if user has been notified
    -- Future job can query WHERE notified = FALSE to send notifications
    notified BOOLEAN DEFAULT FALSE,
    
    -- Audit timestamp
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================
-- INDEXES FOR PERFORMANCE
-- ================================================

-- Index 1: User-centric queries
-- USE CASE: "Show me all promotions removed for this user"
-- WHY DESC? Most recent removals are queried more often
CREATE INDEX idx_promotion_removal_logs_user 
ON promotion_removal_logs(user_id, removed_at DESC);

-- Index 2: Notification system
-- USE CASE: "Find all unnotified removals to send notifications"
-- WHY PARTIAL INDEX? Only index rows where notified = FALSE (smaller index)
-- PERFORMANCE: Faster queries for notification job
CREATE INDEX idx_promotion_removal_logs_notified 
ON promotion_removal_logs(notified) 
WHERE notified = FALSE;

-- Index 3: Cart-centric queries
-- USE CASE: "Show removal history for this specific cart"
CREATE INDEX idx_promotion_removal_logs_cart 
ON promotion_removal_logs(cart_id);

-- Index 4: Analytics queries
-- USE CASE: "How many promotions were removed today/this week?"
CREATE INDEX idx_promotion_removal_logs_removed_at 
ON promotion_removal_logs(removed_at DESC);

-- ================================================
-- COMMENTS FOR DOCUMENTATION
-- ================================================

COMMENT ON TABLE promotion_removal_logs IS 
'Audit log for automatic promotion removals from carts. Used for debugging, compliance, and future user notifications.';

COMMENT ON COLUMN promotion_removal_logs.removal_reason IS 
'Reason for removal: expired (past expires_at), disabled (is_active=false), max_uses_reached (current_uses >= max_uses)';

COMMENT ON COLUMN promotion_removal_logs.promo_metadata IS 
'Full promotion details at time of removal (JSONB). Includes: promotion_id, name, discount_type, discount_value, expires_at, etc.';

COMMENT ON COLUMN promotion_removal_logs.notified IS 
'Flag for future notification system. FALSE = user not yet notified, TRUE = notification sent';

-- ================================================
-- VERIFICATION
-- ================================================

-- Check if table was created successfully
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'promotion_removal_logs'
    ) THEN
        RAISE NOTICE '✓ Migration successful: promotion_removal_logs table created';
    ELSE
        RAISE EXCEPTION '✗ Migration failed: promotion_removal_logs table not found';
    END IF;
END $$;
