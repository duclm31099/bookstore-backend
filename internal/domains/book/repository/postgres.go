package book

import (
	"bookstore-backend/internal/domains/book"
	model "bookstore-backend/internal/domains/book"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

// PostgresRepository - Raw SQL with pgxpool
type postgresRepository struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewPostgresRepository - Constructor
func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) model.RepositoryInterface {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}

// ========================= SEARCH BOOK =====================
// SearchBooks - Full-text search using PostgreSQL tsvector + GIN index
func (r *postgresRepository) SearchBooks(ctx context.Context, req book.SearchBooksRequest) ([]book.BookSearchResponse, error) {
	// Build WHERE clause
	whereConditions := []string{
		"b.deleted_at IS NULL",
		"b.is_active = true",
		"b.search_vector @@ websearch_to_tsquery('simple', $1)",
	}
	args := []interface{}{req.Query}
	argIndex := 2

	// Filter by language if specified
	if req.Language != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("b.language = $%d", argIndex))
		args = append(args, req.Language)
		argIndex++
	}

	whereClause := ""
	for i, cond := range whereConditions {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += cond
	}

	// Build main query with ts_rank_cd for relevance scoring
	query := fmt.Sprintf(`
		SELECT 
			b.id,
			b.title,
			b.slug,
			b.price,
			b.cover_url,
			b.language,
			a.name AS author_name,
			ts_rank_cd(b.search_vector, websearch_to_tsquery('simple', $1), 32) AS rank
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		WHERE %s
		ORDER BY rank DESC, b.view_count DESC
		LIMIT $%d
	`, whereClause, argIndex)

	args = append(args, req.Limit)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[Repository] Search query error: %v", err)
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	// Scan results
	results := make([]book.BookSearchResponse, 0, req.Limit)
	for rows.Next() {
		var result book.BookSearchResponse
		err := rows.Scan(
			&result.ID,
			&result.Title,
			&result.Slug,
			&result.Price,
			&result.CoverURL,
			&result.Language,
			&result.AuthorName,
			&result.Rank,
		)
		if err != nil {
			log.Printf("[Repository] Scan error: %v", err)
			continue
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// ============================================
// API 1: LIST BOOKS (Tối ưu & clean)
// ============================================

// ListBooks - Get list of books with filters, pagination, caching
func (r *postgresRepository) ListBooks(ctx context.Context, filter *book.BookFilter) ([]book.Book, int, error) {
	// Build WHERE clause & args
	whereClause, args := r.buildWhereClause(filter)

	// Get total count
	totalCount, err := r.getBookCount(ctx, whereClause, args)
	if err != nil {
		return nil, 0, err
	}

	// Build main query with JOINs
	query := r.buildListBooksQuery(whereClause, 1)
	logger.Info("buildListBooksQuery", map[string]interface{}{
		"query": query,
	})
	// Append pagination args
	args = append(args, filter.Limit, filter.Offset)

	// Execute query & collect rows
	books, err := r.executeListQuery(ctx, query, args)
	if err != nil {
		return nil, 0, err
	}

	return books, totalCount, nil
}

// GetBookByIDForUpdate - Get book với SELECT FOR UPDATE (lock row)
func (r *postgresRepository) GetBookByIDForUpdate(ctx context.Context, id string) (*model.Book, error) {
	query := `
		SELECT id, title, slug, isbn, author_id, publisher_id, category_id,
		       price, compare_at_price, cost_price, cover_url, description,
		       pages, language, published_year, format, dimensions, weight_grams,
		       ebook_file_url, ebook_file_size_mb, ebook_format,
		       is_active, is_featured, view_count, sold_count,
		       meta_title, meta_description, meta_keywords,
		       rating_average, rating_count, version, images,
		       created_at, updated_at
		FROM books
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`

	var book model.Book
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&book.ID, &book.Title, &book.Slug, &book.ISBN, &book.AuthorID, &book.PublisherID, &book.CategoryID,
		&book.Price, &book.CompareAtPrice, &book.CostPrice, &book.CoverURL, &book.Description,
		&book.Pages, &book.Language, &book.PublishedYear, &book.Format, &book.Dimensions, &book.WeightGrams,
		&book.EbookFileURL, &book.EbookFileSizeMB, &book.EbookFormat,
		&book.IsActive, &book.IsFeatured, &book.ViewCount, &book.SoldCount,
		&book.MetaTitle, &book.MetaDescription, pq.Array(&book.MetaKeywords),
		&book.RatingAverage, &book.RatingCount, &book.Version, pq.Array(&book.Images),
		&book.CreatedAt, &book.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, model.ErrBookNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	return &book, nil
}

// CheckISBNExistsExcept - Check ISBN tồn tại ngoại trừ book hiện tại
func (r *postgresRepository) CheckISBNExistsExcept(ctx context.Context, isbn, excludeID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM books WHERE isbn = $1 AND id != $2 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, isbn, excludeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check ISBN: %w", err)
	}
	return exists, nil
}

// ============================================
// HELPER METHODS - Tách logic
// ============================================

// buildWhereClause - Construct WHERE clause dynamically
// Returns: (whereClause string, args []interface{})
func (r *postgresRepository) buildWhereClause(filter *book.BookFilter) (string, []interface{}) {
	conditions := []string{
		"b.deleted_at IS NULL",
		"b.is_active = true",
	}
	args := []interface{}{}
	argIndex := 1

	// Full-text search
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("b.search_vector @@ plainto_tsquery('english', $%d)", argIndex))
		args = append(args, filter.Search)
		argIndex++
	}

	// Filter by category
	if filter.CategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("b.category_id = $%d", argIndex))
		args = append(args, filter.CategoryID)
		argIndex++
	}

	// Price range
	if filter.PriceMin > 0 {
		conditions = append(conditions, fmt.Sprintf("b.price >= $%d", argIndex))
		args = append(args, filter.PriceMin)
		argIndex++
	}

	if filter.PriceMax > 0 {
		conditions = append(conditions, fmt.Sprintf("b.price <= $%d", argIndex))
		args = append(args, filter.PriceMax)
		argIndex++
	}

	// Language filter
	if filter.Language != "" {
		conditions = append(conditions, fmt.Sprintf("b.language = $%d", argIndex))
		args = append(args, filter.Language)
		argIndex++
	}

	// Optional: is_active filter (admin view)
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("b.is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")
	return whereClause, args
}

