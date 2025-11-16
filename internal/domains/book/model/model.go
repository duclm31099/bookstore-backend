package model

import "time"

// BookImage represents một ảnh của book (original + variants)
type BookImage struct {
	ID     string `json:"id" db:"id"`
	BookID string `json:"book_id" db:"book_id"`

	// URLs của các phiên bản ảnh
	OriginalURL  string  `json:"original_url" db:"original_url"`
	LargeURL     *string `json:"large_url" db:"large_url"` // NULL khi chưa xử lý
	MediumURL    *string `json:"medium_url" db:"medium_url"`
	ThumbnailURL *string `json:"thumbnail_url" db:"thumbnail_url"`

	// Metadata
	SortOrder    int     `json:"sort_order" db:"sort_order"`
	IsCover      bool    `json:"is_cover" db:"is_cover"`
	Status       string  `json:"status" db:"status"` // processing | ready | failed
	ErrorMessage *string `json:"error_message,omitempty" db:"error_message"`

	// Technical info (optional)
	Format        *string `json:"format,omitempty" db:"format"`
	Width         *int    `json:"width,omitempty" db:"width"`
	Height        *int    `json:"height,omitempty" db:"height"`
	FileSizeBytes *int64  `json:"file_size_bytes,omitempty" db:"file_size_bytes"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Status constants
const (
	ImageStatusProcessing = "processing"
	ImageStatusReady      = "ready"
	ImageStatusFailed     = "failed"
)

type BookImageInfo struct {
	ID          string `json:"id"`
	OriginalURL string `json:"original_url"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
}
type CreateBookResponse struct {
	ID     string          `json:"id"`
	Title  string          `json:"title"`
	Images []BookImageInfo `json:"images"`
}
