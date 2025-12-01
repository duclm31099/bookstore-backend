package repository

import (
	"bookstore-backend/internal/domains/book/model"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RepositoryInterface - Định nghĩa data access methods
type RepositoryInterface interface {
	ListBooks(ctx context.Context, filter *model.BookFilter) ([]model.Book, int, error)
	GetBaseBookByID(ctx context.Context, id string) (*model.BaseBookResponse, error)
	GetBookByID(ctx context.Context, id string) (*model.BookDetailRes, []model.InventoryDetailDTO, error)
	GetBookByIDForUpdate(ctx context.Context, id string) (*model.Book, error)
	CheckISBNExistsExcept(ctx context.Context, isbn, excludeID string) (bool, error)
	CreateBook(ctx context.Context, book *model.Book) (uuid.UUID, error)
	UpdateBook(ctx context.Context, book *model.Book) error
	CheckBookHasReservedInventory(ctx context.Context, bookID string) (bool, error)
	CheckBookHasActiveOrders(ctx context.Context, bookID string) (bool, error)
	SoftDeleteBook(ctx context.Context, bookID string, deletedAt time.Time) error
	SearchBooks(ctx context.Context, req model.SearchBooksRequest) ([]model.BookSearchResponse, error)
	CheckISBNExists(ctx context.Context, isbn string) (bool, error)
	GenerateUniqueSlug(ctx context.Context, baseSlug string) (string, error)
	IncrementViewCount(ctx context.Context, bookID string) error
	ValidateAuthor(ctx context.Context, authorID string) (bool, error)
	ValidateCategory(ctx context.Context, categoryID string) (bool, error)
	ValidatePublisher(ctx context.Context, publisherID string) (bool, error)
	GetReviewsHighlight(ctx context.Context, bookID string) ([]model.ReviewDTO, error)
	// NEW: Methods for bulk import
	CreateBookWithTx(ctx context.Context, tx pgx.Tx, book *model.Book) error
	FindBySlugWithTx(ctx context.Context, tx pgx.Tx, slug string) (*model.Book, error)
	GetBooksByIDs(ctx context.Context, ids []string) ([]model.Book, error)
}

// BookFilter - Filter object for database query
