package model

import (
	"bookstore-backend/internal/shared/utils"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// ============ ENTITIES ============

// Book - Domain Entity (from database)
type Book struct {
	// Identity
	ID      uuid.UUID `json:"id" db:"id"`
	Title   string    `json:"title" db:"title"`
	Slug    string    `json:"slug" db:"slug"`
	ISBN    string    `json:"isbn" db:"isbn"`
	Version int       `json:"version" db:"version"`
	// Relationships
	AuthorID    uuid.UUID `json:"author_id" db:"author_id"`
	PublisherID uuid.UUID `json:"publisher_id" db:"publisher_id"`
	CategoryID  uuid.UUID `json:"category_id" db:"category_id"`

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

	//
	RatingAverage float64 `json:"rating_average" db:"rating_average"`
	RatingCount   int     `json:"rating_count" db:"rating_count"`

	// Joined data (chỉ dùng khi query JOIN)
	AuthorName    string `json:"author_name" db:"author_name"`
	CategoryName  string `json:"category_name" db:"category_name"`
	PublisherName string `json:"publisher_name" db:"publisher_name"`
	TotalStock    int    `json:"total_stock" db:"total_stock"`
}

// ============ DTOs ============

// ListBooksRequest - Query parameters
type ListBooksRequest struct {
	Search     string  `form:"search"`    // Full-text search
	CategoryID string  `form:"category"`  // Filter by category
	PriceMin   float64 `form:"price_min"` // Price range
	PriceMax   float64 `form:"price_max"`
	Language   string  `form:"language"`              // Filter by language
	Sort       string  `form:"sort" default:"newest"` // newest, price_asc, price_desc, popular
	Page       int     `form:"page" default:"1"`      // Pagination
	Limit      int     `form:"limit" default:"20"`    // Max 100
	IsActive   *bool   `form:"is_active"`             // Optional: filter active/inactive
}

// ListBooksResponse - Response data
type ListBooksResponse struct {
	ID             uuid.UUID        `json:"id"`
	Title          string           `json:"title"`
	Slug           string           `json:"slug"`
	AuthorName     string           `json:"author_name"`
	PublisherName  string           `json:"publisher_name"`
	Price          decimal.Decimal  `json:"price"`
	CompareAtPrice *decimal.Decimal `json:"compare_at_price,omitempty"`
	CoverURL       *string          `json:"cover_url,omitempty"`
	Language       string           `json:"language"`
	Format         *string          `json:"format,omitempty"`
	RatingAverage  float64          `json:"rating_average"`
	RatingCount    int              `json:"rating_count"`
	ViewCount      int              `json:"view_count"`
	SoldCount      int              `json:"sold_count"`
	IsFeatured     bool             `json:"is_featured"`
	TotalStock     int              `json:"total_stock"`
	CreatedAt      time.Time        `json:"created_at"`
}

// PaginationMeta - Metadata for pagination
type PaginationMeta struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

// ListBooksAPIResponse - Wrapper response
type ListBooksAPIResponse struct {
	Books      []ListBooksResponse `json:"books"`
	Pagination PaginationMeta      `json:"pagination"`
}

// Helper: Validate list request
func ValidateListRequest(req ListBooksRequest) error {
	if req.Page < 1 || req.Limit < 1 {
		return ErrInvalidPageLimit
	}
	if req.Limit > 100 {
		req.Limit = 100 // Cap at 100
	}
	if req.PriceMin < 0 || req.PriceMax < 0 {
		return ErrInvalidPriceRange
	}
	if req.PriceMin > 0 && req.PriceMax > 0 && req.PriceMin > req.PriceMax {
		return ErrInvalidPriceRange
	}
	validSorts := map[string]bool{"newest": true, "price_asc": true, "price_desc": true, "popular": true, "rating": true}
	if req.Sort != "" && !validSorts[req.Sort] {
		return ErrInvalidSort
	}
	return nil
}

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
	ISBN  string    `json:"isbn"`

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
type BaseBookResponse struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

type DeleteBookResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	DeletedAt time.Time `json:"deleted_at"`
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

// inventories
type InventoryDetailDTO struct {
	Location          string     `json:"location"`
	Quantity          int        `json:"quantity"`
	ReservedQuantity  int        `json:"reserved_quantity"`
	AvailableQuantity int        `json:"available_quantity"`
	LowStockThreshold int        `json:"low_stock_threshold"`
	IsLowStock        bool       `json:"is_low_stock"`
	LastRestockAt     *time.Time `json:"last_restock_at,omitempty"`
}

// book detail response
type BookDetailResponse struct {
	ID            uuid.UUID            `json:"id"`
	Title         string               `json:"title"`
	Author        *AuthorDTO           `json:"author,omitempty"`
	Category      *CategoryDTO         `json:"category,omitempty"`
	Publisher     *PublisherDTO        `json:"publisher,omitempty"`
	Price         float64              `json:"price"`
	Language      string               `json:"language"`
	Description   *string              `json:"description,omitempty"`
	CoverURL      *string              `json:"cover_url,omitempty"`
	PublishedYear *int                 `json:"published_year,omitempty"`
	Format        *string              `json:"format,omitempty"`
	TotalStock    int                  `json:"total_stock"`
	Inventories   []InventoryDetailDTO `json:"inventories"`
	RatingAverage float64              `json:"rating_average"`
	RatingCount   int                  `json:"rating_count"`
	Reviews       []ReviewDTO          `json:"reviews"`
}
type BookFilter struct {
	Search     string
	CategoryID string
	PriceMin   float64
	PriceMax   float64
	Language   string
	Sort       string
	Offset     int
	Limit      int
	IsActive   *bool
}

// các DTO liên kết

// Author
type AuthorDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
	Bio  *string   `json:"bio,omitempty"`
}

