CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Relationships
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Review Content
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title TEXT,
    content TEXT NOT NULL,
    
    -- Images (optional)
    images TEXT[],
    
    -- Verification
    is_verified_purchase BOOLEAN DEFAULT true,
    
    -- Moderation
    is_approved BOOLEAN DEFAULT false,
    is_featured BOOLEAN DEFAULT false,
    admin_note TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- One review per user per book
    UNIQUE(user_id, book_id)
);

-- ================================================-- INDEXES-- ================================================

CREATE INDEX idx_reviews_book ON reviews(book_id, is_approved, created_at DESC);
CREATE INDEX idx_reviews_user ON reviews(user_id);
CREATE INDEX idx_reviews_rating ON reviews(rating);
CREATE INDEX idx_reviews_approved ON reviews(is_approved) WHERE is_approved = true;

-- ================================================-- TRIGGERS-- ================================================

CREATE TRIGGER update_reviews_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-update book rating statistics
CREATE OR REPLACE FUNCTION update_book_rating_stats()
RETURNS TRIGGER AS $$
DECLARE
    book_uuid UUID;
BEGIN
    book_uuid := COALESCE(NEW.book_id, OLD.book_id);
    
    UPDATE books
    SET 
        rating_average = (
            SELECT ROUND(AVG(rating)::numeric, 1)
            FROM reviews
            WHERE book_id = book_uuid AND is_approved = true
        ),
        rating_count = (
            SELECT COUNT(*)
            FROM reviews
            WHERE book_id = book_uuid AND is_approved = true
        )
    WHERE id = book_uuid;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_book_rating_insert
    AFTER INSERT ON reviews
    FOR EACH ROW
    WHEN (NEW.is_approved = true)
    EXECUTE FUNCTION update_book_rating_stats();

CREATE TRIGGER trigger_update_book_rating_update
    AFTER UPDATE OF rating, is_approved ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_book_rating_stats();

CREATE TRIGGER trigger_update_book_rating_delete
    AFTER DELETE ON reviews
    FOR EACH ROW
    WHEN (OLD.is_approved = true)
    EXECUTE FUNCTION update_book_rating_stats();

-- ================================================-- ADD RATING COLUMNS TO BOOKS-- ================================================

ALTER TABLE books 
ADD COLUMN IF NOT EXISTS rating_average NUMERIC(2,1) DEFAULT 0.0,
ADD COLUMN IF NOT EXISTS rating_count INT DEFAULT 0;

CREATE INDEX idx_books_rating ON books(rating_average DESC) WHERE rating_count > 0;

-- ================================================-- COMMENTS-- ================================================

COMMENT ON TABLE reviews IS 'Product reviews and ratings with moderation';
COMMENT ON COLUMN reviews.is_verified_purchase IS 'Review from actual purchase (vs fake review)';
COMMENT ON COLUMN reviews.is_approved IS 'Moderation status (pending/approved)';