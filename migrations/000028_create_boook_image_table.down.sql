-- Rollback migration
DROP TRIGGER IF EXISTS update_book_images_updated_at ON book_images;
DROP INDEX IF EXISTS idx_book_images_one_cover;
DROP INDEX IF EXISTS idx_book_images_sort_order;
DROP INDEX IF EXISTS idx_book_images_status;
DROP INDEX IF EXISTS idx_book_images_book_id;
DROP TABLE IF EXISTS book_images;
