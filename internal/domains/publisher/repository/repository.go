package repository

import (
	"bookstore-backend/internal/domains/publisher/model"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository defines all data access operations for Publisher domain
type RepositoryInterface interface {
	// Create inserts a new publisher record
	// Returns the created Publisher with generated ID
	Create(ctx context.Context, publisher *model.Publisher) (*model.Publisher, error)

	// GetByID retrieves a publisher by ID
	// Returns nil if not found
	GetByID(ctx context.Context, id uuid.UUID) (*model.Publisher, error)

	// GetBySlug retrieves a publisher by slug
	// Returns nil if not found
	GetBySlug(ctx context.Context, slug string) (*model.Publisher, error)

	// List retrieves all publishers with pagination
	// offset: starting position (0-based)
	// limit: number of records to return
	List(ctx context.Context, offset, limit int) ([]*model.Publisher, error)

	// Count returns total number of publishers
	Count(ctx context.Context) (int, error)

	// Update updates publisher information (except slug & id)
	// Returns updated Publisher
	Update(ctx context.Context, id uuid.UUID, publisher *model.Publisher) (*model.Publisher, error)

	// Delete removes a publisher record
	Delete(ctx context.Context, id uuid.UUID) error
	// NEW: Methods for bulk import
	FindByNameCaseInsensitive(ctx context.Context, name string) (*model.Publisher, error)
	FindBySlugWithTx(ctx context.Context, tx pgx.Tx, slug string) (*model.Publisher, error)
	CreateWithTx(ctx context.Context, tx pgx.Tx, publisher *model.Publisher) error

	// GetWithBooks retrieves publisher with associated books
	// Returns PublisherWithBooksResponse
	GetWithBooks(ctx context.Context, id uuid.UUID) (*model.PublisherWithBooksResponse, error)

	// ListWithBooks retrieves all publishers with their books (paginated)
	ListWithBooks(ctx context.Context, offset, limit int) ([]*model.PublisherWithBooksResponse, error)
}
