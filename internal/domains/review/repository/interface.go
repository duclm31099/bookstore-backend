package repository

import (
	"context"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/review/model"
)

// =====================================================
// REVIEW REPOSITORY INTERFACE
// =====================================================

type ReviewRepository interface {
	// ========================================
	// CRUD Operations
	// ========================================

	// Create creates new review
	Create(ctx context.Context, review *model.Review) error

	// GetByID gets review by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.Review, error)

	// GetByUserAndBook gets review by user and book (for uniqueness check)
	GetByUserAndBook(ctx context.Context, userID, bookID uuid.UUID) (*model.Review, error)

	// Update updates review
	Update(ctx context.Context, review *model.Review) error

	// Delete deletes review
	Delete(ctx context.Context, id uuid.UUID) error

	// ========================================
	// LIST Operations
	// ========================================

	// ListByBook lists reviews for a book (public, approved only)
	ListByBook(ctx context.Context, bookID uuid.UUID, page, limit int) ([]*model.Review, int, error)

	// ListByUser lists reviews by user
	ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*model.Review, int, error)

	// ListWithFilters lists reviews with filters
	ListWithFilters(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*model.Review, int, error)

	// ========================================
	// STATISTICS
	// ========================================

	// GetBookStatistics gets review statistics for a book
	GetBookStatistics(ctx context.Context, bookID uuid.UUID) (*model.ReviewStatistics, error)

	// GetRatingBreakdown gets rating breakdown (count per rating)
	GetRatingBreakdown(ctx context.Context, bookID uuid.UUID) (map[int]int, error)

	// ========================================
	// ELIGIBILITY & VERIFICATION
	// ========================================

	// CheckEligibility checks if user can review a book
	// Returns: (eligible bool, reason string, error)
	CheckEligibility(ctx context.Context, userID, bookID uuid.UUID) (bool, string, error)

	// HasPurchased checks if user has purchased the book
	HasPurchased(ctx context.Context, userID, bookID uuid.UUID) (bool, uuid.UUID, error)

	// ========================================
	// ADMIN Operations
	// ========================================

	// AdminListReviews lists all reviews with admin filters
	AdminListReviews(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*model.Review, int, error)

	// UpdateModeration updates moderation status
	UpdateModeration(ctx context.Context, id uuid.UUID, isApproved bool, adminNote *string) error

	// UpdateFeatured updates featured status
	UpdateFeatured(ctx context.Context, id uuid.UUID, isFeatured bool) error

	// GetPendingCount gets count of pending reviews (for admin dashboard)
	GetPendingCount(ctx context.Context) (int, error)
}