// Category
type CategoryDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// Publisher
type PublisherDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// Review
type ReviewDTO struct {
	ID        uuid.UUID `json:"id"`
	UserName  string    `json:"user_name"`
	Rating    float64   `json:"rating"`
	Content   string    `json:"content"`
	CreatedAt string    `json:"created_at"`
	Title     int       `json:"title"`
}

// CreateBookRequest - DTO cho create book
type CreateBookRequest struct {
	// Basic info
	Title       string `json:"title" binding:"required,min=1,max=500"`
	ISBN        string `json:"isbn" binding:"omitempty,len=10,numeric"`
	AuthorID    string `json:"author_id" binding:"required,uuid"`
	PublisherID string `json:"publisher_id" binding:"omitempty,uuid"`
	CategoryID  string `json:"category_id" binding:"omitempty,uuid"`

	// Pricing
	Price          float64  `json:"price" binding:"required,gt=0"`
	CompareAtPrice *float64 `json:"compare_at_price" binding:"omitempty,gtefield=Price"`
	CostPrice      *float64 `json:"cost_price" binding:"omitempty,gte=0"`

	// Content
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url" binding:"omitempty,url"`
	Images      []string `json:"images"`

	// Book details
	Pages         *int    `json:"pages" binding:"omitempty,gt=0"`
	Language      string  `json:"language" binding:"required,oneof=vi en"`
	PublishedYear *int    `json:"published_year" binding:"omitempty"`
	Format        *string `json:"format" binding:"omitempty,oneof=paperback hardcover ebook"`
	Dimensions    *string `json:"dimensions"`
	WeightGrams   *int    `json:"weight_grams" binding:"omitempty,gt=0"`

	// Ebook
	EbookFileURL    *string  `json:"ebook_file_url" binding:"omitempty,url"`
	EbookFileSizeMb *float64 `json:"ebook_file_size_mb" binding:"omitempty,gt=0"`
	EbookFormat     *string  `json:"ebook_format" binding:"omitempty,oneof=pdf epub mobi"`

	// Flags
	IsActive   bool `json:"is_active"`
	IsFeatured bool `json:"is_featured"`

	// SEO
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	MetaKeywords    []string `json:"meta_keywords"`
}

// validateCreateRequest - Business validation
func ValidateCreateRequest(req *CreateBookRequest) error {
	// Validate published year: không được lớn hơn năm hiện tại
	if req.PublishedYear != nil {
		currentYear := time.Now().Year()
		if *req.PublishedYear < 1000 || *req.PublishedYear > currentYear {
			return ErrInvalidPublishedYear
		}
	}

	// Validate compare_at_price >= price
	if req.CompareAtPrice != nil && *req.CompareAtPrice < req.Price {
		return ErrInvalidPriceRange
	}

	return nil
}

// generateBookDetailCacheKey - Tạo cache key cho book detail
func GenerateBookDetailCacheKey(bookID string) string {
	return "book:detail:" + bookID
}

// UpdateBookRequest - DTO cho update book
// Tất cả field là optional (dùng pointer để detect nil)
type UpdateBookRequest struct {
	// Basic info
	Title       *string `json:"title" binding:"omitempty,min=1,max=500"`
	ISBN        *string `json:"isbn" binding:"omitempty,len=10,numeric"`
	AuthorID    *string `json:"author_id" binding:"omitempty,uuid"`
	PublisherID *string `json:"publisher_id" binding:"omitempty,uuid"`
	CategoryID  *string `json:"category_id" binding:"omitempty,uuid"`

	// Pricing
	Price          *float64 `json:"price" binding:"omitempty,gt=0"`
	CompareAtPrice *float64 `json:"compare_at_price" binding:"omitempty"`
	CostPrice      *float64 `json:"cost_price" binding:"omitempty,gte=0"`

	// Content
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url" binding:"omitempty,url"`
	Images      []string `json:"images"`

	// Book details
	Pages         *int    `json:"pages" binding:"omitempty,gt=0"`
	Language      *string `json:"language" binding:"omitempty,oneof=vi en"`
	PublishedYear *int    `json:"published_year" binding:"omitempty"`
	Format        *string `json:"format" binding:"omitempty,oneof=paperback hardcover ebook"`
	Dimensions    *string `json:"dimensions"`
	WeightGrams   *int    `json:"weight_grams" binding:"omitempty,gt=0"`

	// Ebook
	EbookFileURL    *string  `json:"ebook_file_url" binding:"omitempty,url"`
	EbookFileSizeMb *float64 `json:"ebook_file_size_mb" binding:"omitempty,gt=0"`
	EbookFormat     *string  `json:"ebook_format" binding:"omitempty,oneof=pdf epub mobi"`

	// Flags
	IsActive   *bool `json:"is_active"`
	IsFeatured *bool `json:"is_featured"`

	// SEO
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	MetaKeywords    []string `json:"meta_keywords"`

	// Optimistic locking
	Version int `json:"version" binding:"required,gte=0"`
}

