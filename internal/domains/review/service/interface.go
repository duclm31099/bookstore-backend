package service

import (
	"context"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/review/model"
)

// =====================================================
// REVIEW SERVICE INTERFACE
// =====================================================

type ServiceInterface interface {
	// ========================================
	// USER OPERATIONS
	// ========================================

	// CreateReview creates new review
	CreateReview(ctx context.Context, userID uuid.UUID, req model.CreateReviewRequest) (*model.ReviewResponse, error)

	// GetReview gets review by ID
	GetReview(ctx context.Context, id uuid.UUID) (*model.ReviewResponse, error)

	// UpdateReview updates user's review
	UpdateReview(ctx context.Context, userID, reviewID uuid.UUID, req model.UpdateReviewRequest) (*model.ReviewResponse, error)

	// DeleteReview deletes user's review
	DeleteReview(ctx context.Context, userID, reviewID uuid.UUID) error

	// ListReviews lists reviews with filters
	ListReviews(ctx context.Context, req model.ListReviewsRequest) (*model.ListReviewsResponse, error)

	// ListMyReviews lists reviews by current user
	ListMyReviews(ctx context.Context, userID uuid.UUID, page, limit int) (*model.ListReviewsResponse, error)

	// ========================================
	// ADMIN OPERATIONS
	// ========================================

	// AdminListReviews lists all reviews with admin filters
	AdminListReviews(ctx context.Context, req model.AdminListReviewsRequest) (*model.ListReviewsResponse, error)

	// AdminGetReview gets review detail (admin view)
	AdminGetReview(ctx context.Context, id uuid.UUID) (*model.AdminReviewResponse, error)

	// AdminModerateReview moderates review (approve/hide)
	AdminModerateReview(ctx context.Context, adminID, reviewID uuid.UUID, req model.ModerateReviewRequest) error

	// AdminFeatureReview features/unfeatures review
	AdminFeatureReview(ctx context.Context, adminID, reviewID uuid.UUID, req model.FeatureReviewRequest) error

	// AdminGetStatistics gets admin dashboard statistics
	AdminGetStatistics(ctx context.Context) (map[string]interface{}, error)
}
