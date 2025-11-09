package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"bookstore-backend/internal/domains/review/model"
)

// =====================================================
// POSTGRES REPOSITORY IMPLEMENTATION
// =====================================================

type postgresReviewRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresReviewRepository(pool *pgxpool.Pool) ReviewRepository {
	return &postgresReviewRepository{pool: pool}
}

// =====================================================
// CREATE
// =====================================================

func (r *postgresReviewRepository) Create(ctx context.Context, review *model.Review) error {
	query := `
		INSERT INTO reviews (
			id, user_id, book_id, order_id,
			rating, title, content, images,
			is_verified_purchase, is_approved, is_featured,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.pool.Exec(ctx, query,
		review.ID,
		review.UserID,
		review.BookID,
		review.OrderID,
		review.Rating,
		review.Title,
		review.Content,
		pq.Array(review.Images),
		review.IsVerifiedPurchase,
		review.IsApproved, // Auto-approve = true by default
		review.IsFeatured,
		review.CreatedAt,
		review.UpdatedAt,
	)

	if err != nil {
		// Check unique constraint violation
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return model.ErrAlreadyReviewed
		}
		return fmt.Errorf("failed to create review: %w", err)
	}

	return nil
}

// =====================================================
// GET BY ID
// =====================================================

func (r *postgresReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Review, error) {
	query := `
		SELECT 
			id, user_id, book_id, order_id,
			rating, title, content, images,
			is_verified_purchase, is_approved, is_featured, admin_note,
			created_at, updated_at
		FROM reviews
		WHERE id = $1
	`

	review := &model.Review{}
	var images []string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&review.ID,
		&review.UserID,
		&review.BookID,
		&review.OrderID,
		&review.Rating,
		&review.Title,
		&review.Content,
		pq.Array(&images),
		&review.IsVerifiedPurchase,
		&review.IsApproved,
		&review.IsFeatured,
		&review.AdminNote,
		&review.CreatedAt,
		&review.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrReviewNotFound
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	review.Images = images
	return review, nil
}

// =====================================================
// GET BY USER AND BOOK
// =====================================================

func (r *postgresReviewRepository) GetByUserAndBook(
	ctx context.Context,
	userID, bookID uuid.UUID,
) (*model.Review, error) {
	query := `
		SELECT 
			id, user_id, book_id, order_id,
			rating, title, content, images,
			is_verified_purchase, is_approved, is_featured, admin_note,
			created_at, updated_at
		FROM reviews
		WHERE user_id = $1 AND book_id = $2
	`

	review := &model.Review{}
	var images []string

	err := r.pool.QueryRow(ctx, query, userID, bookID).Scan(
		&review.ID,
		&review.UserID,
		&review.BookID,
		&review.OrderID,
		&review.Rating,
		&review.Title,
		&review.Content,
		pq.Array(&images),
		&review.IsVerifiedPurchase,
		&review.IsApproved,
		&review.IsFeatured,
		&review.AdminNote,
		&review.CreatedAt,
		&review.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrReviewNotFound
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	review.Images = images
	return review, nil
}

// =====================================================
// UPDATE
// =====================================================

func (r *postgresReviewRepository) Update(ctx context.Context, review *model.Review) error {
	query := `
		UPDATE reviews
		SET 
			rating = $2,
			title = $3,
			content = $4,
			images = $5,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		review.ID,
		review.Rating,
		review.Title,
		review.Content,
		pq.Array(review.Images),
	)

	if err != nil {
		return fmt.Errorf("failed to update review: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrReviewNotFound
	}

	return nil
}

// =====================================================
// DELETE
// =====================================================

func (r *postgresReviewRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reviews WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrReviewNotFound
	}

	return nil
}

// =====================================================
// LIST BY BOOK
// =====================================================

