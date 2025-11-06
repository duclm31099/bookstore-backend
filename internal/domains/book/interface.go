package book

import (
	"context"
	"time"
)

// ServiceInterface - Định nghĩa business logic methods
type ServiceInterface interface {
	ListBooks(ctx context.Context, req ListBooksRequest) ([]ListBooksResponse, *PaginationMeta, error)
	GetBookDetail(ctx context.Context, id string) (*BookDetailResponse, error)
	CreateBook(ctx context.Context, req CreateBookRequest) error
	UpdateBook(ctx context.Context, id string, req UpdateBookRequest) (*BookDetailResponse, error)
	DeleteBook(ctx context.Context, id string) (*DeleteBookResponse, error)
	SearchBooks(ctx context.Context, req SearchBooksRequest) ([]BookSearchResponse, error)
}

// RepositoryInterface - Định nghĩa data access methods
type RepositoryInterface interface {
	ListBooks(ctx context.Context, filter *BookFilter) ([]Book, int, error)
	GetBaseBookByID(ctx context.Context, id string) (*BaseBookResponse, error)
	GetBookByID(ctx context.Context, id string) (*Book, []InventoryDetailDTO, error)
	GetBookByIDForUpdate(ctx context.Context, id string) (*Book, error)
	CheckISBNExistsExcept(ctx context.Context, isbn, excludeID string) (bool, error)
	// GetBookBySlug(ctx context.Context, slug string) (*Book, error)
	CreateBook(ctx context.Context, book *Book) error
	UpdateBook(ctx context.Context, book *Book) error
	CheckBookHasReservedInventory(ctx context.Context, bookID string) (bool, error)
	CheckBookHasActiveOrders(ctx context.Context, bookID string) (bool, error)
	DeleteBook(ctx context.Context, id string) error
	SoftDeleteBook(ctx context.Context, bookID string, deletedAt time.Time) error
	SearchBooks(ctx context.Context, req SearchBooksRequest) ([]BookSearchResponse, error)
	CheckISBNExists(ctx context.Context, isbn string) (bool, error)
	GenerateUniqueSlug(ctx context.Context, baseSlug string) (string, error)
	IncrementViewCount(ctx context.Context, bookID string) error
	ValidateAuthor(ctx context.Context, authorID string) (bool, error)
	ValidateCategory(ctx context.Context, categoryID string) (bool, error)
	ValidatePublisher(ctx context.Context, publisherID string) (bool, error)
	GetReviewsHighlight(ctx context.Context, bookID string) ([]ReviewDTO, error)
}

// CacheInterface - Định nghĩa cache operations
type CacheInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl int) error
	Delete(ctx context.Context, key string) error
	FlushPattern(ctx context.Context, pattern string) error
}

// BookFilter - Filter object for database query
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
