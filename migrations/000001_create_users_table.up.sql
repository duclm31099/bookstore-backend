-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Users table (COMPLETE)
CREATE TABLE IF NOT EXISTS users (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    
    -- Profile
    full_name TEXT NOT NULL,
    phone TEXT,
    
    -- Authorization
    role TEXT NOT NULL DEFAULT 'user' 
        CHECK (role IN ('user', 'admin', 'warehouse', 'cskh')),
    is_active BOOLEAN DEFAULT true,
    
    -- Loyalty
    points INT NOT NULL DEFAULT 0 CHECK (points >= 0),
    
    -- Email Verification
    is_verified BOOLEAN DEFAULT false,
    verification_token TEXT,
    verification_sent_at TIMESTAMPTZ,
    
    -- Password Reset
    reset_token TEXT,
    reset_token_expires_at TIMESTAMPTZ,
    
    -- Activity
    last_login_at TIMESTAMPTZ,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ================================================
-- Indexes
-- ================================================

-- Basic indexes (no issues)
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(is_active, deleted_at) 
    WHERE is_active = true AND deleted_at IS NULL;

-- Verification token (no issue - checking NULL)
CREATE INDEX idx_users_verification_token ON users(verification_token) 
    WHERE verification_token IS NOT NULL;

-- ❌ REMOVED: Index with NOW() - causes IMMUTABLE error
-- CREATE INDEX idx_users_reset_token ON users(reset_token) 
--     WHERE reset_token IS NOT NULL AND reset_token_expires_at > NOW();

-- ✅ FIXED: Simple index without time comparison
CREATE INDEX idx_users_reset_token ON users(reset_token, reset_token_expires_at) 
    WHERE reset_token IS NOT NULL;

-- ================================================
-- Trigger
-- ================================================

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- Comments
-- ================================================

COMMENT ON TABLE users IS 'User accounts with authentication and authorization';
COMMENT ON COLUMN users.role IS 'User role: user (customer), admin, warehouse (staff), cskh (customer service)';
COMMENT ON COLUMN users.points IS 'Loyalty points accumulated from purchases';
COMMENT ON COLUMN users.is_verified IS 'Email verification status';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp (NULL = active)';
COMMENT ON COLUMN users.reset_token IS 'Password reset token (NULL when not in use)';
COMMENT ON COLUMN users.reset_token_expires_at IS 'Reset token expiration timestamp';
