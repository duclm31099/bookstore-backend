-- migrations/000xxx_create_checkout_idempotency.down.sql

DROP INDEX IF EXISTS idx_idempotency_cleanup;
DROP INDEX IF EXISTS idx_idempotency_expires;
DROP INDEX IF EXISTS idx_idempotency_user_created;
DROP TABLE IF EXISTS checkout_idempotency;
