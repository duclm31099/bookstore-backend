DROP MATERIALIZED VIEW IF EXISTS category_tree CASCADE;
DROP TRIGGER IF EXISTS prevent_self_parent ON categories;
DROP FUNCTION IF EXISTS check_category_parent_not_self() CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
