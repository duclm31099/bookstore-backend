-- ================================================
-- CARTS TABLE
-- ================================================

CREATE TABLE IF NOT EXISTS carts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_id TEXT,
    
    items_count INT DEFAULT 0,
    subtotal NUMERIC(12,2) DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '30 days')
);

-- ================================================
-- CART ITEMS TABLE
-- ================================================

CREATE TABLE IF NOT EXISTS cart_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(10,2) NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(cart_id, book_id)
);

-- ================================================
-- INDEXES
-- ================================================

-- Carts
CREATE INDEX idx_carts_user ON carts(user_id);
CREATE INDEX idx_carts_session ON carts(session_id) WHERE session_id IS NOT NULL;

-- ✅ FIX: Bỏ NOW() khỏi WHERE clause
-- Query sẽ filter expired carts trong application code
CREATE INDEX idx_carts_expires ON carts(expires_at);

-- Cart Items
CREATE INDEX idx_cart_items_cart ON cart_items(cart_id);
CREATE INDEX idx_cart_items_book ON cart_items(book_id);

-- ================================================
-- TRIGGERS
-- ================================================

CREATE TRIGGER update_carts_updated_at
    BEFORE UPDATE ON carts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cart_items_updated_at
    BEFORE UPDATE ON cart_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-update cart totals
CREATE OR REPLACE FUNCTION update_cart_totals()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE carts
    SET 
        items_count = (
            SELECT COALESCE(SUM(quantity), 0)
            FROM cart_items
            WHERE cart_id = COALESCE(NEW.cart_id, OLD.cart_id)
        ),
        subtotal = (
            SELECT COALESCE(SUM(quantity * price), 0)
            FROM cart_items
            WHERE cart_id = COALESCE(NEW.cart_id, OLD.cart_id)
        ),
        updated_at = NOW()
    WHERE id = COALESCE(NEW.cart_id, OLD.cart_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_cart_totals_insert
    AFTER INSERT ON cart_items
    FOR EACH ROW
    EXECUTE FUNCTION update_cart_totals();

CREATE TRIGGER trigger_update_cart_totals_update
    AFTER UPDATE OF quantity, price ON cart_items
    FOR EACH ROW
    EXECUTE FUNCTION update_cart_totals();

CREATE TRIGGER trigger_update_cart_totals_delete
    AFTER DELETE ON cart_items
    FOR EACH ROW
    EXECUTE FUNCTION update_cart_totals();

-- ================================================
-- COMMENTS
-- ================================================

COMMENT ON TABLE carts IS 'Shopping carts for authenticated and anonymous users';
COMMENT ON COLUMN carts.session_id IS 'Session ID for anonymous users (NULL for authenticated)';
COMMENT ON COLUMN carts.expires_at IS 'Cart expiration (30 days from creation)';
COMMENT ON TABLE cart_items IS 'Items in shopping cart with quantity and snapshot price';
COMMENT ON COLUMN cart_items.price IS 'Price snapshot at time of adding (may differ from current book price)';
