-- Table để track bulk import jobs (cho async mode)
CREATE TABLE IF NOT EXISTS bulk_import_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- User info
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- File info
    file_name TEXT NOT NULL,
    file_url TEXT NOT NULL,  -- MinIO URL của file CSV uploaded
    file_size_bytes BIGINT,
    
    -- Progress tracking
    total_rows INTEGER NOT NULL DEFAULT 0,
    processed_rows INTEGER NOT NULL DEFAULT 0,
    success_rows INTEGER NOT NULL DEFAULT 0,
    failed_rows INTEGER NOT NULL DEFAULT 0,
    
    -- Status: pending | processing | completed | failed
    status TEXT NOT NULL DEFAULT 'pending',
    
    -- Error details (JSONB array of ValidationError)
    errors JSONB,
    
    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for querying
CREATE INDEX idx_bulk_import_jobs_user_id ON bulk_import_jobs(user_id);
CREATE INDEX idx_bulk_import_jobs_status ON bulk_import_jobs(status);
CREATE INDEX idx_bulk_import_jobs_created ON bulk_import_jobs(created_at DESC);

-- Trigger để auto-update updated_at
CREATE TRIGGER update_bulk_import_jobs_updated_at
    BEFORE UPDATE ON bulk_import_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE bulk_import_jobs IS 'Tracks bulk import book jobs for async processing';
COMMENT ON COLUMN bulk_import_jobs.status IS 'Job status: pending, processing, completed, failed';
COMMENT ON COLUMN bulk_import_jobs.errors IS 'JSON array of validation errors if failed';
-- Cột quan trọng:

--     file_url: Lưu URL của CSV file trong MinIO (để có thể retry hoặc review)

--     total_rows: Tổng số rows trong CSV (parsed từ file)

--     processed_rows: Số rows đã xử lý (update real-time)

--     success_rows: Số rows tạo book thành công

--     failed_rows: Số rows bị lỗi

--     errors: JSONB array chứa chi tiết lỗi từng row (nếu có)

--     status: Lifecycle của job
