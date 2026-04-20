
-- Create index
CREATE INDEX IF NOT EXISTS idx_carts_user_version ON carts(user_id, version);

-- Drop and recreate trigger
DROP TRIGGER IF EXISTS cart_version_trigger ON carts;

CREATE OR REPLACE FUNCTION increment_cart_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cart_version_trigger
BEFORE UPDATE ON carts
FOR EACH ROW
EXECUTE FUNCTION increment_cart_version();
