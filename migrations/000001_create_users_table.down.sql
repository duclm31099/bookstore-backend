-- Drop table
DROP TABLE IF EXISTS users CASCADE;

-- Drop trigger function (nếu không table nào dùng)
-- DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop extension (cẩn thận - có thể tables khác dùng)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
