-- Version: 20251116_002_add_cart_unique_constraints.down.sql

DROP INDEX IF EXISTS idx_carts_user_unique;
DROP INDEX IF EXISTS idx_carts_session_unique;

-- KHÔNG DROP index cũ vì chúng đã tồn tại từ trước