// buildListBooksQuery - Construct complete SELECT query with JOINs & ORDER BY
func (r *postgresRepository) buildListBooksQuery(whereClause string, paramCount int) string {
	return fmt.Sprintf(`
    SELECT 
      b.id, b.title, b.slug, b.isbn, b.author_id, b.publisher_id, 
      b.category_id, b.price, b.compare_at_price, b.cost_price, 
      b.cover_url, b.description, b.pages, b.language, b.published_year, 
      b.format, b.dimensions, b.weight_grams, b.ebook_file_url, 
      b.ebook_file_size_mb, b.ebook_format, b.is_active, b.is_featured, 
      b.view_count, b.sold_count, b.meta_title, b.meta_description, 
      b.meta_keywords, b.rating_average, b.rating_count, b.version, 
      b.images, b.created_at, b.updated_at, b.deleted_at,
      a.name AS author_name,
      c.name AS category_name,
      p.name AS publisher_name,
      i.available_quantity AS total_stock
    FROM books b
    LEFT JOIN authors a ON b.author_id = a.id
    LEFT JOIN categories c ON b.category_id = c.id
    LEFT JOIN publishers p ON b.publisher_id = p.id
    LEFT JOIN inventories i ON b.id = i.book_id
    WHERE %s
    GROUP BY b.id, a.id, c.id, p.id
    ORDER BY b.created_at DESC
    LIMIT $%d OFFSET $%d
  `, whereClause, paramCount, paramCount+1)
}

