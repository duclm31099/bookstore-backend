-- ================================================
-- Publishers Table
-- ================================================

CREATE TABLE IF NOT EXISTS publishers (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Basic Info
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    
    -- Contact
    website TEXT,
    email TEXT,  -- ← Thêm email (useful)
    phone TEXT,  -- ← Thêm phone
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()  -- ← BẠN THIẾU
);

-- ================================================
-- Indexes
-- ================================================

CREATE INDEX idx_publishers_slug ON publishers(slug);
CREATE INDEX idx_publishers_name ON publishers(name);

-- ================================================
-- Trigger
-- ================================================

CREATE TRIGGER update_publishers_updated_at
    BEFORE UPDATE ON publishers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- Comments
-- ================================================

COMMENT ON TABLE publishers IS 'Book publishers information';
COMMENT ON COLUMN publishers.website IS 'Publisher official website URL';
