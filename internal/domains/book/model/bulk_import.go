package model

import (
	"time"
)

// ========================================
// BULK IMPORT JOB MODEL (DB)
// ========================================

// Track progress của một bulk import job
// Dùng cho async mode (polling status)
// Errors field là JSONB array chứa []ValidationError

// BulkImportJob represents a bulk import job
type BulkImportJob struct {
	ID            string `json:"id" db:"id"`
	UserID        string `json:"user_id" db:"user_id"`
	FileName      string `json:"file_name" db:"file_name"`
	FileURL       string `json:"file_url" db:"file_url"`
	FileSizeBytes *int64 `json:"file_size_bytes,omitempty" db:"file_size_bytes"`

	TotalRows     int `json:"total_rows" db:"total_rows"`
	ProcessedRows int `json:"processed_rows" db:"processed_rows"`
	SuccessRows   int `json:"success_rows" db:"success_rows"`
	FailedRows    int `json:"failed_rows" db:"failed_rows"`

	Status string `json:"status" db:"status"`           // pending/processing/completed/failed
	Errors []byte `json:"errors,omitempty" db:"errors"` // JSONB

	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Job status constants
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// ========================================
// CSV PARSING MODEL
// ========================================

// Represent một row trong CSV

// Row field để track row number (cho error reporting)

// Pointer types (*string, *int) cho optional fields

// CSVBookRow represents one row from CSV file
type CSVBookRow struct {
	Row           int    `json:"row"` // Row number (for error tracking)
	Title         string `json:"title"`
	AuthorName    string `json:"author_name"`
	CategoryName  string `json:"category_name"`
	PublisherName string `json:"publisher_name"`

	Price          float64  `json:"price"`
	CompareAtPrice *float64 `json:"compare_at_price,omitempty"`
	CostPrice      *float64 `json:"cost_price,omitempty"`

	ISBN        *string `json:"isbn,omitempty"`
	Description *string `json:"description,omitempty"`
	Pages       *int    `json:"pages,omitempty"`
	Language    *string `json:"language,omitempty"`

	PublishedYear *int    `json:"published_year,omitempty"`
	Format        *string `json:"format,omitempty"` // paperback/hardcover/ebook
	Dimensions    *string `json:"dimensions,omitempty"`
	WeightGrams   *int    `json:"weight_grams,omitempty"`

	ImageURLs []string `json:"image_urls"` // Parsed từ image_url_1...image_url_7

	MetaTitle    *string  `json:"meta_title,omitempty"`
	MetaDesc     *string  `json:"meta_description,omitempty"`
	MetaKeywords []string `json:"meta_keywords,omitempty"` // Parsed từ pipe-delimited
}

// ========================================
// VALIDATION ERROR MODEL
// ========================================

// ImportValidationError represents một lỗi validation từ một row
type ImportValidationError struct {
	Row   int    `json:"row"`             // Row number
	Field string `json:"field"`           // Field name có lỗi
	Value string `json:"value,omitempty"` // Giá trị bị lỗi
	Error string `json:"error"`           // Error message
}

// ========================================
// BULK IMPORT RESULT (Response)
// ========================================

// BulkImportResult là response trả về sau khi import
type BulkImportResult struct {
	Success      bool                    `json:"success"`
	TotalRows    int                     `json:"total_rows"`
	SuccessRows  int                     `json:"success_rows,omitempty"`
	FailedRows   int                     `json:"failed_rows,omitempty"`
	Errors       []ImportValidationError `json:"errors,omitempty"`
	CreatedBooks []string                `json:"created_book_ids,omitempty"` // List of book IDs
}

// ========================================
// INTERNAL PROCESSING MODELS
// ========================================

// ImageUploadResult tracks image upload status cho một row
type ImageUploadResult struct {
	RowNumber    int         // CSV row number
	TempKeys     []string    // Temp keys trong MinIO
	ImageRecords []BookImage // Prepared book_image records
	Error        error       // Error nếu có
}

// EntityCache caches các entity đã tạo để reuse
type EntityCache struct {
	Authors    map[string]string // normalized name -> author_id
	Categories map[string]string // normalized name -> category_id
	Publishers map[string]string // normalized name -> publisher_id
}

// NewEntityCache tạo cache rỗng
func NewEntityCache() *EntityCache {
	return &EntityCache{
		Authors:    make(map[string]string),
		Categories: make(map[string]string),
		Publishers: make(map[string]string),
	}
}
