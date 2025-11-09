package service

import (
	"bookstore-backend/internal/domains/publisher/model"
	"bookstore-backend/internal/domains/publisher/repository"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// publisherService implements publisher.Service
type publisherService struct {
	repo repository.RepositoryInterface
}

// NewPublisherService creates a new publisher service instance
// Dependency injection pattern - receives repository from container
func NewPublisherService(repo repository.RepositoryInterface) ServiceInterface {
	return &publisherService{
		repo: repo,
	}
}

// CreatePublisher creates a new publisher
func (s *publisherService) CreatePublisher(ctx context.Context, req *model.PublisherCreateRequest) (*model.PublisherResponse, error) {
	if req == nil {
		return nil, model.NewInvalidPublisherName("request cannot be nil")
	}

	// Validate request
	if err := model.ValidatePublisherCreate(req); err != nil {
		return nil, err
	}

	// Create publisher model from request
	pub := &model.Publisher{
		Name:        strings.TrimSpace(req.Name),
		Slug:        strings.ToLower(strings.TrimSpace(req.Slug)),
		Website:     strings.TrimSpace(req.Website),
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:       strings.TrimSpace(req.Phone),
		Address:     *req.Address,
		Description: *req.Description,
	}
	// Call repository to persist
	createdPub, err := s.repo.Create(ctx, pub)
	if err != nil {
		return nil, err // Errors dari repo sudah dalam format PublisherError
	}

	return s.modelToResponse(createdPub), nil
}

// GetPublisher retrieves a publisher by ID
func (s *publisherService) GetPublisher(ctx context.Context, id uuid.UUID) (*model.PublisherResponse, error) {
	if id == uuid.Nil {
		return nil, model.NewInvalidPublisherID("id cannot be nil")
	}

	pub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if pub == nil {
		return nil, model.NewPublisherNotFound()
	}

	return s.modelToResponse(pub), nil
}

// GetPublisherBySlug retrieves a publisher by slug
func (s *publisherService) GetPublisherBySlug(ctx context.Context, slug string) (*model.PublisherResponse, error) {
	pub, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher by slug: %w", err)
	}

	if pub == nil {
		return nil, errors.New("publisher not found")
	}

	return s.modelToResponse(pub), nil
}

// ListPublishers retrieves all publishers with pagination
// page: 1-based page number
// pageSize: items per page
func (s *publisherService) ListPublishers(ctx context.Context, page, pageSize int) ([]*model.PublisherResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Get total count
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, model.NewListPublisherError(err)
	}

	// Get publishers for this page
	pubs, err := s.repo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, model.NewListPublisherError(err)
	}

	// Convert to responses
	responses := make([]*model.PublisherResponse, len(pubs))
	for i, pub := range pubs {
		responses[i] = s.modelToResponse(pub)
	}

	return responses, total, nil
}

// UpdatePublisher updates publisher information
func (s *publisherService) UpdatePublisher(ctx context.Context, id uuid.UUID, req *model.PublisherUpdateRequest) (*model.PublisherResponse, error) {
	if id == uuid.Nil {
		return nil, model.NewInvalidPublisherID("id cannot be nil")
	}

	if req == nil {
		return nil, model.NewInvalidPublisherName("request cannot be nil")
	}

	// Validate request
	if err := model.ValidatePublisherUpdate(req); err != nil {
		return nil, err
	}

	// First verify publisher exists
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, model.NewPublisherNotFound()
	}

	// Prepare update model
	updatePub := &model.Publisher{
		Name:    strings.TrimSpace(req.Name),
		Website: strings.TrimSpace(req.Website),
		Email:   strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:   strings.TrimSpace(req.Phone),
	}

	// Use existing values if not provided
	if updatePub.Name == "" {
		updatePub.Name = existing.Name
	}
	if updatePub.Website == "" {
		updatePub.Website = existing.Website
	}
	if updatePub.Email == "" {
		updatePub.Email = existing.Email
	}
	if updatePub.Phone == "" {
		updatePub.Phone = existing.Phone
	}

	updatedPub, err := s.repo.Update(ctx, id, updatePub)
	if err != nil {
		return nil, err
	}

	return s.modelToResponse(updatedPub), nil
}

// DeletePublisher removes a publisher
func (s *publisherService) DeletePublisher(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return model.NewInvalidPublisherID("id cannot be nil")
	}
	// First verify publisher exists
	pub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get publisher: %w", err)
	}

	if pub == nil {
		return model.NewInvalidPublisherID("id cannot be nil")
	}

	// Delete publisher
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// GetPublisherWithBooks retrieves publisher with associated books
func (s *publisherService) GetPublisherWithBooks(ctx context.Context, id uuid.UUID) (*model.PublisherWithBooksResponse, error) {
	pubWithBooks, err := s.repo.GetWithBooks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher with books: %w", err)
	}

	if pubWithBooks == nil {
		return nil, errors.New("publisher not found")
	}

	return pubWithBooks, nil
}

// ListPublishersWithBooks retrieves all publishers with their books
func (s *publisherService) ListPublishersWithBooks(ctx context.Context, page, pageSize int) ([]*model.PublisherWithBooksResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Get total count
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count publishers: %w", err)
	}

	// Get publishers with books for this page
	pubsWithBooks, err := s.repo.ListWithBooks(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list publishers with books: %w", err)
	}

	return pubsWithBooks, total, nil
}

// Helper: Convert Publisher model to PublisherResponse DTO
func (s *publisherService) modelToResponse(pub *model.Publisher) *model.PublisherResponse {
	return &model.PublisherResponse{
		ID:          pub.ID,
		Name:        pub.Name,
		Slug:        pub.Slug,
		Website:     pub.Website,
		Email:       pub.Email,
		Phone:       pub.Phone,
		Address:     &pub.Address,
		Description: &pub.Description,
	}
}
