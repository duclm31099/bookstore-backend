// internal/domains/author/service/author_service.go
package service

import (
	"bookstore-backend/internal/domains/author/model"
	"bookstore-backend/internal/domains/author/repository"
	"bookstore-backend/internal/shared/utils"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// authorService implements author.Service interface
type authorService struct {
	repo repository.RepositoryInterface // Repository dependency (injected)
}

// NewAuthorService creates a new author service instance
// Dependency Injection pattern:
// - Service depends on Repository abstraction (interface), not concrete type
// - Allows easy testing (mock repository)
// - Follows Dependency Inversion Principle (SOLID)
func NewAuthorService(repo repository.RepositoryInterface) ServiceInterface {
	return &authorService{
		repo: repo,
	}
}

// GetWithBookCount retrieves author with book count
func (s *authorService) GetWithBookCount(ctx context.Context, id uuid.UUID) (*model.Author, int, error) {
	// Validate UUID
	if id == uuid.Nil {
		return nil, 0, model.ErrAuthorNotFound
	}

	// Get author
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	// Get book count
	bookCount, err := s.repo.GetBookCount(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	return a, bookCount, nil
}

func (s *authorService) GetByID(ctx context.Context, id uuid.UUID) (*model.Author, error) {
	// Validate UUID
	if id == uuid.Nil {
		return nil, model.ErrAuthorNotFound
	}

	// Repository handles cache + DB
	return s.repo.GetByID(ctx, id)
}

// GetBySlug - Simple implementation
func (s *authorService) GetBySlug(ctx context.Context, slug string) (*model.Author, error) {
	// Validate slug
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return nil, model.ErrAuthorNotFound
	}

	// Repository handles cache + DB
	return s.repo.GetBySlug(ctx, slug)
}

func (s *authorService) Create(ctx context.Context, req *model.CreateAuthorRequest) (*model.Author, error) {

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, model.ErrInvalidName
	}
	if len(name) < model.MinNameLength {
		return nil, fmt.Errorf("name too short minimum %d length", model.MinNameLength)
	}
	if len(name) > model.MaxNameLength {
		return nil, fmt.Errorf("name too long maximum %d length", model.MaxNameLength)
	}

	if len(*req.Bio) > model.MaxBioLength {
		return nil, fmt.Errorf("bio too long maximum %d length", model.MaxBioLength)
	}

	baseSlug := utils.GenerateSlug(req.Name)
	exists, err := s.repo.ExistsBySlug(ctx, baseSlug)
	if err != nil || exists {
		return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	newAuthor := &model.Author{
		Name:     name,
		Slug:     baseSlug,
		Bio:      req.Bio,
		PhotoURL: req.PhotoURL,
		Version:  0, // Initial version
	}
	createdAuthor, err := s.repo.Create(ctx, newAuthor)
	if err != nil {
		return nil, fmt.Errorf("failed to create author: %w", err)
	}
	return createdAuthor, nil
}
func (s *authorService) GetAll(ctx context.Context, filter model.AuthorFilter) ([]model.Author, int64, error) {
	// ═══════════════════════════════════════════════════════════
	// CRITICAL: VALIDATE ỨƠ SANITIZE PAGINATION PARAMETERS
	// ═══════════════════════════════════════════════════════════

	// Default values
	if filter.Limit <= 0 {
		filter.Limit = 20 // Default page size
	}

	// Prevent abuse: Max limit
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Negative offset protection
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// ═══════════════════════════════════════════════════════════
	// CRITICAL: VALIDATE SORT PARAMETERS (PREVENT SQL INJECTION)
	// ═══════════════════════════════════════════════════════════

	// Whitelist allowed sort columns
	allowedSortColumns := map[string]bool{
		"name":       true,
		"created_at": true,
		"updated_at": true,
	}

	if filter.SortBy == "" {
		filter.SortBy = "created_at" // Default sort
	}

	// CRITICAL: Reject non-whitelisted columns
	if !allowedSortColumns[filter.SortBy] {
		return nil, 0, fmt.Errorf("invalid sort column: %s", filter.SortBy)
	}

	// Validate sort order
	filter.Order = strings.ToUpper(filter.Order)
	if filter.Order != "ASC" && filter.Order != "DESC" {
		filter.Order = "DESC" // Default order
	}

	// ═══════════════════════════════════════════════════════════
	// QUERY DATABASE
	// ═══════════════════════════════════════════════════════════

	authors, total, err := s.repo.GetAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return authors, total, nil
}

// Update implements author.Service.Update with conflict detection
func (s *authorService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateAuthorRequest) (*model.Author, error) {
	// ═══════════════════════════════════════════════════════════
	// STEP 1: FETCH CURRENT AUTHOR
	// ═══════════════════════════════════════════════════════════

	// Get current state from database
	currentAuthor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err // ErrAuthorNotFound nếu không tồn tại
	}

	// ═══════════════════════════════════════════════════════════
	// STEP 2: VERSION CONFLICT DETECTION (OPTIMISTIC LOCKING)
	// ═══════════════════════════════════════════════════════════

	// Client MUST send current version they're updating from
	// If versions don't match, another user has modified the author
	if req.Version != currentAuthor.Version {
		return nil, model.ErrVersionMismatch
	}

	// ═══════════════════════════════════════════════════════════
	// STEP 3: APPLY PARTIAL UPDATES
	// ═══════════════════════════════════════════════════════════

	// Only update fields that are non-nil (PATCH behavior)
	updatedAuthor := *currentAuthor // Copy current state

	if req.Name != nil {
		// Validate new name
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, model.ErrInvalidName
		}
		if len(name) < model.MinNameLength || len(name) > model.MaxNameLength {
			return nil, fmt.Errorf("invalid name length")
		}

		// If name changes, regenerate slug
		if name != currentAuthor.Name {
			newSlug := utils.GenerateSlug(name)

			// Check if new slug already exists (excluding current author)
			if newSlug != currentAuthor.Slug {
				exists, err := s.repo.ExistsBySlug(ctx, newSlug)
				if err != nil {
					return nil, err
				}
				if exists {
					return nil, model.ErrDuplicateSlug
				}
				updatedAuthor.Slug = newSlug
			}

			updatedAuthor.Name = name
		}
	}

	if req.Bio != nil {
		if len(*req.Bio) > model.MaxBioLength {
			return nil, fmt.Errorf("bio too long")
		}
		updatedAuthor.Bio = req.Bio
	}

	// ═══════════════════════════════════════════════════════════
	// STEP 4: PERSIST WITH VERSION CHECK
	// ═══════════════════════════════════════════════════════════

	// Repository will:
	// 1. Execute UPDATE with WHERE version = currentVersion
	// 2. If no rows affected, another update happened → ErrVersionMismatch
	// 3. Increment version automatically
	result, err := s.repo.Update(ctx, &updatedAuthor, currentAuthor.Version)
	if err != nil {
		return nil, err
	}

	return result, nil
}
func (s *authorService) Delete(ctx context.Context, id uuid.UUID) error {
	// ═══════════════════════════════════════════════════════════
	// CRITICAL: CHECK REFERENTIAL INTEGRITY TRƯỚC KHI XÓA
	// ═══════════════════════════════════════════════════════════

	// Check if author has linked books
	bookCount, err := s.repo.GetBookCount(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check book count: %w", err)
	}

	// Business rule: CANNOT delete author with books
	if bookCount > 0 {
		return fmt.Errorf("%w: author has %d linked books", model.ErrAuthorHasBooks, bookCount)
	}

	// Safe to delete
	return s.repo.Delete(ctx, id)
}
func (s *authorService) Search(ctx context.Context, query string, filter model.AuthorFilter) ([]model.Author, int64, error) {
	// ═══════════════════════════════════════════════════════════
	// CRITICAL: SANITIZE SEARCH QUERY
	// ═══════════════════════════════════════════════════════════

	// Trim whitespace
	query = strings.TrimSpace(query)

	// Empty query protection
	if query == "" {
		return []model.Author{}, 0, nil // Return empty, not error
	}

	// Minimum length (prevent short query performance issues)
	if len(query) < 2 {
		return nil, 0, fmt.Errorf("search query too short: minimum 2 characters")
	}

	// Maximum length (prevent abuse)
	if len(query) > 100 {
		query = query[:100] // Truncate
	}

	// ═══════════════════════════════════════════════════════════
	// CRITICAL: ESCAPE SPECIAL CHARACTERS
	// ═══════════════════════════════════════════════════════════

	// PostgreSQL ILIKE wildcards: % and _
	// Escape them to prevent unintended wildcards
	query = escapeWildcards(query)

	// Validate pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	return s.repo.Search(ctx, query, filter)
}

