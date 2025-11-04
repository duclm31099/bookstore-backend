package publisher

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines all data access operations for Publisher domain
type Repository interface {
	// Create inserts a new publisher record
	// Returns the created Publisher with generated ID
	Create(ctx context.Context, publisher *Publisher) (*Publisher, error)

	// GetByID retrieves a publisher by ID
	// Returns nil if not found
	GetByID(ctx context.Context, id uuid.UUID) (*Publisher, error)

	// GetBySlug retrieves a publisher by slug
	// Returns nil if not found
	GetBySlug(ctx context.Context, slug string) (*Publisher, error)

	// List retrieves all publishers with pagination
	// offset: starting position (0-based)
	// limit: number of records to return
	List(ctx context.Context, offset, limit int) ([]*Publisher, error)

	// Count returns total number of publishers
	Count(ctx context.Context) (int, error)

	// Update updates publisher information (except slug & id)
	// Returns updated Publisher
	Update(ctx context.Context, id uuid.UUID, publisher *Publisher) (*Publisher, error)

	// Delete removes a publisher record
	Delete(ctx context.Context, id uuid.UUID) error

	// GetWithBooks retrieves publisher with associated books
	// Returns PublisherWithBooksResponse
	GetWithBooks(ctx context.Context, id uuid.UUID) (*PublisherWithBooksResponse, error)

	// ListWithBooks retrieves all publishers with their books (paginated)
	ListWithBooks(ctx context.Context, offset, limit int) ([]*PublisherWithBooksResponse, error)
}
