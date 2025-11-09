-- +goose Down
-- 2025-11-10 00:00:01 Revert add columns to publishers

-- Xóa trigger trước
DROP TRIGGER IF EXISTS trigger_publishers_update_updated_at ON publishers;

-- Xóa index
DROP INDEX IF EXISTS idx_publishers_is_active;

-- Xóa các cột (cẩn thận: nếu có dữ liệu thì sẽ lỗi)
ALTER TABLE publishers
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS is_active;