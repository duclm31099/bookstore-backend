package publisher

import (
	"context"

	"github.com/google/uuid"
)

// Service defines all business logic operations for Publisher domain
type Service interface {
	// CreatePublisher creates a new publisher
	CreatePublisher(ctx context.Context, req *PublisherCreateRequest) (*PublisherResponse, error)

	// GetPublisher retrieves a publisher by ID
	GetPublisher(ctx context.Context, id uuid.UUID) (*PublisherResponse, error)

	// GetPublisherBySlug retrieves a publisher by slug
	GetPublisherBySlug(ctx context.Context, slug string) (*PublisherResponse, error)

	// ListPublishers retrieves all publishers with pagination
	ListPublishers(ctx context.Context, page, pageSize int) ([]*PublisherResponse, int, error)

	// UpdatePublisher updates publisher information
	UpdatePublisher(ctx context.Context, id uuid.UUID, req *PublisherUpdateRequest) (*PublisherResponse, error)

	// DeletePublisher removes a publisher
	DeletePublisher(ctx context.Context, id uuid.UUID) error

	// GetPublisherWithBooks retrieves publisher with associated books
	GetPublisherWithBooks(ctx context.Context, id uuid.UUID) (*PublisherWithBooksResponse, error)

	// ListPublishersWithBooks retrieves all publishers with their books
	ListPublishersWithBooks(ctx context.Context, page, pageSize int) ([]*PublisherWithBooksResponse, int, error)
}
