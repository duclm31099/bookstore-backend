-- ================================================
-- PROMOTION USAGE TABLE
-- ================================================

CREATE TABLE IF NOT EXISTS promotion_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    promotion_id UUID NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- ✅ Giờ orders table đã tồn tại!
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    discount_amount NUMERIC(10,2) NOT NULL,
    used_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(promotion_id, order_id)
);

-- ================================================
-- INDEXES
-- ================================================

CREATE INDEX idx_promotion_usage_promotion ON promotion_usage(promotion_id);
CREATE INDEX idx_promotion_usage_user ON promotion_usage(user_id);
CREATE INDEX idx_promotion_usage_order ON promotion_usage(order_id);

-- ================================================
-- TRIGGER: Auto-increment usage counter
-- ================================================

CREATE OR REPLACE FUNCTION increment_promotion_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE promotions
    SET current_uses = current_uses + 1
    WHERE id = NEW.promotion_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_increment_promotion_usage
    AFTER INSERT ON promotion_usage
    FOR EACH ROW
    EXECUTE FUNCTION increment_promotion_usage();

COMMENT ON TABLE promotion_usage IS 'Track which users used which promotions in which orders';
