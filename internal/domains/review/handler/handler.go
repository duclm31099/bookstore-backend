package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/review/model"
	"bookstore-backend/internal/domains/review/service"
)

// =====================================================
// REVIEW HANDLER
// =====================================================

type ReviewHandler struct {
	reviewService service.ServiceInterface
}

func NewReviewHandler(reviewService service.ServiceInterface) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
	}
}

// =====================================================
// HELPER FUNCTIONS
// =====================================================

// getUserID extracts user ID from JWT claims
func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, model.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// =====================================================
// USER REVIEW ENDPOINTS
// =====================================================

// CreateReview creates new review
// POST /api/v1/reviews
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	// Step 1: Get user ID from JWT
	userID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Bind request body
	var req model.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 3: Validate request
	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 4: Call service
	response, err := h.reviewService.CreateReview(c.Request.Context(), userID, req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return success
	respondSuccess(c, http.StatusCreated, response)
}

// GetReview gets review by ID
// GET /api/v1/reviews/:id
func (h *ReviewHandler) GetReview(c *gin.Context) {
	// Step 1: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 2: Call service
	response, err := h.reviewService.GetReview(c.Request.Context(), reviewID)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 3: Return success
	respondSuccess(c, http.StatusOK, response)
}

// UpdateReview updates user's review
// PUT /api/v1/reviews/:id
func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	// Step 1: Get user ID
	userID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 3: Bind request body
	var req model.UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 4: Validate request
	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 5: Call service
	response, err := h.reviewService.UpdateReview(c.Request.Context(), userID, reviewID, req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 6: Return success
	respondSuccess(c, http.StatusOK, response)
}

// DeleteReview deletes user's review
// DELETE /api/v1/reviews/:id
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	// Step 1: Get user ID
	userID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 3: Call service
	err = h.reviewService.DeleteReview(c.Request.Context(), userID, reviewID)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 4: Return success
	respondSuccess(c, http.StatusOK, gin.H{
		"message": "Review deleted successfully",
	})
}

