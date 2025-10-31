DROP TRIGGER IF EXISTS trigger_increment_promotion_usage ON promotion_usage;
DROP FUNCTION IF EXISTS increment_promotion_usage() CASCADE;
DROP TABLE IF EXISTS promotion_usage CASCADE;
