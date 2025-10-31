DROP TRIGGER IF EXISTS trigger_ensure_single_default ON addresses;
DROP FUNCTION IF EXISTS ensure_single_default_address() CASCADE;
DROP TRIGGER IF EXISTS update_addresses_updated_at ON addresses;
DROP TABLE IF EXISTS addresses CASCADE;