package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/review/model"
	"bookstore-backend/internal/domains/review/repository"
)

// =====================================================
// SERVICE IMPLEMENTATION
// =====================================================

type reviewService struct {
	reviewRepo repository.ReviewRepository
	// Add other dependencies if needed (e.g., book service, user service)
}

func NewReviewService(
	reviewRepo repository.ReviewRepository,
) ServiceInterface {
	return &reviewService{
		reviewRepo: reviewRepo,
	}
}

// =====================================================
// CREATE REVIEW
// =====================================================

func (s *reviewService) CreateReview(
	ctx context.Context,
	userID uuid.UUID,
	req model.CreateReviewRequest,
) (*model.ReviewResponse, error) {
	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Check eligibility
	eligible, reason, err := s.reviewRepo.CheckEligibility(ctx, userID, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check eligibility: %w", err)
	}
	if !eligible {
		return nil, model.NewNotEligibleError(reason)
	}

	// Step 3: Verify user has purchased from this order
	// (Additional check: ensure order_id belongs to user and contains book_id)
	hasPurchased, verifiedOrderID, err := s.reviewRepo.HasPurchased(ctx, userID, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify purchase: %w", err)
	}
	if !hasPurchased {
		return nil, model.NewNotEligibleError("No purchase found for this book")
	}

	// Step 4: Create review entity
	review := &model.Review{
		ID:                 uuid.New(),
		UserID:             userID,
		BookID:             req.BookID,
		OrderID:            verifiedOrderID, // Use verified order ID
		Rating:             req.Rating,
		Title:              req.Title,
		Content:            req.Content,
		Images:             req.Images,
		IsVerifiedPurchase: true,  // Always true since we verified
		IsApproved:         true,  // Auto-approve
		IsFeatured:         false, // Admin sets this
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Step 5: Save to database
	if err := s.reviewRepo.Create(ctx, review); err != nil {
		if err == model.ErrAlreadyReviewed {
			return nil, model.NewAlreadyReviewedError()
		}
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	// Step 6: Build response
	response := &model.ReviewResponse{
		ID:     review.ID,
		BookID: review.BookID,
		UserInfo: model.UserInfo{
			ID:   review.UserID,
			Name: "User", // TODO: Get from user service
		},
		Rating:             review.Rating,
		Title:              review.Title,
		Content:            review.Content,
		Images:             review.Images,
		IsVerifiedPurchase: review.IsVerifiedPurchase,
		IsFeatured:         review.IsFeatured,
		CreatedAt:          review.CreatedAt,
		UpdatedAt:          review.UpdatedAt,
	}

	return response, nil
}

// =====================================================
// GET REVIEW
// =====================================================

func (s *reviewService) GetReview(
	ctx context.Context,
	id uuid.UUID,
) (*model.ReviewResponse, error) {
	// Get review
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		if err == model.ErrReviewNotFound {
			return nil, model.NewReviewNotFoundError()
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Only return approved reviews to public
	if !review.IsApproved {
		return nil, model.NewReviewNotFoundError()
	}

	// Build response
	response := &model.ReviewResponse{
		ID:     review.ID,
		BookID: review.BookID,
		UserInfo: model.UserInfo{
			ID:   review.UserID,
			Name: "User", // TODO: Get from user service
		},
		Rating:             review.Rating,
		Title:              review.Title,
		Content:            review.Content,
		Images:             review.Images,
		IsVerifiedPurchase: review.IsVerifiedPurchase,
		IsFeatured:         review.IsFeatured,
		CreatedAt:          review.CreatedAt,
		UpdatedAt:          review.UpdatedAt,
	}

	return response, nil
}

// =====================================================
// UPDATE REVIEW
// =====================================================

func (s *reviewService) UpdateReview(
	ctx context.Context,
	userID, reviewID uuid.UUID,
	req model.UpdateReviewRequest,
) (*model.ReviewResponse, error) {
	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Get existing review
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		if err == model.ErrReviewNotFound {
			return nil, model.NewReviewNotFoundError()
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Step 3: Verify ownership
	if review.UserID != userID {
		return nil, model.NewUnauthorizedError("You can only edit your own reviews")
	}

	// Step 4: Update fields (only if provided)
	if req.Rating != nil {
		review.Rating = *req.Rating
	}
	if req.Title != nil {
		review.Title = req.Title
	}
	if req.Content != nil {
		review.Content = *req.Content
	}
	if req.Images != nil {
		review.Images = req.Images
	}

	review.UpdatedAt = time.Now()

	// Step 5: Save changes
	if err := s.reviewRepo.Update(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	// Step 6: Build response
	response := &model.ReviewResponse{
		ID:     review.ID,
		BookID: review.BookID,
		UserInfo: model.UserInfo{
			ID:   review.UserID,
			Name: "User",
		},
		Rating:             review.Rating,
		Title:              review.Title,
		Content:            review.Content,
		Images:             review.Images,
		IsVerifiedPurchase: review.IsVerifiedPurchase,
		IsFeatured:         review.IsFeatured,
		CreatedAt:          review.CreatedAt,
		UpdatedAt:          review.UpdatedAt,
	}

	return response, nil
}

// =====================================================
// DELETE REVIEW
// =====================================================

func (s *reviewService) DeleteReview(
	ctx context.Context,
	userID, reviewID uuid.UUID,
) error {
	// Step 1: Get review
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		if err == model.ErrReviewNotFound {
			return model.NewReviewNotFoundError()
		}
		return fmt.Errorf("failed to get review: %w", err)
	}

	// Step 2: Verify ownership
	if review.UserID != userID {
		return model.NewUnauthorizedError("You can only delete your own reviews")
	}

	// Step 3: Delete review
	if err := s.reviewRepo.Delete(ctx, reviewID); err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	return nil
}

// =====================================================
// LIST REVIEWS
// =====================================================

func (s *reviewService) ListReviews(
	ctx context.Context,
	req model.ListReviewsRequest,
) (*model.ListReviewsResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var reviews []*model.Review
	var total int
	var err error

	// If book_id provided, list by book
	if req.BookID != nil {
		reviews, total, err = s.reviewRepo.ListByBook(ctx, *req.BookID, req.Page, req.Limit)
	} else if req.UserID != nil {
		// If user_id provided, list by user
		reviews, total, err = s.reviewRepo.ListByUser(ctx, *req.UserID, req.Page, req.Limit)
	} else {
		// Otherwise, list with filters
		filters := make(map[string]interface{})
		if req.Rating != nil {
			filters["rating"] = *req.Rating
		}
		reviews, total, err = s.reviewRepo.ListWithFilters(ctx, filters, req.Page, req.Limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	// Build response
	reviewResponses := make([]model.ReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, model.ReviewResponse{
			ID:     review.ID,
			BookID: review.BookID,
			UserInfo: model.UserInfo{
				ID:   review.UserID,
				Name: "User", // TODO: Batch get from user service
			},
			Rating:             review.Rating,
			Title:              review.Title,
			Content:            review.Content,
			Images:             review.Images,
			IsVerifiedPurchase: review.IsVerifiedPurchase,
			IsFeatured:         review.IsFeatured,
			CreatedAt:          review.CreatedAt,
			UpdatedAt:          review.UpdatedAt,
		})
	}

	// Get statistics (if book_id provided)
	var statistics model.ReviewStatistics
	if req.BookID != nil {
		stats, err := s.reviewRepo.GetBookStatistics(ctx, *req.BookID)
		if err != nil {
			return nil, fmt.Errorf("failed to get statistics: %w", err)
		}
		statistics = *stats
	}

	// Build pagination
	totalPages := (total + req.Limit - 1) / req.Limit
	pagination := model.PaginationMeta{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return &model.ListReviewsResponse{
		Reviews:    reviewResponses,
		Statistics: statistics,
		Pagination: pagination,
	}, nil
}

// =====================================================
// LIST MY REVIEWS
// =====================================================

func (s *reviewService) ListMyReviews(
	ctx context.Context,
	userID uuid.UUID,
	page, limit int,
) (*model.ListReviewsResponse, error) {
	// Get reviews by user
	reviews, total, err := s.reviewRepo.ListByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	// Build response
	reviewResponses := make([]model.ReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, model.ReviewResponse{
			ID:     review.ID,
			BookID: review.BookID,
			UserInfo: model.UserInfo{
				ID:   review.UserID,
				Name: "User",
			},
			Rating:             review.Rating,
			Title:              review.Title,
			Content:            review.Content,
			Images:             review.Images,
			IsVerifiedPurchase: review.IsVerifiedPurchase,
			IsFeatured:         review.IsFeatured,
			CreatedAt:          review.CreatedAt,
			UpdatedAt:          review.UpdatedAt,
		})
	}

	// Build pagination
	totalPages := (total + limit - 1) / limit
	pagination := model.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	return &model.ListReviewsResponse{
		Reviews:    reviewResponses,
		Pagination: pagination,
	}, nil
}

// =====================================================
// ADMIN: LIST REVIEWS
// =====================================================

func (s *reviewService) AdminListReviews(
	ctx context.Context,
	req model.AdminListReviewsRequest,
) (*model.ListReviewsResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build filters
	filters := make(map[string]interface{})
	if req.BookID != nil {
		filters["book_id"] = *req.BookID
	}
	if req.UserID != nil {
		filters["user_id"] = *req.UserID
	}
	if req.Rating != nil {
		filters["rating"] = *req.Rating
	}
	if req.IsApproved != nil {
		filters["is_approved"] = *req.IsApproved
	}
	if req.IsFeatured != nil {
		filters["is_featured"] = *req.IsFeatured
	}
	if req.Search != nil {
		filters["search"] = *req.Search
	}

	// Get reviews
	reviews, total, err := s.reviewRepo.AdminListReviews(ctx, filters, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	// Build response
	reviewResponses := make([]model.ReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, model.ReviewResponse{
			ID:     review.ID,
			BookID: review.BookID,
			UserInfo: model.UserInfo{
				ID:   review.UserID,
				Name: "User", // TODO: Get from user service
			},
			Rating:             review.Rating,
			Title:              review.Title,
			Content:            review.Content,
			Images:             review.Images,
			IsVerifiedPurchase: review.IsVerifiedPurchase,
			IsFeatured:         review.IsFeatured,
			CreatedAt:          review.CreatedAt,
			UpdatedAt:          review.UpdatedAt,
		})
	}

	// Build pagination
	totalPages := (total + req.Limit - 1) / req.Limit
	pagination := model.PaginationMeta{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return &model.ListReviewsResponse{
		Reviews:    reviewResponses,
		Pagination: pagination,
	}, nil
}

// =====================================================
// ADMIN: GET REVIEW
// =====================================================

func (s *reviewService) AdminGetReview(
	ctx context.Context,
	id uuid.UUID,
) (*model.AdminReviewResponse, error) {
	// Get review (admin can see all, including unapproved)
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		if err == model.ErrReviewNotFound {
			return nil, model.NewReviewNotFoundError()
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Build admin response (includes admin_note and moderation status)
	response := &model.AdminReviewResponse{
		ReviewResponse: model.ReviewResponse{
			ID:     review.ID,
			BookID: review.BookID,
			UserInfo: model.UserInfo{
				ID:   review.UserID,
				Name: "User",
			},
			Rating:             review.Rating,
			Title:              review.Title,
			Content:            review.Content,
			Images:             review.Images,
			IsVerifiedPurchase: review.IsVerifiedPurchase,
			IsFeatured:         review.IsFeatured,
			CreatedAt:          review.CreatedAt,
			UpdatedAt:          review.UpdatedAt,
		},
		OrderID:    review.OrderID,
		AdminNote:  review.AdminNote,
		IsApproved: review.IsApproved,
	}

	return response, nil
}

// =====================================================
// ADMIN: MODERATE REVIEW
// =====================================================

func (s *reviewService) AdminModerateReview(
	ctx context.Context,
	adminID, reviewID uuid.UUID,
	req model.ModerateReviewRequest,
) error {
	// Step 1: Get review
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		if err == model.ErrReviewNotFound {
			return model.NewReviewNotFoundError()
		}
		return fmt.Errorf("failed to get review: %w", err)
	}

	// Step 2: Update moderation status
	if err := s.reviewRepo.UpdateModeration(ctx, reviewID, req.IsApproved, req.AdminNote); err != nil {
		return fmt.Errorf("failed to update moderation: %w", err)
	}

	// Step 3: Create audit log (TODO)
	// Log admin action for compliance

	// Step 4: If review was hidden, notify user (optional)
	if !req.IsApproved && review.IsApproved {
		// TODO: Send notification to user
		// "Your review has been hidden by admin: {reason}"
	}

	return nil
}

// =====================================================
// ADMIN: FEATURE REVIEW
// =====================================================

func (s *reviewService) AdminFeatureReview(
	ctx context.Context,
	adminID, reviewID uuid.UUID,
	req model.FeatureReviewRequest,
) error {
	// Step 1: Get review
	_, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		if err == model.ErrReviewNotFound {
			return model.NewReviewNotFoundError()
		}
		return fmt.Errorf("failed to get review: %w", err)
	}

	// Step 2: Update featured status
	if err := s.reviewRepo.UpdateFeatured(ctx, reviewID, req.IsFeatured); err != nil {
		return fmt.Errorf("failed to update featured: %w", err)
	}

	// Step 3: Create audit log (TODO)

	return nil
}

// =====================================================
// ADMIN: GET STATISTICS
// =====================================================

func (s *reviewService) AdminGetStatistics(ctx context.Context) (map[string]interface{}, error) {
	// Get pending reviews count
	pendingCount, err := s.reviewRepo.GetPendingCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	// TODO: Add more statistics:
	// - Total reviews
	// - Reviews today
	// - Average rating across all books
	// - Most reviewed books

	statistics := map[string]interface{}{
		"pending_reviews": pendingCount,
		// Add more stats here
	}

	return statistics, nil
}