// escapeWildcards prevents user from injecting SQL wildcards
func escapeWildcards(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // Escape backslash first
	s = strings.ReplaceAll(s, "%", "\\%")   // Escape %
	s = strings.ReplaceAll(s, "_", "\\_")   // Escape _
	return s
}
func (s *authorService) BulkDelete(ctx context.Context, req model.BulkDeleteRequest) (int, []model.BulkError, error) {
	// ═══════════════════════════════════════════════════════════
	// CRITICAL: VALIDATE INPUT SIZE
	// ═══════════════════════════════════════════════════════════

	if len(req.IDs) == 0 {
		return 0, nil, nil
	}

	// Prevent abuse: Max batch size
	const maxBatchSize = 100
	if len(req.IDs) > maxBatchSize {
		return 0, nil, fmt.Errorf("batch size exceeds maximum (%d)", maxBatchSize)
	}

	// ═══════════════════════════════════════════════════════════
	// CRITICAL: VALIDATE EACH AUTHOR INDIVIDUALLY
	// ═══════════════════════════════════════════════════════════

	var validIDs []uuid.UUID
	var bulkErrors []model.BulkError

	// Pre-validation: Check constraints for each author
	for _, id := range req.IDs {
		// Check if author has books
		bookCount, err := s.repo.GetBookCount(ctx, id)
		if err != nil {
			bulkErrors = append(bulkErrors, model.BulkError{
				ID:      id,
				Message: fmt.Sprintf("failed to check books: %v", err),
			})
			continue
		}

		if bookCount > 0 {
			bulkErrors = append(bulkErrors, model.BulkError{
				ID:      id,
				Message: fmt.Sprintf("author has %d linked books", bookCount),
			})
			continue
		}

		validIDs = append(validIDs, id)
	}

	// If no valid IDs, return early
	if len(validIDs) == 0 {
		return 0, bulkErrors, nil
	}

	// ═══════════════════════════════════════════════════════════
	// DELEGATE TO REPOSITORY (WITH TRANSACTION)
	// ═══════════════════════════════════════════════════════════

	successCount, repoErrors, err := s.repo.BulkDelete(ctx, validIDs)
	if err != nil {
		return 0, nil, fmt.Errorf("bulk delete failed: %w", err)
	}

	// Merge errors from validation and deletion
	bulkErrors = append(bulkErrors, repoErrors...)

	return successCount, bulkErrors, nil
}
