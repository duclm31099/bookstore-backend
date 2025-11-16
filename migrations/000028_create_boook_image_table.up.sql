-- Tạo bảng lưu trữ thông tin ảnh của Book
CREATE TABLE IF NOT EXISTS book_images (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Foreign key đến bảng books
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    
    -- URL các phiên bản ảnh
    original_url TEXT NOT NULL,           -- Ảnh gốc đã upload lên MinIO
    large_url TEXT,                       -- Variant 1200x1200px
    medium_url TEXT,                      -- Variant 600x600px  
    thumbnail_url TEXT,                   -- Variant 300x300px
    
    -- Metadata
    sort_order INTEGER NOT NULL DEFAULT 0,  -- Thứ tự hiển thị (0 = đầu tiên)
    is_cover BOOLEAN DEFAULT false,         -- Đánh dấu ảnh cover chính
    status TEXT NOT NULL DEFAULT 'processing', -- processing | ready | failed
    error_message TEXT,                     -- Lỗi nếu xử lý thất bại
    
    -- Thông tin kỹ thuật (optional, hữu ích cho analytics)
    format TEXT,                            -- jpg, png, webp
    width INTEGER,                          -- Chiều rộng ảnh gốc
    height INTEGER,                         -- Chiều cao ảnh gốc
    file_size_bytes BIGINT,                 -- Kích thước file gốc
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes để tăng tốc query
CREATE INDEX idx_book_images_book_id ON book_images(book_id);
CREATE INDEX idx_book_images_status ON book_images(status);
CREATE INDEX idx_book_images_sort_order ON book_images(book_id, sort_order);

-- Đảm bảo mỗi book chỉ có 1 ảnh cover
CREATE UNIQUE INDEX idx_book_images_one_cover 
    ON book_images(book_id) 
    WHERE is_cover = true;

-- Trigger tự động update updated_at
CREATE TRIGGER update_book_images_updated_at
    BEFORE UPDATE ON book_images
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comment giải thích
COMMENT ON TABLE book_images IS 'Stores book cover images and their variants (thumbnail, medium, large)';
COMMENT ON COLUMN book_images.status IS 'Image processing status: processing (đang xử lý), ready (sẵn sàng), failed (thất bại)';
COMMENT ON COLUMN book_images.is_cover IS 'Marks the primary cover image (only one per book)';
COMMENT ON COLUMN book_images.sort_order IS 'Display order (0 = first image shown)';
