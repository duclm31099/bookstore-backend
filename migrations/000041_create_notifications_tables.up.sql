-- ================================================
-- Migration: Create Notifications Tables
-- Purpose: Core notification system infrastructure
-- Version: 000041
-- ================================================

-- WHY THIS SYSTEM?
-- 1. Multi-channel notifications (in-app, email, push)
-- 2. User preference management
-- 3. Delivery tracking and retry mechanism
-- 4. Prevent duplicate notifications
-- 5. Support for future real-time updates

-- ================================================
-- TABLE 1: NOTIFICATIONS
-- ================================================

CREATE TABLE IF NOT EXISTS notifications (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- User reference
    -- WHY CASCADE? If user deleted, their notifications should be deleted too
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Notification content
    type VARCHAR(50) NOT NULL,  -- promotion_removed, order_status, payment, etc.
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    data JSONB,  -- Additional structured data for rendering
    
    -- Read status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    
    -- Delivery status
    is_sent BOOLEAN DEFAULT FALSE,
    sent_at TIMESTAMPTZ,
    
    -- Delivery channels
    -- WHY ARRAY? A notification can be sent via multiple channels
    channels TEXT[] NOT NULL DEFAULT '{in_app}',
    
    -- Per-channel delivery status
    -- WHY JSONB? Flexible tracking: {"email": "sent", "push": "failed", "in_app": "delivered"}
    delivery_status JSONB DEFAULT '{}',
    
    -- Reference to source entity
    -- WHY? Link notification back to order, cart, promotion, etc.
    reference_type VARCHAR(50),  -- order, cart, promotion_removal_log, etc.
    reference_id UUID,
    
    -- Idempotency
    -- WHY? Prevent duplicate notifications from retry logic
    -- Format: {type}:{reference_id}:{user_id}
    idempotency_key VARCHAR(255) UNIQUE,
    
    -- Priority and expiration
    -- WHY PRIORITY? High priority notifications sent first
    -- 1=low (marketing), 2=medium (updates), 3=high (urgent)
    priority INT DEFAULT 2 CHECK (priority BETWEEN 1 AND 3),
    
    -- WHY EXPIRES_AT? Auto-cleanup old notifications
    expires_at TIMESTAMPTZ,
    
    -- Audit timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================
-- TABLE 2: NOTIFICATION PREFERENCES
-- ================================================

CREATE TABLE IF NOT EXISTS notification_preferences (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- User reference (one preference set per user)
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Per-type channel preferences
    -- WHY JSONB? Flexible structure, easy to add new notification types
    -- Structure: {"promotion_removed": {"in_app": true, "email": false, "push": false}, ...}
    preferences JSONB NOT NULL DEFAULT '{
        "promotion_removed": {"in_app": true, "email": false, "push": false},
        "order_status": {"in_app": true, "email": true, "push": true},
        "payment": {"in_app": true, "email": true, "push": false},
        "new_promotion": {"in_app": true, "email": false, "push": false},
        "review_response": {"in_app": true, "email": false, "push": false},
        "system_alert": {"in_app": true, "email": true, "push": false}
    }',
    
    -- Global settings
    -- WHY DO_NOT_DISTURB? User can temporarily disable all notifications
    do_not_disturb BOOLEAN DEFAULT FALSE,
    
    -- WHY QUIET HOURS? Respect user's sleep time (no emails/push during these hours)
    quiet_hours_start TIME,  -- e.g., 22:00
    quiet_hours_end TIME,    -- e.g., 08:00
    
    -- Audit timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================
-- INDEXES FOR PERFORMANCE
-- ================================================

-- Index 1: User's notifications (most common query)
-- USE CASE: "Show me my notifications, newest first"
-- WHY DESC? Recent notifications queried more often
CREATE INDEX idx_notifications_user 
ON notifications(user_id, created_at DESC);

-- Index 2: Unread notifications
-- USE CASE: "Show unread count badge"
-- WHY PARTIAL INDEX? Only index unread notifications (smaller, faster)
CREATE INDEX idx_notifications_unread 
ON notifications(user_id, is_read) 
WHERE is_read = FALSE;

-- Index 3: Unsent notifications (for background job)
-- USE CASE: "Find notifications that need to be sent"
-- WHY PARTIAL INDEX? Only index unsent notifications
CREATE INDEX idx_notifications_unsent 
ON notifications(is_sent, created_at) 
WHERE is_sent = FALSE;

-- Index 4: Notification type (for analytics)
-- USE CASE: "How many order_status notifications sent today?"
CREATE INDEX idx_notifications_type 
ON notifications(type, created_at DESC);

-- Index 5: Reference lookup
-- USE CASE: "Find notification for this order"
CREATE INDEX idx_notifications_reference 
ON notifications(reference_type, reference_id);

-- Index 6: Idempotency check
-- Already has UNIQUE constraint, which creates index automatically

-- Index 7: Expired notifications cleanup
-- USE CASE: Background job to delete expired notifications
CREATE INDEX idx_notifications_expired 
ON notifications(expires_at) 
WHERE expires_at IS NOT NULL;

-- ================================================
-- TRIGGERS
-- ================================================

-- Trigger 1: Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_notification_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_notification_timestamp
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_updated_at();

CREATE TRIGGER trigger_update_preference_timestamp
    BEFORE UPDATE ON notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_updated_at();

-- Trigger 2: Auto-set read_at when is_read changes to true
CREATE OR REPLACE FUNCTION set_notification_read_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_read = TRUE AND OLD.is_read = FALSE THEN
        NEW.read_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_read_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION set_notification_read_at();

-- Trigger 3: Auto-set sent_at when is_sent changes to true
CREATE OR REPLACE FUNCTION set_notification_sent_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_sent = TRUE AND OLD.is_sent = FALSE THEN
        NEW.sent_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_sent_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION set_notification_sent_at();

-- ================================================
-- DEFAULT PREFERENCES FOR EXISTING USERS
-- ================================================

-- WHY? Existing users don't have preferences yet
-- Create default preferences for all existing users
INSERT INTO notification_preferences (user_id)
SELECT id FROM users
WHERE id NOT IN (SELECT user_id FROM notification_preferences)
ON CONFLICT (user_id) DO NOTHING;

-- ================================================
-- COMMENTS FOR DOCUMENTATION
-- ================================================

COMMENT ON TABLE notifications IS 
'Stores all notifications sent to users across multiple channels (in-app, email, push)';

COMMENT ON COLUMN notifications.type IS 
'Notification type: promotion_removed, order_status, payment, new_promotion, review_response, system_alert';

COMMENT ON COLUMN notifications.channels IS 
'Delivery channels for this notification: {in_app, email, push}';

COMMENT ON COLUMN notifications.delivery_status IS 
'Per-channel delivery status (JSONB): {"email": "sent", "push": "failed", "in_app": "delivered"}';

COMMENT ON COLUMN notifications.idempotency_key IS 
'Unique key to prevent duplicate notifications. Format: {type}:{reference_id}:{user_id}';

COMMENT ON COLUMN notifications.priority IS 
'Priority level: 1=low (marketing), 2=medium (updates), 3=high (urgent)';

COMMENT ON TABLE notification_preferences IS 
'User preferences for notification delivery channels per notification type';

COMMENT ON COLUMN notification_preferences.preferences IS 
'JSONB structure defining which channels are enabled for each notification type';

COMMENT ON COLUMN notification_preferences.quiet_hours_start IS 
'Start of quiet hours (no email/push notifications). Example: 22:00';

COMMENT ON COLUMN notification_preferences.quiet_hours_end IS 
'End of quiet hours. Example: 08:00';

-- ================================================
-- VERIFICATION
-- ================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'notifications'
    ) AND EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'notification_preferences'
    ) THEN
        RAISE NOTICE '✓ Migration successful: notification tables created';
    ELSE
        RAISE EXCEPTION '✗ Migration failed: tables not found';
    END IF;
END $$;