// ListReviews lists reviews with filters
// GET /api/v1/reviews
func (h *ReviewHandler) ListReviews(c *gin.Context) {
	// Step 1: Bind query parameters
	var req model.ListReviewsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// Step 2: Validate request
	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 3: Call service
	response, err := h.reviewService.ListReviews(c.Request.Context(), req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 4: Return success
	respondSuccess(c, http.StatusOK, response)
}

// ListMyReviews lists reviews by current user
// GET /api/v1/reviews/me
func (h *ReviewHandler) ListMyReviews(c *gin.Context) {
	// Step 1: Get user ID
	userID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Step 3: Call service
	response, err := h.reviewService.ListMyReviews(c.Request.Context(), userID, page, limit)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 4: Return success
	respondSuccess(c, http.StatusOK, response)
}

// GetBookReviews lists reviews for a specific book
// GET /api/v1/books/:book_id/reviews
func (h *ReviewHandler) GetBookReviews(c *gin.Context) {
	// Step 1: Parse book ID
	bookIDStr := c.Param("book_id")
	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid book ID")
		return
	}

	// Step 2: Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Step 3: Build request
	req := model.ListReviewsRequest{
		BookID: &bookID,
		Page:   page,
		Limit:  limit,
	}

	// Step 4: Call service
	response, err := h.reviewService.ListReviews(c.Request.Context(), req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return success
	respondSuccess(c, http.StatusOK, response)
}

// =====================================================
// ADMIN REVIEW ENDPOINTS
// =====================================================

// AdminListReviews lists all reviews with admin filters
// GET /api/v1/admin/reviews
func (h *ReviewHandler) AdminListReviews(c *gin.Context) {
	// Step 1: Verify admin role (done by middleware)

	// Step 2: Bind query parameters
	var req model.AdminListReviewsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// Step 3: Validate request
	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Step 4: Call service
	response, err := h.reviewService.AdminListReviews(c.Request.Context(), req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return success
	respondSuccess(c, http.StatusOK, response)
}

// AdminGetReview gets review detail (admin view)
// GET /api/v1/admin/reviews/:id
func (h *ReviewHandler) AdminGetReview(c *gin.Context) {
	// Step 1: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 2: Call service
	response, err := h.reviewService.AdminGetReview(c.Request.Context(), reviewID)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 3: Return success
	respondSuccess(c, http.StatusOK, response)
}

// AdminModerateReview moderates review (approve/hide)
// PATCH /api/v1/admin/reviews/:id/moderate
func (h *ReviewHandler) AdminModerateReview(c *gin.Context) {
	// Step 1: Get admin ID
	adminID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 3: Bind request body
	var req model.ModerateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 4: Call service
	err = h.reviewService.AdminModerateReview(c.Request.Context(), adminID, reviewID, req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return success
	respondSuccess(c, http.StatusOK, gin.H{
		"message": "Review moderated successfully",
	})
}

// AdminFeatureReview features/unfeatures review
// PATCH /api/v1/admin/reviews/:id/feature
func (h *ReviewHandler) AdminFeatureReview(c *gin.Context) {
	// Step 1: Get admin ID
	adminID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 3: Bind request body
	var req model.FeatureReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Step 4: Call service
	err = h.reviewService.AdminFeatureReview(c.Request.Context(), adminID, reviewID, req)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 5: Return success
	respondSuccess(c, http.StatusOK, gin.H{
		"message": "Review featured status updated successfully",
	})
}

// AdminGetStatistics gets admin dashboard statistics
// GET /api/v1/admin/reviews/statistics
func (h *ReviewHandler) AdminGetStatistics(c *gin.Context) {
	// Step 1: Call service
	statistics, err := h.reviewService.AdminGetStatistics(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Step 2: Return success
	respondSuccess(c, http.StatusOK, statistics)
}

// AdminDeleteReview deletes review (admin only)
// DELETE /api/v1/admin/reviews/:id
func (h *ReviewHandler) AdminDeleteReview(c *gin.Context) {
	// Step 1: Get admin ID (for audit log)
	adminID, err := getUserID(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "AUTH_ERROR", "Unauthorized")
		return
	}

	// Step 2: Parse review ID
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	// Step 3: Delete review (admin can delete any review)
	// Use service method that bypasses user ownership check
	err = h.reviewService.DeleteReview(c.Request.Context(), adminID, reviewID)
	if err != nil {
		statusCode, errCode := mapReviewError(err)
		respondError(c, statusCode, errCode, err.Error())
		return
	}

	// Step 4: Return success
	respondSuccess(c, http.StatusOK, gin.H{
		"message": "Review deleted successfully",
	})
}

// respondSuccess sends success response
func respondSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
	})
}

// respondError sends error response
func respondError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// mapReviewError maps review error to HTTP status code
func mapReviewError(err error) (int, string) {
	if reviewErr, ok := err.(*model.ReviewError); ok {
		switch reviewErr.Code {
		case model.ErrCodeReviewNotFound:
			return http.StatusNotFound, reviewErr.Code
		case model.ErrCodeAlreadyReviewed:
			return http.StatusConflict, reviewErr.Code
		case model.ErrCodeNotEligible:
			return http.StatusForbidden, reviewErr.Code
		case model.ErrCodeCannotEdit, model.ErrCodeCannotDelete:
			return http.StatusForbidden, reviewErr.Code
		case model.ErrCodeUnauthorized:
			return http.StatusUnauthorized, reviewErr.Code
		case model.ErrCodeInvalidRating, model.ErrCodeContentTooShort, model.ErrCodeContentTooLong:
			return http.StatusBadRequest, reviewErr.Code
		default:
			return http.StatusInternalServerError, "INTERNAL_ERROR"
		}
	}
	return http.StatusInternalServerError, "INTERNAL_ERROR"
}
