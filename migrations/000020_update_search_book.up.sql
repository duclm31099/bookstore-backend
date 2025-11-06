-- ================================================
-- FIX: Change tsvector config from 'english' to 'simple'
-- Reason: Better support for Vietnamese text
-- ================================================

-- Drop old trigger
DROP TRIGGER IF EXISTS books_search_vector_trigger ON books;
DROP FUNCTION IF EXISTS books_search_vector_update();

-- Recreate function with 'simple' config
CREATE OR REPLACE FUNCTION books_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Recreate trigger
CREATE TRIGGER books_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, description ON books
    FOR EACH ROW
    EXECUTE FUNCTION books_search_vector_update();

-- Update existing records with new config
UPDATE books SET search_vector = 
    setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(description, '')), 'B')
WHERE deleted_at IS NULL;

COMMENT ON COLUMN books.search_vector IS 'Full-text search vector (title:A + description:B) using simple config for Vietnamese support';
