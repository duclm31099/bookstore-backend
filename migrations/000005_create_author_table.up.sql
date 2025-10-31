-- ================================================
-- Authors Table
-- ================================================

CREATE TABLE IF NOT EXISTS authors (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Basic Info
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    
    -- Details
    bio TEXT,
    photo_url TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()  -- ← BẠN THIẾU CÁI NÀY
);

-- ================================================
-- Indexes
-- ================================================

CREATE INDEX idx_authors_slug ON authors(slug);
CREATE INDEX idx_authors_name ON authors(name);  -- ← Thêm để search by name

-- ================================================
-- Trigger
-- ================================================

CREATE TRIGGER update_authors_updated_at
    BEFORE UPDATE ON authors
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- Comments
-- ================================================

COMMENT ON TABLE authors IS 'Book authors/writers information';
COMMENT ON COLUMN authors.slug IS 'URL-friendly unique identifier';
COMMENT ON COLUMN authors.bio IS 'Author biography (supports Markdown)';
