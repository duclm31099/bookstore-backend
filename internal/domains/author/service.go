package author

import (
	"context"

	"github.com/google/uuid"
)

// Service defines business logic operations for Author domain
// This interface abstracts away implementation details and allows:
// 1. Easy mocking in tests
// 2. Multiple implementations (e.g., caching layer, logging decorator)
// 3. Clear separation between business logic and data access
type Service interface {
	// Create creates a new author with generated slug
	// Business rules:
	// - Name must not be empty and <= 255 chars
	// - Slug is auto-generated from name (Vietnamese-friendly)
	// - Slug must be unique (check before insert)
	// - Bio max 5000 chars (if provided)
	// Returns: Created author with ID, slug, version=0
	// Errors: ErrInvalidName, ErrDuplicateSlug
	Create(ctx context.Context, req *CreateAuthorRequest) (*Author, error)

	// GetByID retrieves author by UUID
	// Returns: Author entity with all fields
	// Errors: ErrAuthorNotFound
	GetByID(ctx context.Context, id uuid.UUID) (*Author, error)

	// GetBySlug retrieves author by URL-friendly slug
	// Use case: SEO-friendly URLs (e.g., /authors/nguyen-nhat-anh)
	// Returns: Author entity
	// Errors: ErrAuthorNotFound
	GetBySlug(ctx context.Context, slug string) (*Author, error)

	// GetAll retrieves paginated list of authors with filtering
	// Business rules:
	// - Default limit: 20, max: 100
	// - Default sort: created_at DESC
	// - Search by name is case-insensitive partial match
	// Returns: Authors slice + total count for pagination
	GetAll(ctx context.Context, filter AuthorFilter) ([]Author, int64, error)

	// Update updates existing author with conflict detection
	// Business rules:
	// - Must provide current version (optimistic locking)
	// - Only update non-nil fields (partial update)
	// - If slug changes, check uniqueness
	// - Increment version on success
	// Returns: Updated author with new version
	// Errors: ErrAuthorNotFound, ErrVersionMismatch, ErrDuplicateSlug
	Update(ctx context.Context, id uuid.UUID, req *UpdateAuthorRequest) (*Author, error)

	// Delete removes author (with business rule validation)
	// Business rules:
	// - Cannot delete if author has linked books
	// - Check book_authors junction table first
	// Use case: Admin cleanup of unused authors
	// Errors: ErrAuthorNotFound, ErrAuthorHasBooks
	Delete(ctx context.Context, id uuid.UUID) error

	// BulkDelete removes multiple authors
	// Business rules:
	// - Check each author for linked books
	// - Continue on individual errors (partial success)
	// Returns: Success count + errors for failed items
	BulkDelete(ctx context.Context, ids BulkDeleteRequest) (int, []BulkError, error)

	// Search performs full-text search on author names
	// Business rules:
	// - Case-insensitive
	// - Partial matching
	// - Results ordered by relevance (exact → starts with → contains)
	// Use case: Autocomplete, search bar
	Search(ctx context.Context, query string, filter AuthorFilter) ([]Author, int64, error)

	// GetWithBookCount retrieves author with aggregated book count
	// Use case: Author detail page showing "150 books by this author"
	// Returns: Author + book count (denormalized)
	// Errors: ErrAuthorNotFound
	GetWithBookCount(ctx context.Context, id uuid.UUID) (*Author, int, error)
}
