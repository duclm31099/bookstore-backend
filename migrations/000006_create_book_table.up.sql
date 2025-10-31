CREATE TABLE IF NOT EXISTS books (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    isbn TEXT UNIQUE,
    
    -- Relationships
    author_id UUID NOT NULL REFERENCES authors(id) ON DELETE RESTRICT,
    publisher_id UUID REFERENCES publishers(id) ON DELETE SET NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    
    -- Pricing
    price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    compare_at_price NUMERIC(10,2) CHECK (compare_at_price >= price),
    cost_price NUMERIC(10,2) CHECK (cost_price >= 0),
    
    -- Media
    cover_url TEXT,
    images TEXT[],
    
    -- Content & Specs
    description TEXT,
    pages INT CHECK (pages > 0),
    language TEXT DEFAULT 'vi',
    published_year INT CHECK (published_year >= 1000 AND published_year <= EXTRACT(YEAR FROM NOW()) + 1),
    format TEXT CHECK (format IN ('paperback', 'hardcover', 'ebook')),
    dimensions TEXT,
    weight_grams INT CHECK (weight_grams > 0),
    
    -- eBook Fields
    ebook_file_url TEXT,
    ebook_file_size_mb DECIMAL(5,2),
    ebook_format TEXT CHECK (ebook_format IN ('pdf', 'epub', 'mobi')),
    
    -- Status & Metrics
    is_active BOOLEAN DEFAULT true,
    is_featured BOOLEAN DEFAULT false,
    view_count INT DEFAULT 0,
    sold_count INT DEFAULT 0,
    
    -- SEO
    meta_title TEXT,
    meta_description TEXT,
    meta_keywords TEXT[],
    
    -- Full-text Search
    search_vector tsvector,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ================================================
-- INDEXES
-- ================================================

-- Basic lookups
CREATE INDEX idx_books_slug ON books(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_books_isbn ON books(isbn) WHERE isbn IS NOT NULL;
CREATE INDEX idx_books_active ON books(is_active) 
    WHERE is_active = true AND deleted_at IS NULL;

-- Foreign key indexes (critical for JOIN performance)
CREATE INDEX idx_books_author ON books(author_id);
CREATE INDEX idx_books_publisher ON books(publisher_id);
CREATE INDEX idx_books_category ON books(category_id);

-- Full-text search (GIN index for tsvector)
CREATE INDEX idx_books_search ON books USING GIN(search_vector);

-- Filtering & Sorting
CREATE INDEX idx_books_price ON books(price) WHERE is_active = true;
CREATE INDEX idx_books_format ON books(format);
CREATE INDEX idx_books_language ON books(language);
CREATE INDEX idx_books_featured ON books(is_featured) WHERE is_featured = true;

-- Sorting indexes (FIX: Bỏ DESC khỏi tên index)
CREATE INDEX idx_books_created ON books(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_books_views ON books(view_count DESC);
CREATE INDEX idx_books_sold ON books(sold_count DESC);

-- Composite index for catalog queries
CREATE INDEX idx_books_catalog ON books(category_id, is_active, price) 
    WHERE deleted_at IS NULL;

-- ================================================
-- TRIGGERS
-- ================================================

-- Auto-update search vector on title/description change
CREATE OR REPLACE FUNCTION books_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER books_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, description ON books
    FOR EACH ROW
    EXECUTE FUNCTION books_search_vector_update();

-- Auto-update updated_at
CREATE TRIGGER update_books_updated_at
    BEFORE UPDATE ON books
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- COMMENTS
-- ================================================

COMMENT ON TABLE books IS 'Main products catalog with full-text search and multi-format support';
COMMENT ON COLUMN books.slug IS 'SEO-friendly URL (e.g., nha-gia-kim-paulo-coelho)';
COMMENT ON COLUMN books.isbn IS 'International Standard Book Number (10 or 13 digits)';
COMMENT ON COLUMN books.compare_at_price IS 'Original price for showing discount (must be >= price)';
COMMENT ON COLUMN books.cost_price IS 'Purchase cost from supplier (for margin calculation)';
COMMENT ON COLUMN books.images IS 'Array of additional image URLs';
COMMENT ON COLUMN books.search_vector IS 'Auto-updated full-text search vector (title:A + description:B)';
COMMENT ON COLUMN books.is_featured IS 'Show in homepage featured section';
COMMENT ON COLUMN books.deleted_at IS 'Soft delete (keep for order history)';