func (r *postgresReviewRepository) ListByBook(
	ctx context.Context,
	bookID uuid.UUID,
	page, limit int,
) ([]*model.Review, int, error) {
	// Only show approved reviews to public
	query := `
		SELECT 
			r.id, r.user_id, r.book_id, r.order_id,
			r.rating, r.title, r.content, r.images,
			r.is_verified_purchase, r.is_approved, r.is_featured, r.admin_note,
			r.created_at, r.updated_at
		FROM reviews r
		WHERE r.book_id = $1 AND r.is_approved = true
		ORDER BY r.is_featured DESC, r.created_at DESC
		LIMIT $2 OFFSET $3
	`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, bookID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*model.Review
	for rows.Next() {
		review := &model.Review{}
		var images []string

		err := rows.Scan(
			&review.ID,
			&review.UserID,
			&review.BookID,
			&review.OrderID,
			&review.Rating,
			&review.Title,
			&review.Content,
			pq.Array(&images),
			&review.IsVerifiedPurchase,
			&review.IsApproved,
			&review.IsFeatured,
			&review.AdminNote,
			&review.CreatedAt,
			&review.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}

		review.Images = images
		reviews = append(reviews, review)
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM reviews WHERE book_id = $1 AND is_approved = true`
	var total int
	err = r.pool.QueryRow(ctx, countQuery, bookID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	return reviews, total, nil
}

// =====================================================
// LIST BY USER
// =====================================================

func (r *postgresReviewRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	page, limit int,
) ([]*model.Review, int, error) {
	query := `
		SELECT 
			id, user_id, book_id, order_id,
			rating, title, content, images,
			is_verified_purchase, is_approved, is_featured, admin_note,
			created_at, updated_at
		FROM reviews
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*model.Review
	for rows.Next() {
		review := &model.Review{}
		var images []string

		err := rows.Scan(
			&review.ID,
			&review.UserID,
			&review.BookID,
			&review.OrderID,
			&review.Rating,
			&review.Title,
			&review.Content,
			pq.Array(&images),
			&review.IsVerifiedPurchase,
			&review.IsApproved,
			&review.IsFeatured,
			&review.AdminNote,
			&review.CreatedAt,
			&review.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}

		review.Images = images
		reviews = append(reviews, review)
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM reviews WHERE user_id = $1`
	var total int
	err = r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	return reviews, total, nil
}

// =====================================================
// LIST WITH FILTERS
// =====================================================

func (r *postgresReviewRepository) ListWithFilters(
	ctx context.Context,
	filters map[string]interface{},
	page, limit int,
) ([]*model.Review, int, error) {
	// Build dynamic query
	query := `
		SELECT 
			id, user_id, book_id, order_id,
			rating, title, content, images,
			is_verified_purchase, is_approved, is_featured, admin_note,
			created_at, updated_at
		FROM reviews
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Add filters
	if bookID, ok := filters["book_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND book_id = $%d", argCount)
		args = append(args, bookID)
		argCount++
	}

	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, userID)
		argCount++
	}

	if rating, ok := filters["rating"].(int); ok {
		query += fmt.Sprintf(" AND rating = $%d", argCount)
		args = append(args, rating)
		argCount++
	}

	// Only approved reviews for public
	query += " AND is_approved = true"

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, (page-1)*limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*model.Review
	for rows.Next() {
		review := &model.Review{}
		var images []string

		err := rows.Scan(
			&review.ID,
			&review.UserID,
			&review.BookID,
			&review.OrderID,
			&review.Rating,
			&review.Title,
			&review.Content,
			pq.Array(&images),
			&review.IsVerifiedPurchase,
			&review.IsApproved,
			&review.IsFeatured,
			&review.AdminNote,
			&review.CreatedAt,
			&review.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}

		review.Images = images
		reviews = append(reviews, review)
	}

	// Count with same filters
	countQuery := `SELECT COUNT(*) FROM reviews WHERE 1=1`
	countArgs := []interface{}{}
	countArgNum := 1

	if bookID, ok := filters["book_id"].(uuid.UUID); ok {
		countQuery += fmt.Sprintf(" AND book_id = $%d", countArgNum)
		countArgs = append(countArgs, bookID)
		countArgNum++
	}

	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		countQuery += fmt.Sprintf(" AND user_id = $%d", countArgNum)
		countArgs = append(countArgs, userID)
		countArgNum++
	}

	if rating, ok := filters["rating"].(int); ok {
		countQuery += fmt.Sprintf(" AND rating = $%d", countArgNum)
		countArgs = append(countArgs, rating)
		countArgNum++
	}

	countQuery += " AND is_approved = true"

	var total int
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	return reviews, total, nil
}

// =====================================================
// STATISTICS
// =====================================================

