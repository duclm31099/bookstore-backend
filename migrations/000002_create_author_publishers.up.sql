-- ================================================
-- Enable Required Extensions
-- ================================================

-- UUID generation function
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ================================================
-- Helper Functions
-- ================================================

-- Auto-update updated_at column on any UPDATE
-- Dùng chung cho tất cả tables có updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Generate unique order number: ORD-YYYYMMDD-XXXX
CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS TEXT AS $$
BEGIN
    RETURN 'ORD-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || LPAD(NEXTVAL('order_number_seq')::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;

-- Create sequence for order numbers
CREATE SEQUENCE IF NOT EXISTS order_number_seq START 1;

-- ================================================
-- Comments
-- ================================================

COMMENT ON FUNCTION update_updated_at_column() IS 'Automatically updates updated_at timestamp on row update';
COMMENT ON FUNCTION generate_order_number() IS 'Generates unique order number in format ORD-YYYYMMDD-XXXX';
