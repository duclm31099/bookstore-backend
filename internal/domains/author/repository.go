package author

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for Author data access operations
// This abstraction allows:
// 1. Easy testing via mocking
// 2. Swapping database implementations
// 3. Clear separation of concerns
type Repository interface {
	// Create inserts a new author
	// Returns: created author with ID, timestamps, version=0
	// Errors: ErrDuplicateSlug if slug exists
	Create(ctx context.Context, author *Author) (*Author, error)

	// GetByID retrieves author by UUID
	// Returns: ErrAuthorNotFound if not exists
	GetByID(ctx context.Context, id uuid.UUID) (*Author, error)

	// GetBySlug retrieves author by URL slug
	// Returns: ErrAuthorNotFound if not exists
	GetBySlug(ctx context.Context, slug string) (*Author, error)

	// GetAll retrieves paginated list of authors
	// Supports: sorting, filtering
	// Returns: authors slice + total count for pagination
	GetAll(ctx context.Context, filter AuthorFilter) ([]Author, int64, error)

	// Update updates an existing author with optimistic locking
	// Version check: currentVersion must match DB version
	// Returns: updated author with incremented version
	// Errors: ErrVersionMismatch if conflict, ErrAuthorNotFound if not exists
	Update(ctx context.Context, author *Author, currentVersion int) (*Author, error)

	// Delete removes author by ID
	// Business rule: should check for linked books in service layer first
	// Returns: ErrAuthorNotFound if not exists
	Delete(ctx context.Context, id uuid.UUID) error

	// BulkDelete removes multiple authors
	// Returns: count of successfully deleted + errors for failed ones
	BulkDelete(ctx context.Context, ids []uuid.UUID) (int, []BulkError, error)

	// ExistsByID checks if author exists
	// Useful for validation without fetching full data
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// ExistsBySlug checks if slug is taken
	// Useful for uniqueness validation before insert/update
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// GetBookCount returns number of books by author
	// For denormalized data in AuthorDetailResponse
	GetBookCount(ctx context.Context, authorID uuid.UUID) (int, error)

	// Search performs full-text search on author names
	// Supports: partial matching, pagination
	Search(ctx context.Context, query string, filter AuthorFilter) ([]Author, int64, error)
}
