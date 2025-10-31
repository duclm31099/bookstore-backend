-- ================================================
-- Categories Table (Hierarchical - Tree Structure)
-- ================================================

CREATE TABLE IF NOT EXISTS categories (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Basic Info
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    
    -- Hierarchy
    parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    sort_order INT DEFAULT 0,
    
    -- Display
    description TEXT,  -- ← Thêm description
    icon_url TEXT,     -- ← Thêm icon cho UI
    is_active BOOLEAN DEFAULT true,  -- ← Thêm để hide categories
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()  -- ← BẠN THIẾU
);

-- ================================================
-- Indexes
-- ================================================

CREATE INDEX idx_categories_slug ON categories(slug);
CREATE INDEX idx_categories_parent ON categories(parent_id);
CREATE INDEX idx_categories_sort ON categories(parent_id, sort_order);  -- ← Composite index tốt hơn
CREATE INDEX idx_categories_active ON categories(is_active) WHERE is_active = true;

-- ================================================
-- Trigger
-- ================================================

CREATE TRIGGER update_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- Constraints & Validation
-- ================================================

-- Prevent circular references (category cannot be its own parent)
CREATE OR REPLACE FUNCTION check_category_parent_not_self()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.id = NEW.parent_id THEN
        RAISE EXCEPTION 'Category cannot be its own parent';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_self_parent
    BEFORE INSERT OR UPDATE ON categories
    FOR EACH ROW
    WHEN (NEW.parent_id IS NOT NULL)
    EXECUTE FUNCTION check_category_parent_not_self();

-- ================================================
-- Materialized View for Category Tree (Performance)
-- ================================================

CREATE MATERIALIZED VIEW category_tree AS
WITH RECURSIVE tree AS (
    -- Root categories
    SELECT 
        id,
        name,
        slug,
        parent_id,
        sort_order,
        1 as level,
        ARRAY[sort_order] as path,
        name::TEXT as full_path
    FROM categories
    WHERE parent_id IS NULL AND is_active = true
    
    UNION ALL
    
    -- Child categories
    SELECT 
        c.id,
        c.name,
        c.slug,
        c.parent_id,
        c.sort_order,
        t.level + 1,
        t.path || c.sort_order,
        t.full_path || ' > ' || c.name
    FROM categories c
    INNER JOIN tree t ON c.parent_id = t.id
    WHERE c.is_active = true
)
SELECT * FROM tree
ORDER BY path;

CREATE UNIQUE INDEX idx_category_tree_id ON category_tree(id);

-- ================================================
-- Comments
-- ================================================

COMMENT ON TABLE categories IS 'Hierarchical product categories (tree structure)';
COMMENT ON COLUMN categories.parent_id IS 'Parent category ID (NULL for root categories)';
COMMENT ON COLUMN categories.sort_order IS 'Display order within same level';
COMMENT ON MATERIALIZED VIEW category_tree IS 'Flattened category hierarchy for fast queries';