// getBookCount - Get total count for pagination
func (r *postgresRepository) getBookCount(ctx context.Context, whereClause string, args []interface{}) (int, error) {
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM books b
		LEFT JOIN inventories i ON b.id = i.book_id
		WHERE %s
		GROUP BY b.id
	`, whereClause)

	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		log.Printf("[BookRepo] Count query error: %v", err)
		return 0, fmt.Errorf("count query failed: %w", err)
	}

	return totalCount, nil
}

// executeListQuery - Execute query & map rows to Book struct using pgx.CollectRows
func (r *postgresRepository) executeListQuery(ctx context.Context, query string, args []interface{}) ([]book.Book, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[BookRepo] Query error: %v", err)
		return nil, fmt.Errorf("list books query failed: %w", err)
	}

	// Use pgx.CollectRows for cleaner scanning
	books, err := pgx.CollectRows(rows, pgx.RowToStructByName[book.Book])
	if err != nil {
		log.Printf("[BookRepo] Collect rows error: %v", err)
		return nil, fmt.Errorf("collect rows failed: %w", err)
	}

	return books, nil
}

// ============================================
// API 2: GET BOOK DETAIL
// ============================================

func (r *postgresRepository) GetBookByID(ctx context.Context, id string) (*model.Book, []model.InventoryDetailDTO, error) {
	query := `SELECT 
				b.*, 
				a.id AS author_id, a.name AS author_name, a.slug AS author_slug, a.bio AS author_bio,
				c.id AS category_id, c.name AS category_name, c.slug AS category_slug,
				p.id AS publisher_id, p.name AS publisher_name, p.slug AS publisher_slug, p.website AS publisher_website,
				COALESCE(inv.total, 0) AS total_stock,
				COALESCE(inv.details, '[]') AS inventories_json,
				COALESCE(r.avg_rating, 0)::numeric(2,1) AS rating_average,
				COALESCE(r.count, 0) AS rating_count
			FROM books b
			LEFT JOIN authors a ON b.author_id = a.id
			LEFT JOIN categories c ON b.category_id = c.id
			LEFT JOIN publishers p ON b.publisher_id = p.id
			LEFT JOIN LATERAL (
					SELECT SUM(i.available_quantity) AS total, 
							json_agg(json_build_object(
								'location', i.warehouse_location,
								'quantity', i.quantity,
								'reserved_quantity', i.reserved_quantity,
								'available_quantity', i.available_quantity,
								'low_stock_threshold', i.low_stock_threshold,
								'is_low_stock', i.is_low_stock,
								'last_restock_at', i.last_restock_at
							) ORDER BY i.warehouse_location) AS details
					FROM inventories i
					WHERE i.book_id = b.id
			) inv ON true
			LEFT JOIN LATERAL (
					SELECT AVG(rating) AS avg_rating, COUNT(*) AS count
					FROM reviews
					WHERE book_id = b.id
			) r ON true
			WHERE b.id = $1 AND b.deleted_at IS NULL AND b.is_active = true;
			`
	row := r.pool.QueryRow(ctx, query, id)
	var inventoriesJson string
	var book book.Book // entity struct
	err := row.Scan(&book)
	if err == pgx.ErrNoRows {
		return nil, nil, model.ErrBookNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	var inventories []model.InventoryDetailDTO
	_ = json.Unmarshal([]byte(inventoriesJson), &inventories)
	return &book, inventories, nil
}

// UpdateBook - Update book with optimistic locking
func (r *postgresRepository) UpdateBook(ctx context.Context, book *book.Book) error {
	query := `
		UPDATE books
		SET title = $1, slug = $2, isbn = $3, author_id = $4, publisher_id = $5, category_id = $6,
		    price = $7, compare_at_price = $8, cost_price = $9, cover_url = $10, description = $11,
		    pages = $12, language = $13, published_year = $14, format = $15, dimensions = $16, weight_grams = $17,
		    ebook_file_url = $18, ebook_file_size_mb = $19, ebook_format = $20,
		    is_active = $21, is_featured = $22,
		    meta_title = $23, meta_description = $24, meta_keywords = $25,
		    version = $26, images = $27, updated_at = $28
		WHERE id = $29 AND version = $30 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query,
		book.Title, book.Slug, book.ISBN, book.AuthorID, book.PublisherID, book.CategoryID,
		book.Price, book.CompareAtPrice, book.CostPrice, book.CoverURL, book.Description,
		book.Pages, book.Language, book.PublishedYear, book.Format, book.Dimensions, book.WeightGrams,
		book.EbookFileURL, book.EbookFileSizeMB, book.EbookFormat,
		book.IsActive, book.IsFeatured,
		book.MetaTitle, book.MetaDescription, pq.Array(book.MetaKeywords),
		book.Version, pq.Array(book.Images), book.UpdatedAt,
		book.ID, book.Version-1, // WHERE version = old version
	)

	if err != nil {
		return fmt.Errorf("failed to update book: %w", err)
	}

	// Check if any row was updated
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return model.ErrVersionConflict
	}

	return nil
}

