-- ================================================
-- Rollback Migration: Drop Notifications Tables
-- ================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_set_sent_at ON notifications;
DROP TRIGGER IF EXISTS trigger_set_read_at ON notifications;
DROP TRIGGER IF EXISTS trigger_update_notification_timestamp ON notifications;
DROP TRIGGER IF EXISTS trigger_update_preference_timestamp ON notification_preferences;

-- Drop trigger functions
DROP FUNCTION IF EXISTS set_notification_sent_at();
DROP FUNCTION IF EXISTS set_notification_read_at();
DROP FUNCTION IF EXISTS update_notification_updated_at();

-- Drop indexes (CASCADE will handle this, but explicit is clearer)
DROP INDEX IF EXISTS idx_notifications_user;
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_unsent;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_reference;
DROP INDEX IF EXISTS idx_notifications_expired;

-- Drop tables
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
