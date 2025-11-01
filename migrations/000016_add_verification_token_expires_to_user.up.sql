-- migrations/0003_add_verification_token_expires_at_to_users.up.sql

ALTER TABLE users ADD COLUMN verification_token_expires_at TIMESTAMPTZ;

-- Optional: Add comment for documentation
COMMENT ON COLUMN users.verification_token_expires_at IS 
'Expiration time for verification token. Set when token is generated, cleared when verified.';
