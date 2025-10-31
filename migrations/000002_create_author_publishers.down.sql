-- Drop in reverse order
DROP SEQUENCE IF EXISTS order_number_seq CASCADE;
DROP FUNCTION IF EXISTS generate_order_number() CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