// validateUpdateRequest - Business validation cho update
func ValidateUpdateRequest(req *UpdateBookRequest) error {
	// Validate published year
	if req.PublishedYear != nil {
		currentYear := time.Now().Year()
		if *req.PublishedYear < 1000 || *req.PublishedYear > currentYear {
			return ErrInvalidPublishedYear
		}
	}

	// Validate compare_at_price >= price (nếu cả 2 đều được update)
	if req.CompareAtPrice != nil && req.Price != nil {
		if *req.CompareAtPrice < *req.Price {
			return ErrInvalidPriceRange
		}
	}

	return nil
}

// Helper: Apply updates to existing book
func ApplyUpdates(existing Book, req UpdateBookRequest, newSlug string) {
	if req.Title != nil {
		existing.Title = *req.Title
		existing.Slug = newSlug
	}
	if req.ISBN != nil {
		existing.ISBN = *req.ISBN
	}
	if req.AuthorID != nil {
		existing.AuthorID = utils.ParseStringToUUID(*req.AuthorID)
	}
	if req.PublisherID != nil {
		existing.PublisherID = utils.ParseStringToUUID(*req.PublisherID)
	}
	if req.CategoryID != nil {
		existing.CategoryID = utils.ParseStringToUUID(*req.CategoryID)
	}
	if req.Price != nil {
		existing.Price = decimal.NewFromFloat(*req.Price)
	}
	if req.CompareAtPrice != nil {
		existing.CompareAtPrice = utils.ParseFloatToDecimal(req.CompareAtPrice)
	}
	if req.CostPrice != nil {
		existing.CostPrice = utils.ParseFloatToDecimal(req.CostPrice)
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.CoverURL != nil {
		existing.CoverURL = req.CoverURL
	}
	if req.Images != nil {
		existing.Images = req.Images
	}
	if req.Pages != nil {
		existing.Pages = req.Pages
	}
	if req.Language != nil {
		existing.Language = *req.Language
	}
	if req.PublishedYear != nil {
		existing.PublishedYear = req.PublishedYear
	}
	if req.Format != nil {
		existing.Format = req.Format
	}
	if req.Dimensions != nil {
		existing.Dimensions = req.Dimensions
	}
	if req.WeightGrams != nil {
		existing.WeightGrams = req.WeightGrams
	}
	if req.EbookFileURL != nil {
		existing.EbookFileURL = req.EbookFileURL
	}
	if req.EbookFileSizeMb != nil {
		decimalValue := decimal.NewFromFloat(*req.EbookFileSizeMb)
		existing.EbookFileSizeMB = &decimalValue
	}
	if req.EbookFormat != nil {
		existing.EbookFormat = req.EbookFormat
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.IsFeatured != nil {
		existing.IsFeatured = *req.IsFeatured
	}
	if req.MetaTitle != nil {
		existing.MetaTitle = req.MetaTitle
	}
	if req.MetaDescription != nil {
		existing.MetaDescription = req.MetaDescription
	}
	if req.MetaKeywords != nil {
		existing.MetaKeywords = req.MetaKeywords
	}

	// Update metadata
	existing.Version++
	existing.UpdatedAt = time.Now()
}

//	-================================== SEARCH DTO ============================
//
// SearchBooksRequest - Query parameters for search
type SearchBooksRequest struct {
	Query    string `form:"q" binding:"required,min=2,max=200"`
	Language string `form:"language" binding:"omitempty,oneof=vi en"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// BookSearchResponse - Simplified book info for search results
type BookSearchResponse struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Slug       string  `json:"slug"`
	AuthorName string  `json:"author_name"`
	CoverURL   *string `json:"cover_url,omitempty"`
	Price      float64 `json:"price"`
	Language   string  `json:"language"`
	Rank       float64 `json:"rank"` // Relevance score for debugging
}

// SearchBooksAPIResponse - Wrapper for search results
type SearchBooksAPIResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    []BookSearchResponse `json:"data"`
	Meta    *SearchMeta          `json:"meta,omitempty"`
}

// SearchMeta - Metadata about search results
type SearchMeta struct {
	Query       string `json:"query"`
	ResultCount int    `json:"result_count"`
	TookMs      int64  `json:"took_ms"` // Query execution time
}
