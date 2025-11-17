-- Migration: Add unique constraints (KHÔNG DROP index cũ)
-- Version: 20251116_002_add_cart_unique_constraints.up.sql

-- Thêm unique index mới (không drop cũ)
CREATE UNIQUE INDEX idx_carts_user_unique 
ON carts(user_id) 
WHERE user_id IS NOT NULL AND session_id IS NULL;

CREATE UNIQUE INDEX idx_carts_session_unique 
ON carts(session_id) 
WHERE session_id IS NOT NULL AND user_id IS NULL;

COMMENT ON INDEX idx_carts_user_unique IS 'Ensure one active cart per authenticated user';
COMMENT ON INDEX idx_carts_session_unique IS 'Ensure one active cart per anonymous session';

-- Giữ lại index cũ để hỗ trợ cart merge và các query khác
-- idx_carts_user: Hỗ trợ query tất cả cart by user_id (kể cả cart merge)
-- idx_carts_session: Hỗ trợ query tất cả cart by session_id (kể cả cart merge)
