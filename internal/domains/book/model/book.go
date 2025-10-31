package model

import (
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// BookFormat represents valid book formats
type BookFormat string

const (
	BookFormatPaperback BookFormat = "paperback"
	BookFormatHardcover BookFormat = "hardcover"
	BookFormatEbook     BookFormat = "ebook"
)

func (f BookFormat) IsValid() bool {
	switch f {
	case BookFormatPaperback, BookFormatHardcover, BookFormatEbook:
		return true
	}
	return false
}

func (f BookFormat) String() string {
	return string(f)
}

// EbookFormat represents valid ebook formats
type EbookFormat string

const (
	EbookFormatPDF  EbookFormat = "pdf"
	EbookFormatEPUB EbookFormat = "epub"
	EbookFormatMOBI EbookFormat = "mobi"
)

func (ef EbookFormat) IsValid() bool {
	switch ef {
	case EbookFormatPDF, EbookFormatEPUB, EbookFormatMOBI:
		return true
	}
	return false
}

func (ef EbookFormat) String() string {
	return string(ef)
}

// Book represents the main book entity
type Book struct {
	// Identity
	ID    uuid.UUID `json:"id" db:"id"`
	Title string    `json:"title" db:"title"`
	Slug  string    `json:"slug" db:"slug"`
	ISBN  *string   `json:"isbn" db:"isbn"`

	// Relationships
	AuthorID    uuid.UUID  `json:"author_id" db:"author_id"`
	PublisherID *uuid.UUID `json:"publisher_id" db:"publisher_id"`
	CategoryID  *uuid.UUID `json:"category_id" db:"category_id"`

	// Pricing
	Price          decimal.Decimal  `json:"price" db:"price"`
	CompareAtPrice *decimal.Decimal `json:"compare_at_price" db:"compare_at_price"`
	CostPrice      *decimal.Decimal `json:"cost_price" db:"cost_price"`

	// Media
	CoverURL *string        `json:"cover_url" db:"cover_url"`
	Images   pq.StringArray `json:"images" db:"images"`

	// Content & Specs
	Description   *string `json:"description" db:"description"`
	Pages         *int    `json:"pages" db:"pages"`
	Language      string  `json:"language" db:"language"`
	PublishedYear *int    `json:"published_year" db:"published_year"`
	Format        *string `json:"format" db:"format"`
	Dimensions    *string `json:"dimensions" db:"dimensions"`
	WeightGrams   *int    `json:"weight_grams" db:"weight_grams"`

	// eBook Fields
	EbookFileURL    *string          `json:"ebook_file_url" db:"ebook_file_url"`
	EbookFileSizeMB *decimal.Decimal `json:"ebook_file_size_mb" db:"ebook_file_size_mb"`
	EbookFormat     *string          `json:"ebook_format" db:"ebook_format"`

	// Status & Metrics
	IsActive   bool `json:"is_active" db:"is_active"`
	IsFeatured bool `json:"is_featured" db:"is_featured"`
	ViewCount  int  `json:"view_count" db:"view_count"`
	SoldCount  int  `json:"sold_count" db:"sold_count"`

	// SEO
	MetaTitle       *string        `json:"meta_title" db:"meta_title"`
	MetaDescription *string        `json:"meta_description" db:"meta_description"`
	MetaKeywords    pq.StringArray `json:"meta_keywords" db:"meta_keywords"`

	// Full-text Search (không cần map vào struct, auto-generated)
	// SearchVector - handled by database trigger

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// BookRequest represents the request payload for creating/updating books
type BookRequest struct {
	// Identity
	Title string  `json:"title" validate:"required,min=1,max=500"`
	Slug  string  `json:"slug" validate:"required,min=1,max=500"`
	ISBN  *string `json:"isbn" validate:"omitempty,isbn"`

	// Relationships
	AuthorID    uuid.UUID  `json:"author_id" validate:"required"`
	PublisherID *uuid.UUID `json:"publisher_id" validate:"omitempty"`
	CategoryID  *uuid.UUID `json:"category_id" validate:"omitempty"`

	// Pricing
	Price          float64  `json:"price" validate:"required,gte=0"`
	CompareAtPrice *float64 `json:"compare_at_price" validate:"omitempty,gtefield=Price"`
	CostPrice      *float64 `json:"cost_price" validate:"omitempty,gte=0"`

	// Media
	CoverURL *string  `json:"cover_url" validate:"omitempty,url"`
	Images   []string `json:"images" validate:"omitempty,dive,url"`

	// Content & Specs
	Description   *string `json:"description" validate:"omitempty,max=10000"`
	Pages         *int    `json:"pages" validate:"omitempty,gt=0"`
	Language      string  `json:"language" validate:"required,oneof=vi en"`
	PublishedYear *int    `json:"published_year" validate:"omitempty,gte=1000"`
	Format        *string `json:"format" validate:"omitempty,oneof=paperback hardcover ebook"`
	Dimensions    *string `json:"dimensions" validate:"omitempty,max=50"`
	WeightGrams   *int    `json:"weight_grams" validate:"omitempty,gt=0"`

	// eBook Fields
	EbookFileURL    *string  `json:"ebook_file_url" validate:"omitempty,url"`
	EbookFileSizeMB *float64 `json:"ebook_file_size_mb" validate:"omitempty,gt=0"`
	EbookFormat     *string  `json:"ebook_format" validate:"omitempty,oneof=pdf epub mobi"`

	// Status & Metrics
	IsActive   bool `json:"is_active"`
	IsFeatured bool `json:"is_featured"`

	// SEO
	MetaTitle       *string  `json:"meta_title" validate:"omitempty,max=100"`
	MetaDescription *string  `json:"meta_description" validate:"omitempty,max=300"`
	MetaKeywords    []string `json:"meta_keywords" validate:"omitempty,max=20,dive,max=50"`
}

// BookResponse represents the response payload for book
type BookResponse struct {
	// Identity
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Slug  string    `json:"slug"`
	ISBN  *string   `json:"isbn,omitempty"`

	// Relationships (populated via JOIN)
	Author    *AuthorResponse    `json:"author,omitempty"`
	Publisher *PublisherResponse `json:"publisher,omitempty"`
	Category  *CategoryResponse  `json:"category,omitempty"`

	// Pricing
	Price          decimal.Decimal  `json:"price"`
	CompareAtPrice *decimal.Decimal `json:"compare_at_price,omitempty"`
	CostPrice      *decimal.Decimal `json:"cost_price,omitempty"`

	// Media
	CoverURL *string  `json:"cover_url,omitempty"`
	Images   []string `json:"images,omitempty"`

	// Content & Specs
	Description   *string `json:"description,omitempty"`
	Pages         *int    `json:"pages,omitempty"`
	Language      string  `json:"language"`
	PublishedYear *int    `json:"published_year,omitempty"`
	Format        *string `json:"format,omitempty"`
	Dimensions    *string `json:"dimensions,omitempty"`
	WeightGrams   *int    `json:"weight_grams,omitempty"`

	// eBook Fields
	EbookFileURL    *string          `json:"ebook_file_url,omitempty"`
	EbookFileSizeMB *decimal.Decimal `json:"ebook_file_size_mb,omitempty"`
	EbookFormat     *string          `json:"ebook_format,omitempty"`

	// Status & Metrics
	IsActive   bool `json:"is_active"`
	IsFeatured bool `json:"is_featured"`
	ViewCount  int  `json:"view_count"`
	SoldCount  int  `json:"sold_count"`

	// SEO
	MetaTitle       *string  `json:"meta_title,omitempty"`
	MetaDescription *string  `json:"meta_description,omitempty"`
	MetaKeywords    []string `json:"meta_keywords,omitempty"`

	// Optional calculated fields
	DiscountPercentage *float64 `json:"discount_percentage,omitempty"`
	TotalStock         *int     `json:"total_stock,omitempty"`
	AverageRating      *float64 `json:"average_rating,omitempty"`
	ReviewCount        *int     `json:"review_count,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BookListResponse represents simplified book for list views
type BookListResponse struct {
	ID         uuid.UUID       `json:"id"`
	Title      string          `json:"title"`
	Slug       string          `json:"slug"`
	AuthorName string          `json:"author_name"`
	Price      decimal.Decimal `json:"price"`
	CoverURL   *string         `json:"cover_url,omitempty"`
	Format     *string         `json:"format,omitempty"`
	IsActive   bool            `json:"is_active"`
	IsFeatured bool            `json:"is_featured"`
	ViewCount  int             `json:"view_count"`
	SoldCount  int             `json:"sold_count"`
}

// BookSearchQuery represents search/filter parameters
type BookSearchQuery struct {
	// Full-text search
	Query string `json:"query" form:"query" validate:"omitempty,max=200"`

	// Filters
	CategoryID  *uuid.UUID `json:"category_id" form:"category_id"`
	AuthorID    *uuid.UUID `json:"author_id" form:"author_id"`
	PublisherID *uuid.UUID `json:"publisher_id" form:"publisher_id"`
	Format      *string    `json:"format" form:"format" validate:"omitempty,oneof=paperback hardcover ebook"`
	Language    *string    `json:"language" form:"language" validate:"omitempty,oneof=vi en"`
	PriceMin    *float64   `json:"price_min" form:"price_min" validate:"omitempty,gte=0"`
	PriceMax    *float64   `json:"price_max" form:"price_max" validate:"omitempty,gte=0"`
	IsActive    *bool      `json:"is_active" form:"is_active"`
	IsFeatured  *bool      `json:"is_featured" form:"is_featured"`

	// Pagination
	Page  int `json:"page" form:"page" validate:"omitempty,gte=1"`
	Limit int `json:"limit" form:"limit" validate:"omitempty,gte=1,lte=100"`

	// Sorting
	Sort string `json:"sort" form:"sort" validate:"omitempty,oneof=price_asc price_desc created_at_desc view_count_desc sold_count_desc"`
}

// AuthorResponse for nested response
type AuthorResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Bio      *string   `json:"bio,omitempty"`
	PhotoURL *string   `json:"photo_url,omitempty"`
}

// PublisherResponse for nested response
type PublisherResponse struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Slug    string    `json:"slug"`
	Website *string   `json:"website,omitempty"`
}

// CategoryResponse for nested response
type CategoryResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	SortOrder int        `json:"sort_order"`
}

// ToResponse converts Book to BookResponse
func (b *Book) ToResponse() *BookResponse {
	resp := &BookResponse{
		ID:              b.ID,
		Title:           b.Title,
		Slug:            b.Slug,
		ISBN:            b.ISBN,
		Price:           b.Price,
		CompareAtPrice:  b.CompareAtPrice,
		CostPrice:       b.CostPrice,
		CoverURL:        b.CoverURL,
		Images:          []string(b.Images),
		Description:     b.Description,
		Pages:           b.Pages,
		Language:        b.Language,
		PublishedYear:   b.PublishedYear,
		Format:          b.Format,
		Dimensions:      b.Dimensions,
		WeightGrams:     b.WeightGrams,
		EbookFileURL:    b.EbookFileURL,
		EbookFileSizeMB: b.EbookFileSizeMB,
		EbookFormat:     b.EbookFormat,
		IsActive:        b.IsActive,
		IsFeatured:      b.IsFeatured,
		ViewCount:       b.ViewCount,
		SoldCount:       b.SoldCount,
		MetaTitle:       b.MetaTitle,
		MetaDescription: b.MetaDescription,
		MetaKeywords:    []string(b.MetaKeywords),
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}

	// Calculate discount percentage if applicable
	if b.HasDiscount() {
		discount := b.DiscountPercentage()
		resp.DiscountPercentage = &discount
	}

	return resp
}

// IsEbook checks if the book is an ebook
func (b *Book) IsEbook() bool {
	return b.Format != nil && *b.Format == string(BookFormatEbook)
}

// HasDiscount checks if book has a discount
func (b *Book) HasDiscount() bool {
	if b.CompareAtPrice == nil {
		return false
	}
	return b.CompareAtPrice.GreaterThan(b.Price)
}

// DiscountPercentage calculates discount percentage
func (b *Book) DiscountPercentage() float64 {
	if !b.HasDiscount() {
		return 0
	}
	discount := b.CompareAtPrice.Sub(b.Price)
	percentage := discount.Div(*b.CompareAtPrice).Mul(decimal.NewFromInt(100))
	result, _ := percentage.Float64()
	return result
}

// ProfitMargin calculates profit margin if cost_price is available
func (b *Book) ProfitMargin() *decimal.Decimal {
	if b.CostPrice == nil || b.CostPrice.IsZero() {
		return nil
	}
	profit := b.Price.Sub(*b.CostPrice)
	margin := profit.Div(*b.CostPrice).Mul(decimal.NewFromInt(100))
	return &margin
}

// Validate validates the book data
func (b *Book) Validate() error {
	// Validate format enum
	if b.Format != nil {
		format := BookFormat(*b.Format)
		if !format.IsValid() {
			return ErrInvalidBookFormat
		}
	}

	// Validate ebook format enum
	if b.EbookFormat != nil {
		ebookFormat := EbookFormat(*b.EbookFormat)
		if !ebookFormat.IsValid() {
			return ErrInvalidEbookFormat
		}
	}

	// Validate price constraints
	if b.Price.LessThan(decimal.Zero) {
		return ErrInvalidPrice
	}

	if b.CompareAtPrice != nil && b.CompareAtPrice.LessThan(b.Price) {
		return ErrCompareAtPriceTooLow
	}

	if b.CostPrice != nil && b.CostPrice.LessThan(decimal.Zero) {
		return ErrInvalidCostPrice
	}

	return nil
}

// Custom errors
var (
	ErrInvalidBookFormat    = NewValidationError("invalid book format")
	ErrInvalidEbookFormat   = NewValidationError("invalid ebook format")
	ErrInvalidPrice         = NewValidationError("price must be >= 0")
	ErrCompareAtPriceTooLow = NewValidationError("compare_at_price must be >= price")
	ErrInvalidCostPrice     = NewValidationError("cost_price must be >= 0")
)

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Scan implements sql.Scanner for custom array handling if needed
func (b *Book) Value() (driver.Value, error) {
	// Implementation if needed for custom types
	return nil, nil
}
