CREATE OR REPLACE FUNCTION set_default_if_first_address()
RETURNS TRIGGER AS $$
BEGIN
  IF (SELECT COUNT(*) FROM addresses WHERE user_id = NEW.user_id) = 1 THEN
    NEW.is_default = true;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_default_if_first
  BEFORE INSERT ON addresses
  FOR EACH ROW
  EXECUTE FUNCTION set_default_if_first_address();
