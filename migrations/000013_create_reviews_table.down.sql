ALTER TABLE books DROP COLUMN IF EXISTS rating_average;
ALTER TABLE books DROP COLUMN IF EXISTS rating_count;
DROP TRIGGER IF EXISTS trigger_update_book_rating_delete ON reviews;
DROP TRIGGER IF EXISTS trigger_update_book_rating_update ON reviews;
DROP TRIGGER IF EXISTS trigger_update_book_rating_insert ON reviews;
DROP FUNCTION IF EXISTS update_book_rating_stats() CASCADE;
DROP TRIGGER IF EXISTS update_reviews_updated_at ON reviews;
DROP TABLE IF EXISTS reviews CASCADE;