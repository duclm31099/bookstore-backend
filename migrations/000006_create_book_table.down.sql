DROP TRIGGER IF EXISTS books_search_vector_trigger ON books;
DROP FUNCTION IF EXISTS books_search_vector_update() CASCADE;
DROP TRIGGER IF EXISTS update_books_updated_at ON books;
DROP TABLE IF EXISTS books CASCADE;