// CreateBook - Insert new book to database
func (r *postgresRepository) CreateBook(ctx context.Context, book *model.Book) error {
	query := `
		INSERT INTO books (
			id, title, slug, isbn, author_id, publisher_id, category_id,
			price, compare_at_price, cost_price, cover_url, description,
			pages, language, published_year, format, dimensions, weight_grams,
			ebook_file_url, ebook_file_size_mb, ebook_format,
			is_active, is_featured, view_count, sold_count,
			meta_title, meta_description, meta_keywords,
			rating_average, rating_count, version, images,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21,
			$22, $23, $24, $25,
			$26, $27, $28,
			$29, $30, $31, $32,
			$33, $34
		)
	`

	_, err := r.pool.Exec(ctx, query,
		book.ID, book.Title, book.Slug, book.ISBN, book.AuthorID, book.PublisherID, book.CategoryID,
		book.Price, book.CompareAtPrice, book.CostPrice, book.CoverURL, book.Description,
		book.Pages, book.Language, book.PublishedYear, book.Format, book.Dimensions, book.WeightGrams,
		book.EbookFileURL, book.EbookFileSizeMB, book.EbookFormat,
		book.IsActive, book.IsFeatured, book.ViewCount, book.SoldCount,
		book.MetaTitle, book.MetaDescription, pq.Array(book.MetaKeywords),
		book.RatingAverage, book.RatingCount, book.Version, pq.Array(book.Images),
		book.CreatedAt, book.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert book: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetReviewsHighlight(ctx context.Context, bookID string) ([]model.ReviewDTO, error) {
	query := `
		SELECT r.id, r.rating, r.content, r.created_at, r.title, u.full_name AS user_name
		FROM reviews r
		JOIN users u ON r.user_id = u.id
		WHERE r.book_id = $1
		ORDER BY r.created_at DESC
		LIMIT 3;
	`
	rows, err := r.pool.Query(ctx, query, bookID)
	if err != nil {
		return nil, err
	}
	reviews := []model.ReviewDTO{}
	for rows.Next() {
		var review model.ReviewDTO
		err := rows.Scan(&review.ID, &review.Rating, &review.Content,
			&review.CreatedAt, &review.Title, &review.UserName)
		if err == nil {
			reviews = append(reviews, review)
		}
	}
	return reviews, nil
}

func (r *postgresRepository) IncrementViewCount(ctx context.Context, bookID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE books SET view_count = view_count + 1 WHERE id = $1`, bookID)
	return err
}

// ============================================
// VALIDATION HELPERS
// ============================================

func (r *postgresRepository) ValidateAuthor(ctx context.Context, authorID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM authors WHERE id = $1 AND deleted_at IS NULL)", authorID).Scan(&exists)
	if err != nil {
		log.Printf("[BookRepo] Validate author error: %v", err)
		return false, err
	}
	return exists, nil
}

func (r *postgresRepository) ValidateCategory(ctx context.Context, categoryID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1 AND deleted_at IS NULL)", categoryID).Scan(&exists)
	if err != nil {
		log.Printf("[BookRepo] Validate category error: %v", err)
		return false, err
	}
	return exists, nil
}

func (r *postgresRepository) ValidatePublisher(ctx context.Context, publisherID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM publishers WHERE id = $1 AND deleted_at IS NULL)", publisherID).Scan(&exists)
	if err != nil {
		log.Printf("[BookRepo] Validate publisher error: %v", err)
		return false, err
	}
	return exists, nil
}

// CheckISBNExists - Kiểm tra ISBN đã tồn tại chưa
func (r *postgresRepository) CheckISBNExists(ctx context.Context, isbn string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM books WHERE isbn = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, isbn).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check ISBN: %w", err)
	}
	return exists, nil
}
func (r *postgresRepository) GetBaseBookByID(ctx context.Context, id string) (*book.BaseBookResponse, error) {
	query := `SELECT id, title FROM books WHERE id = $1 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query, id)
	var book book.BaseBookResponse
	err := row.Scan(&book.ID, &book.Title)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

// GenerateUniqueSlug - Tạo slug unique (thêm suffix nếu trùng)
func (r *postgresRepository) GenerateUniqueSlug(ctx context.Context, baseSlug string) (string, error) {
	slug := baseSlug
	counter := 1

	for {
		// Check if slug exists
		query := `SELECT EXISTS(SELECT 1 FROM books WHERE slug = $1 AND deleted_at IS NULL)`
		var exists bool
		err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("failed to check slug: %w", err)
		}

		if !exists {
			return slug, nil
		}

		// Slug exists, try with suffix
		counter++
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)

		// Prevent infinite loop
		if counter > 100 {
			return "", fmt.Errorf("failed to generate unique slug after 100 attempts")
		}
	}
}

