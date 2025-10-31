CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Recipient Information
    recipient_name TEXT NOT NULL,
    phone TEXT NOT NULL,
    
    -- Vietnam Address Structure
    province TEXT NOT NULL,
    district TEXT NOT NULL,
    ward TEXT NOT NULL,
    street TEXT NOT NULL,
    
    -- Optional details
    address_type TEXT CHECK (address_type IN ('home', 'office', 'other')),
    is_default BOOLEAN DEFAULT false,
    notes TEXT, -- Delivery instructions
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================-- INDEXES-- ================================================

CREATE INDEX idx_addresses_user ON addresses(user_id);

-- Unique constraint: Only 1 default address per user
CREATE UNIQUE INDEX idx_addresses_default_per_user 
    ON addresses(user_id) 
    WHERE is_default = true;

-- Geographic queries (optional, for nearest warehouse)
CREATE INDEX idx_addresses_province ON addresses(province);

-- ================================================-- TRIGGERS-- ================================================

CREATE TRIGGER update_addresses_updated_at
    BEFORE UPDATE ON addresses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-set other addresses to non-default when new default is set
CREATE OR REPLACE FUNCTION ensure_single_default_address()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_default = true THEN
        -- Set all other addresses of this user to non-default
        UPDATE addresses 
        SET is_default = false 
        WHERE user_id = NEW.user_id 
        AND id != NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ensure_single_default
    AFTER INSERT OR UPDATE OF is_default ON addresses
    FOR EACH ROW
    WHEN (NEW.is_default = true)
    EXECUTE FUNCTION ensure_single_default_address();

-- ================================================-- COMMENTS-- ================================================

COMMENT ON TABLE addresses IS 'User delivery addresses with Vietnam address format';
COMMENT ON COLUMN addresses.is_default IS 'Default address for checkout (only one per user)';
COMMENT ON COLUMN addresses.notes IS 'Delivery instructions for shipper';