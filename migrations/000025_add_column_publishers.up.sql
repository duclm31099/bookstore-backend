-- +goose Up
-- 2025-11-10 00:00:01 Add description, address, is_active to publishers

ALTER TABLE publishers
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS address TEXT,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Index để tìm nhà xuất bản đang hoạt động nhanh hơn
CREATE INDEX IF NOT EXISTS idx_publishers_is_active 
    ON publishers(is_active) 
    WHERE is_active = true;

-- Comment cho dễ maintain
COMMENT ON COLUMN publishers.description IS 'Mô tả chi tiết về nhà xuất bản';
COMMENT ON COLUMN publishers.address IS 'Địa chỉ trụ sở nhà xuất bản';
COMMENT ON COLUMN publishers.is_active IS 'Nhà xuất bản còn hoạt động hay không';

-- Trigger tự động cập nhật updated_at (nếu bạn đã có function update_updated_at_column())
CREATE TRIGGER trigger_publishers_update_updated_at
    BEFORE UPDATE ON publishers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();