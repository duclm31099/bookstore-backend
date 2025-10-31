-- ================================================
-- SEQUENCE
-- ================================================

CREATE SEQUENCE IF NOT EXISTS order_number_seq START 1;

CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS TEXT AS $$
BEGIN
    RETURN 'ORD-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || 
           LPAD(NEXTVAL('order_number_seq')::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- ORDERS TABLE
-- ================================================

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number TEXT UNIQUE NOT NULL DEFAULT generate_order_number(),
    
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE RESTRICT,
    
    -- ✅ NULLABLE: promotions table đã tồn tại rồi
    promotion_id UUID REFERENCES promotions(id) ON DELETE SET NULL,
    
    subtotal NUMERIC(12,2) NOT NULL,
    shipping_fee NUMERIC(10,2) DEFAULT 0,
    discount_amount NUMERIC(10,2) DEFAULT 0,
    total NUMERIC(12,2) NOT NULL,
    
    payment_method TEXT NOT NULL CHECK (payment_method IN ('cod', 'vnpay', 'momo', 'bank_transfer')),
    payment_status TEXT DEFAULT 'pending' CHECK (payment_status IN ('pending', 'paid', 'failed', 'refunded')),
    payment_details JSONB,
    paid_at TIMESTAMPTZ,
    
    status TEXT DEFAULT 'pending' CHECK (status IN (
        'pending', 'confirmed', 'processing', 'shipping', 'delivered', 'cancelled', 'returned'
    )),
    
    tracking_number TEXT,
    estimated_delivery_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    
    customer_note TEXT,
    admin_note TEXT,
    cancellation_reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ
);

-- ================================================
-- ORDER ITEMS TABLE
-- ================================================

CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE RESTRICT,
    
    book_title TEXT NOT NULL,
    book_slug TEXT NOT NULL,
    book_cover_url TEXT,
    author_name TEXT,
    
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(10,2) NOT NULL,
    subtotal NUMERIC(10,2) NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================
-- INDEXES
-- ================================================

CREATE INDEX idx_orders_number ON orders(order_number);
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status, created_at DESC);
CREATE INDEX idx_orders_payment ON orders(payment_status);
CREATE INDEX idx_orders_promotion ON orders(promotion_id);
CREATE INDEX idx_orders_created ON orders(created_at DESC);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_book ON order_items(book_id);

-- ================================================
-- TRIGGERS
-- ================================================

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE orders IS 'Customer orders with payment and delivery tracking';
COMMENT ON TABLE order_items IS 'Order line items with snapshot prices';
