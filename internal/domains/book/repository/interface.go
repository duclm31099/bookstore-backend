package repository

import (
	"bookstore-backend/internal/domains/book/model"
	"context"
	"time"
)

// RepositoryInterface - Định nghĩa data access methods
type RepositoryInterface interface {
	ListBooks(ctx context.Context, filter *model.BookFilter) ([]model.Book, int, error)
	GetBaseBookByID(ctx context.Context, id string) (*model.BaseBookResponse, error)
	GetBookByID(ctx context.Context, id string) (*model.Book, []model.InventoryDetailDTO, error)
	GetBookByIDForUpdate(ctx context.Context, id string) (*model.Book, error)
	CheckISBNExistsExcept(ctx context.Context, isbn, excludeID string) (bool, error)
	// GetBookBySlug(ctx context.Context, slug string) (*Book, error)
	CreateBook(ctx context.Context, book *model.Book) error
	UpdateBook(ctx context.Context, book *model.Book) error
	CheckBookHasReservedInventory(ctx context.Context, bookID string) (bool, error)
	CheckBookHasActiveOrders(ctx context.Context, bookID string) (bool, error)
	DeleteBook(ctx context.Context, id string) error
	SoftDeleteBook(ctx context.Context, bookID string, deletedAt time.Time) error
	SearchBooks(ctx context.Context, req model.SearchBooksRequest) ([]model.BookSearchResponse, error)
	CheckISBNExists(ctx context.Context, isbn string) (bool, error)
	GenerateUniqueSlug(ctx context.Context, baseSlug string) (string, error)
	IncrementViewCount(ctx context.Context, bookID string) error
	ValidateAuthor(ctx context.Context, authorID string) (bool, error)
	ValidateCategory(ctx context.Context, categoryID string) (bool, error)
	ValidatePublisher(ctx context.Context, publisherID string) (bool, error)
	GetReviewsHighlight(ctx context.Context, bookID string) ([]model.ReviewDTO, error)
}

// BookFilter - Filter object for database query
