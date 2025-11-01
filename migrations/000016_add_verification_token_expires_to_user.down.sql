-- migrations/0003_add_verification_token_expires_at_to_users.down.sql

ALTER TABLE users DROP COLUMN IF EXISTS verification_token_expires_at;