// ============================================
// CACHE OPERATIONS
// ============================================

// ============================================
// STUB METHODS (API 3, 4)
// ============================================

func (r *postgresRepository) GetBookBySlug(ctx context.Context, slug string) (*book.Book, error) {
	return nil, nil
}

func (r *postgresRepository) DeleteBook(ctx context.Context, id string) error {
	return nil
}

func (r *postgresRepository) SearchByFullText(ctx context.Context, query string, limit int) ([]book.Book, error) {
	return nil, nil
}

// SoftDeleteBook - Set deleted_at timestamp
func (r *postgresRepository) SoftDeleteBook(ctx context.Context, bookID string, deletedAt time.Time) error {
	query := `
		UPDATE books
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, deletedAt, bookID)
	if err != nil {
		return fmt.Errorf("failed to soft delete book: %w", err)
	}

	// Check if book was found and deleted
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return model.ErrBookNotFound
	}

	return nil
}

// CheckBookHasActiveOrders - Kiểm tra sách có order đang active không
// Active orders: status IN ('pending', 'processing', 'confirmed', 'paid')
func (r *postgresRepository) CheckBookHasActiveOrders(ctx context.Context, bookID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.id
			WHERE oi.book_id = $1 
			  AND o.status IN ('pending', 'processing', 'confirmed', 'paid')
			  AND o.deleted_at IS NULL
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check active orders: %w", err)
	}

	return exists, nil
}

// CheckBookHasReservedInventory - Kiểm tra sách có inventory đang reserved không
func (r *postgresRepository) CheckBookHasReservedInventory(ctx context.Context, bookID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM inventories
			WHERE book_id = $1 
			  AND reserved_quantity > 0
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check reserved inventory: %w", err)
	}

	return exists, nil
}