func (r *postgresReviewRepository) GetBookStatistics(
	ctx context.Context,
	bookID uuid.UUID,
) (*model.ReviewStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_reviews,
			COALESCE(ROUND(AVG(rating)::numeric, 1), 0) as average_rating
		FROM reviews
		WHERE book_id = $1 AND is_approved = true
	`

	stats := &model.ReviewStatistics{}
	err := r.pool.QueryRow(ctx, query, bookID).Scan(
		&stats.TotalReviews,
		&stats.AverageRating,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// Get rating breakdown
	breakdown, err := r.GetRatingBreakdown(ctx, bookID)
	if err != nil {
		return nil, err
	}
	stats.RatingBreakdown = breakdown

	return stats, nil
}

func (r *postgresReviewRepository) GetRatingBreakdown(
	ctx context.Context,
	bookID uuid.UUID,
) (map[int]int, error) {
	query := `
		SELECT rating, COUNT(*) as count
		FROM reviews
		WHERE book_id = $1 AND is_approved = true
		GROUP BY rating
		ORDER BY rating DESC
	`

	rows, err := r.pool.Query(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := make(map[int]int)
	// Initialize all ratings to 0
	for i := 1; i <= 5; i++ {
		breakdown[i] = 0
	}

	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, fmt.Errorf("failed to scan rating breakdown: %w", err)
		}
		breakdown[rating] = count
	}

	return breakdown, nil
}

// =====================================================
// ELIGIBILITY & VERIFICATION
// =====================================================

func (r *postgresReviewRepository) CheckEligibility(
	ctx context.Context,
	userID, bookID uuid.UUID,
) (bool, string, error) {
	// Check if user has already reviewed this book
	_, err := r.GetByUserAndBook(ctx, userID, bookID)
	if err == nil {
		return false, "You have already reviewed this book", nil
	}
	if err != model.ErrReviewNotFound {
		return false, "", err
	}

	// Check if user has purchased this book
	hasPurchased, _, err := r.HasPurchased(ctx, userID, bookID)
	if err != nil {
		return false, "", err
	}
	if !hasPurchased {
		return false, "You must purchase this book before reviewing", nil
	}

	return true, "", nil
}

func (r *postgresReviewRepository) HasPurchased(
	ctx context.Context,
	userID, bookID uuid.UUID,
) (bool, uuid.UUID, error) {
	// Check if user has any order with this book
	query := `
		SELECT o.id
		FROM orders o
		INNER JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = $1 
		AND oi.book_id = $2
		AND o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
		LIMIT 1
	`

	var orderID uuid.UUID
	err := r.pool.QueryRow(ctx, query, userID, bookID).Scan(&orderID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, uuid.Nil, nil
		}
		return false, uuid.Nil, fmt.Errorf("failed to check purchase: %w", err)
	}

	return true, orderID, nil
}

// =====================================================
// ADMIN OPERATIONS
// =====================================================

func (r *postgresReviewRepository) AdminListReviews(
	ctx context.Context,
	filters map[string]interface{},
	page, limit int,
) ([]*model.Review, int, error) {
	// Build query with admin filters (can see all reviews including unapproved)
	query := `
		SELECT 
			id, user_id, book_id, order_id,
			rating, title, content, images,
			is_verified_purchase, is_approved, is_featured, admin_note,
			created_at, updated_at
		FROM reviews
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if bookID, ok := filters["book_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND book_id = $%d", argCount)
		args = append(args, bookID)
		argCount++
	}

	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, userID)
		argCount++
	}

	if isApproved, ok := filters["is_approved"].(bool); ok {
		query += fmt.Sprintf(" AND is_approved = $%d", argCount)
		args = append(args, isApproved)
		argCount++
	}

	if isFeatured, ok := filters["is_featured"].(bool); ok {
		query += fmt.Sprintf(" AND is_featured = $%d", argCount)
		args = append(args, isFeatured)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (content ILIKE $%d OR title ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, (page-1)*limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*model.Review
	for rows.Next() {
		review := &model.Review{}
		var images []string

		err := rows.Scan(
			&review.ID,
			&review.UserID,
			&review.BookID,
			&review.OrderID,
			&review.Rating,
			&review.Title,
			&review.Content,
			pq.Array(&images),
			&review.IsVerifiedPurchase,
			&review.IsApproved,
			&review.IsFeatured,
			&review.AdminNote,
			&review.CreatedAt,
			&review.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}

		review.Images = images
		reviews = append(reviews, review)
	}

	// Count total (copy filter logic for count query)
	countQuery := `SELECT COUNT(*) FROM reviews WHERE 1=1`
	countArgs := []interface{}{}
	countArgNum := 1

	if bookID, ok := filters["book_id"].(uuid.UUID); ok {
		countQuery += fmt.Sprintf(" AND book_id = $%d", countArgNum)
		countArgs = append(countArgs, bookID)
		countArgNum++
	}

	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		countQuery += fmt.Sprintf(" AND user_id = $%d", countArgNum)
		countArgs = append(countArgs, userID)
		countArgNum++
	}

	if isApproved, ok := filters["is_approved"].(bool); ok {
		countQuery += fmt.Sprintf(" AND is_approved = $%d", countArgNum)
		countArgs = append(countArgs, isApproved)
		countArgNum++
	}

	if isFeatured, ok := filters["is_featured"].(bool); ok {
		countQuery += fmt.Sprintf(" AND is_featured = $%d", countArgNum)
		countArgs = append(countArgs, isFeatured)
		countArgNum++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		countQuery += fmt.Sprintf(" AND (content ILIKE $%d OR title ILIKE $%d)", countArgNum, countArgNum)
		countArgs = append(countArgs, "%"+search+"%")
		countArgNum++
	}

	var total int
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	return reviews, total, nil
}

func (r *postgresReviewRepository) UpdateModeration(
	ctx context.Context,
	id uuid.UUID,
	isApproved bool,
	adminNote *string,
) error {
	query := `
		UPDATE reviews
		SET is_approved = $2, admin_note = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, isApproved, adminNote)
	if err != nil {
		return fmt.Errorf("failed to update moderation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrReviewNotFound
	}

	return nil
}

func (r *postgresReviewRepository) UpdateFeatured(
	ctx context.Context,
	id uuid.UUID,
	isFeatured bool,
) error {
	query := `
		UPDATE reviews
		SET is_featured = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, isFeatured)
	if err != nil {
		return fmt.Errorf("failed to update featured: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrReviewNotFound
	}

	return nil
}

func (r *postgresReviewRepository) GetPendingCount(ctx context.Context) (int, error) {
	// Since auto-approve = true, this will return 0 unless admin manually hides reviews
	query := `SELECT COUNT(*) FROM reviews WHERE is_approved = false`

	var count int
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending count: %w", err)
	}

	return count, nil
}
