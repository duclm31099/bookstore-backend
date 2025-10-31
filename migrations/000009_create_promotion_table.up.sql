-- ================================================
-- PROMOTIONS TABLE (Độc lập)
-- ================================================

CREATE TABLE IF NOT EXISTS promotions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    
    discount_type TEXT NOT NULL CHECK (discount_type IN ('percentage', 'fixed')),
    discount_value NUMERIC(10,2) NOT NULL CHECK (discount_value > 0),
    max_discount_amount NUMERIC(10,2),
    
    min_order_amount NUMERIC(10,2) DEFAULT 0,
    applicable_category_ids UUID[],
    first_order_only BOOLEAN DEFAULT false,
    
    max_uses INT,
    max_uses_per_user INT DEFAULT 1,
    current_uses INT DEFAULT 0,
    
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CHECK (expires_at > starts_at),
    CHECK (discount_type = 'fixed' OR (discount_type = 'percentage' AND discount_value <= 100))
);

-- Indexes
CREATE UNIQUE INDEX idx_promotions_code_lower ON promotions(LOWER(code));
CREATE INDEX idx_promotions_active ON promotions(is_active, starts_at, expires_at)
    WHERE is_active = true;

-- Trigger
CREATE TRIGGER update_promotions_updated_at
    BEFORE UPDATE ON promotions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE promotions IS 'Promotional codes and discount campaigns';
