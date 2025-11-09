package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =====================================================
// USER REQUEST DTOs
// =====================================================

// CreateReviewRequest request to create review
type CreateReviewRequest struct {
	BookID  uuid.UUID `json:"book_id" binding:"required"`
	OrderID uuid.UUID `json:"order_id" binding:"required"`
	Rating  int       `json:"rating" binding:"required,min=1,max=5"`
	Title   *string   `json:"title"`
	Content string    `json:"content" binding:"required,min=10,max=2000"`
	Images  []string  `json:"images"`
}

func (r *CreateReviewRequest) Validate() error {
	if r.Rating < 1 || r.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	if len(r.Content) < 10 {
		return fmt.Errorf("content must be at least 10 characters")
	}
	if len(r.Content) > 2000 {
		return fmt.Errorf("content must not exceed 2000 characters")
	}
	if len(r.Images) > 5 {
		return fmt.Errorf("maximum 5 images allowed")
	}
	return nil
}

// UpdateReviewRequest request to update review
type UpdateReviewRequest struct {
	Rating  *int     `json:"rating"`
	Title   *string  `json:"title"`
	Content *string  `json:"content"`
	Images  []string `json:"images"`
}

func (r *UpdateReviewRequest) Validate() error {
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 5) {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	if r.Content != nil {
		if len(*r.Content) < 10 {
			return fmt.Errorf("content must be at least 10 characters")
		}
		if len(*r.Content) > 2000 {
			return fmt.Errorf("content must not exceed 2000 characters")
		}
	}
	if len(r.Images) > 5 {
		return fmt.Errorf("maximum 5 images allowed")
	}
	return nil
}

// ListReviewsRequest request to list reviews
type ListReviewsRequest struct {
	BookID *uuid.UUID `form:"book_id"`
	UserID *uuid.UUID `form:"user_id"`
	Rating *int       `form:"rating"`
	Page   int        `form:"page" binding:"min=1"`
	Limit  int        `form:"limit" binding:"min=1,max=100"`
}

func (r *ListReviewsRequest) Validate() error {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.Limit < 1 || r.Limit > 100 {
		r.Limit = 20
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 5) {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	return nil
}

// =====================================================
// ADMIN REQUEST DTOs
// =====================================================

// AdminListReviewsRequest admin request to list reviews
type AdminListReviewsRequest struct {
	BookID     *uuid.UUID `form:"book_id"`
	UserID     *uuid.UUID `form:"user_id"`
	Rating     *int       `form:"rating"`
	IsApproved *bool      `form:"is_approved"`
	IsFeatured *bool      `form:"is_featured"`
	Search     *string    `form:"search"`
	Page       int        `form:"page"`
	Limit      int        `form:"limit"`
}

func (r *AdminListReviewsRequest) Validate() error {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.Limit < 1 || r.Limit > 100 {
		r.Limit = 50
	}
	return nil
}

// ModerateReviewRequest admin request to moderate review
type ModerateReviewRequest struct {
	IsApproved bool    `json:"is_approved"`
	AdminNote  *string `json:"admin_note"`
}

// FeatureReviewRequest admin request to feature review
type FeatureReviewRequest struct {
	IsFeatured bool `json:"is_featured"`
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// ReviewResponse response for review detail
type ReviewResponse struct {
	ID       uuid.UUID `json:"id"`
	BookID   uuid.UUID `json:"book_id"`
	BookInfo *BookInfo `json:"book_info,omitempty"`
	UserInfo UserInfo  `json:"user_info"`

	Rating  int      `json:"rating"`
	Title   *string  `json:"title"`
	Content string   `json:"content"`
	Images  []string `json:"images"`

	IsVerifiedPurchase bool `json:"is_verified_purchase"`
	IsFeatured         bool `json:"is_featured"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserInfo user information in review
type UserInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// Don't expose email for privacy
}

// BookInfo book information in review
type BookInfo struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Slug  string    `json:"slug"`
}

// ListReviewsResponse response for list reviews
type ListReviewsResponse struct {
	Reviews    []ReviewResponse `json:"reviews"`
	Statistics ReviewStatistics `json:"statistics"`
	Pagination PaginationMeta   `json:"pagination"`
}

// ReviewStatistics review statistics
type ReviewStatistics struct {
	TotalReviews    int         `json:"total_reviews"`
	AverageRating   float64     `json:"average_rating"`
	RatingBreakdown map[int]int `json:"rating_breakdown"` // {5: 100, 4: 50, ...}
}

// PaginationMeta pagination metadata
type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// =====================================================
// ADMIN RESPONSE DTOs
// =====================================================

// AdminReviewResponse admin response with full details
type AdminReviewResponse struct {
	ReviewResponse
	OrderID    uuid.UUID `json:"order_id"`
	AdminNote  *string   `json:"admin_note"`
	IsApproved bool      `json:"is_approved"`
}
