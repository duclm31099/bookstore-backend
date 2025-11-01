-- Drop functions
DROP FUNCTION IF EXISTS get_latest_payment_transaction(UUID);
DROP FUNCTION IF EXISTS is_payment_expired(UUID);
DROP FUNCTION IF EXISTS sync_order_payment_status();

-- Drop tables
DROP TABLE IF EXISTS payment_webhook_logs;
DROP TABLE IF EXISTS payment_transactions;
