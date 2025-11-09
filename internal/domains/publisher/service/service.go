package service

import (
	"bookstore-backend/internal/domains/publisher/model"
	"context"

	"github.com/google/uuid"
)

// Service defines all business logic operations for Publisher domain
type ServiceInterface interface {
	// CreatePublisher creates a new publisher
	CreatePublisher(ctx context.Context, req *model.PublisherCreateRequest) (*model.PublisherResponse, error)

	// GetPublisher retrieves a publisher by ID
	GetPublisher(ctx context.Context, id uuid.UUID) (*model.PublisherResponse, error)

	// GetPublisherBySlug retrieves a publisher by slug
	GetPublisherBySlug(ctx context.Context, slug string) (*model.PublisherResponse, error)

	// ListPublishers retrieves all publishers with pagination
	ListPublishers(ctx context.Context, page, pageSize int) ([]*model.PublisherResponse, int, error)

	// UpdatePublisher updates publisher information
	UpdatePublisher(ctx context.Context, id uuid.UUID, req *model.PublisherUpdateRequest) (*model.PublisherResponse, error)

	// DeletePublisher removes a publisher
	DeletePublisher(ctx context.Context, id uuid.UUID) error

	// GetPublisherWithBooks retrieves publisher with associated books
	GetPublisherWithBooks(ctx context.Context, id uuid.UUID) (*model.PublisherWithBooksResponse, error)

	// ListPublishersWithBooks retrieves all publishers with their books
	ListPublishersWithBooks(ctx context.Context, page, pageSize int) ([]*model.PublisherWithBooksResponse, int, error)
}
