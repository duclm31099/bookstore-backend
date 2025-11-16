package repository

import (
	"bookstore-backend/internal/domains/book/model"
	"bookstore-backend/pkg/cache"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
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
func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) RepositoryInterface {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}

// ========================= SEARCH BOOK =====================
// SearchBooks - Full-text search using PostgreSQL tsvector + GIN index
func (r *postgresRepository) SearchBooks(ctx context.Context, req model.SearchBooksRequest) ([]model.BookSearchResponse, error) {
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
	results := make([]model.BookSearchResponse, 0, req.Limit)
	for rows.Next() {
		var result model.BookSearchResponse
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
func (r *postgresRepository) ListBooks(ctx context.Context, filter *model.BookFilter) ([]model.Book, int, error) {
	// Build WHERE clause & args
	whereClause, args := r.buildWhereClause(filter)

	// Get total count
	totalCount, err := r.getBookCount(ctx, whereClause, args)
	if err != nil {
		return nil, 0, err
	}

	// Build main query with JOINs
	query := r.buildListBooksQuery(whereClause, 1)
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
func (r *postgresRepository) buildWhereClause(filter *model.BookFilter) (string, []interface{}) {
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

// buildListBooksQuery - FIXED: Use warehouse_inventory + books_total_stock VIEW
func (r *postgresRepository) buildListBooksQuery(whereClause string, paramCount int) string {
	return fmt.Sprintf(`
		SELECT 
			b.id, b.title, b.slug, b.isbn, b.author_id, b.publisher_id, 
			b.category_id, b.price, b.compare_at_price, b.cost_price, 
			b.cover_url, b.description, b.pages, b.language, b.published_year, 
			b.format, b.dimensions, b.weight_grams, b.ebook_file_url, 
			b.ebook_file_size_mb, b.ebook_format, b.is_active, b.is_featured, 
			b.view_count, b.sold_count, b.meta_title, b.meta_description, 
			COALESCE(b.meta_keywords, ARRAY[]::text[]) AS meta_keywords, b.rating_average, b.rating_count, b.version, 
			b.images, b.created_at, b.updated_at, b.deleted_at,
			a.name AS author_name,
			c.name AS category_name,
			p.name AS publisher_name,
			COALESCE(bts.available, 0) AS total_stock
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		LEFT JOIN categories c ON b.category_id = c.id
		LEFT JOIN publishers p ON b.publisher_id = p.id
		LEFT JOIN books_total_stock bts ON b.id = bts.book_id
		WHERE %s
		ORDER BY b.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramCount, paramCount+1)
}

// getBookCount - FIXED: Remove GROUP BY
func (r *postgresRepository) getBookCount(ctx context.Context, whereClause string, args []interface{}) (int, error) {
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM books b
		WHERE %s
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
func (r *postgresRepository) executeListQuery(ctx context.Context, query string, args []interface{}) ([]model.Book, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[BookRepo] Query error: %v", err)
		return nil, fmt.Errorf("list books query failed: %w", err)
	}

	// Use pgx.CollectRows for cleaner scanning
	books, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Book])
	if err != nil {
		log.Printf("[BookRepo] Collect rows error: %v", err)
		return nil, fmt.Errorf("collect rows failed: %w", err)
	}

	return books, nil
}

// ============================================
// API 2: GET BOOK DETAIL - FIXED
// ============================================
// Book struct với tất cả trường từ books table + joined data

func (r *postgresRepository) GetBookByID(ctx context.Context, id string) (*model.BookDetailRes, []model.InventoryDetailDTO, error) {
	query := `SELECT 
        b.id, b.title, b.slug, b.isbn, b.author_id, b.publisher_id, b.category_id,
        b.price, b.compare_at_price, b.cost_price, b.cover_url, b.description,
        b.pages, b.language, b.published_year, b.format, b.dimensions,
        b.weight_grams, b.ebook_file_url, b.ebook_file_size_mb, b.ebook_format,
        b.is_active, b.is_featured, b.view_count, b.sold_count,
        b.meta_title, b.meta_description, 
        COALESCE(b.meta_keywords, ARRAY[]::text[]) AS meta_keywords,
        b.version, 
        COALESCE(b.images, ARRAY[]::text[]) AS images,
        b.created_at, b.updated_at, b.deleted_at,
        a.name AS author_name, a.slug AS author_slug, a.bio AS author_bio,
        c.name AS category_name, c.slug AS category_slug,
        p.name AS publisher_name, p.slug AS publisher_slug, p.website AS publisher_website,
        COALESCE(inv.total, 0) AS total_stock,
        COALESCE(inv.details, '[]'::json) AS inventories_json,
        COALESCE(r.avg_rating, 0)::numeric(2,1) AS rating_average,
        COALESCE(r.count, 0) AS rating_count
    FROM books b
    LEFT JOIN authors a ON b.author_id = a.id
    LEFT JOIN categories c ON b.category_id = c.id
    LEFT JOIN publishers p ON b.publisher_id = p.id
    LEFT JOIN LATERAL (
        SELECT 
            SUM(wi.quantity - wi.reserved) AS total, 
            json_agg(json_build_object(
                'warehouse_id', wi.warehouse_id,
                'warehouse_name', w.name,
                'warehouse_code', w.code,
                'quantity', wi.quantity,
                'reserved', wi.reserved,
                'available', wi.quantity - wi.reserved,
                'alert_threshold', wi.alert_threshold,
                'is_low_stock', wi.quantity < wi.alert_threshold,
                'last_restocked_at', wi.last_restocked_at
            ) ORDER BY w.name) AS details
        FROM warehouse_inventory wi
        INNER JOIN warehouses w ON wi.warehouse_id = w.id
        WHERE wi.book_id = b.id
            AND w.deleted_at IS NULL
            AND w.is_active = true
    ) inv ON true
    LEFT JOIN LATERAL (
        SELECT AVG(rating) AS avg_rating, COUNT(*) AS count
        FROM reviews
        WHERE book_id = b.id
    ) r ON true
    WHERE b.id = $1 AND b.deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, id)

	var book model.BookDetailRes
	var inventoriesJSON []byte // Scan JSON as []byte

	// Scan theo đúng thứ tự cột trong SELECT
	err := row.Scan(
		// b.* fields (43 cột từ books table)
		&book.ID,
		&book.Title,
		&book.Slug,
		&book.ISBN,
		&book.AuthorID,
		&book.PublisherID,
		&book.CategoryID,
		&book.Price,
		&book.CompareAtPrice,
		&book.CostPrice,
		&book.CoverURL,
		&book.Description,
		&book.Pages,
		&book.Language,
		&book.PublishedYear,
		&book.Format,
		&book.Dimensions,
		&book.WeightGrams,
		&book.EbookFileURL,
		&book.EbookFileSizeMB,
		&book.EbookFormat,
		&book.IsActive,
		&book.IsFeatured,
		&book.ViewCount,
		&book.SoldCount,
		&book.MetaTitle,
		&book.MetaDescription,
		&book.MetaKeywords,
		&book.Version,
		&book.Images,
		&book.CreatedAt,
		&book.UpdatedAt,
		&book.DeletedAt,

		// Author fields (3 cột)
		&book.AuthorName,
		&book.AuthorSlug,
		&book.AuthorBio,

		// Category fields (2 cột)
		&book.CategoryName,
		&book.CategorySlug,

		// Publisher fields (3 cột)
		&book.PublisherName,
		&book.PublisherSlug,
		&book.PublisherWebsite,

		// Inventory aggregate (1 cột)
		&book.TotalStock,
		&inventoriesJSON, // Scan JSON vào []byte

		// Rating aggregate (2 cột)
		&book.RatingAverage,
		&book.RatingCount,
	)

	if err == pgx.ErrNoRows {
		return nil, nil, model.ErrBookNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scan book: %w", err)
	}

	// Parse inventories JSON
	var inventories []model.InventoryDetailDTO
	if len(inventoriesJSON) > 0 && string(inventoriesJSON) != "[]" {
		if err := json.Unmarshal(inventoriesJSON, &inventories); err != nil {
			return nil, nil, fmt.Errorf("failed to parse inventories: %w", err)
		}
	}

	return &book, inventories, nil
}

// UpdateBook - Update book with optimistic locking
func (r *postgresRepository) UpdateBook(ctx context.Context, book *model.Book) error {
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
		book.ID, book.Version-1,
	)

	if err != nil {
		return fmt.Errorf("failed to update book: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return model.ErrVersionConflict
	}

	return nil
}

// CreateBook - Insert new book to database
func (r *postgresRepository) CreateBook(ctx context.Context, book *model.Book) (uuid.UUID, error) {
	query := `
		INSERT INTO books ( title, slug, isbn, author_id, publisher_id, category_id,
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
			$33
		)
		RETURNING id
	`
	var bookID uuid.UUID
	err := r.pool.QueryRow(ctx, query, book.Title, book.Slug, book.ISBN, book.AuthorID, book.PublisherID, book.CategoryID,
		book.Price, book.CompareAtPrice, book.CostPrice, book.CoverURL, book.Description,
		book.Pages, book.Language, book.PublishedYear, book.Format, book.Dimensions, book.WeightGrams,
		book.EbookFileURL, book.EbookFileSizeMB, book.EbookFormat,
		book.IsActive, book.IsFeatured, book.ViewCount, book.SoldCount,
		book.MetaTitle, book.MetaDescription, pq.Array(book.MetaKeywords),
		book.RatingAverage, book.RatingCount, book.Version, pq.Array(book.Images),
		book.CreatedAt, book.UpdatedAt,
	).Scan(&bookID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert book: %w", err)
	}

	return bookID, nil
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
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM authors WHERE id = $1)", authorID).Scan(&exists)
	if err != nil {
		log.Printf("[BookRepo] Validate author error: %v", err)
		return false, err
	}
	return exists, nil
}

func (r *postgresRepository) ValidateCategory(ctx context.Context, categoryID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", categoryID).Scan(&exists)
	if err != nil {
		log.Printf("[BookRepo] Validate category error: %v", err)
		return false, err
	}
	return exists, nil
}

func (r *postgresRepository) ValidatePublisher(ctx context.Context, publisherID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM publishers WHERE id = $1)", publisherID).Scan(&exists)
	if err != nil {
		log.Printf("[BookRepo] Validate publisher error: %v", err)
		return false, err
	}
	return exists, nil
}

func (r *postgresRepository) CheckISBNExists(ctx context.Context, isbn string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM books WHERE isbn = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, isbn).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check ISBN: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) GetBaseBookByID(ctx context.Context, id string) (*model.BaseBookResponse, error) {
	query := `SELECT id, title FROM books WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var book model.BaseBookResponse
	err := row.Scan(&book.ID, &book.Title)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *postgresRepository) GenerateUniqueSlug(ctx context.Context, baseSlug string) (string, error) {
	slug := baseSlug
	counter := 1

	for {
		query := `SELECT EXISTS(SELECT 1 FROM books WHERE slug = $1)`
		var exists bool
		err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("failed to check slug: %w", err)
		}

		if !exists {
			return slug, nil
		}

		counter++
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)

		if counter > 100 {
			return "", fmt.Errorf("failed to generate unique slug after 100 attempts")
		}
	}
}

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

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return model.ErrBookNotFound
	}

	return nil
}

// CheckBookHasActiveOrders - UNCHANGED (no inventory reference)
func (r *postgresRepository) CheckBookHasActiveOrders(ctx context.Context, bookID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.id
			WHERE oi.book_id = $1 
				AND o.status IN ('pending', 'processing', 'confirmed', 'paid')
				AND o.cancelled_at IS NULL
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check active orders: %w", err)
	}

	return exists, nil
}

// CheckBookHasReservedInventory - FIXED: Use warehouse_inventory
func (r *postgresRepository) CheckBookHasReservedInventory(ctx context.Context, bookID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM warehouse_inventory
			WHERE book_id = $1 
				AND reserved > 0
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, bookID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check reserved inventory: %w", err)
	}

	return exists, nil
}

// CreateBookWithTx tạo book trong transaction
func (r *postgresRepository) CreateBookWithTx(ctx context.Context, tx pgx.Tx, book *model.Book) error {
	query := `
        INSERT INTO books (
            id, title, slug, isbn, author_id, publisher_id, category_id,
            price, compare_at_price, cost_price, description,
            pages, language, published_year, format, dimensions, weight_grams,
            is_active, is_featured,
            meta_title, meta_description, meta_keywords,
            version, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7,
            $8, $9, $10, $11,
            $12, $13, $14, $15, $16, $17,
            $18, $19,
            $20, $21, $22,
            $23, $24, $25
        )
    `

	now := time.Now()
	book.CreatedAt = now
	book.UpdatedAt = now

	_, err := tx.Exec(ctx, query,
		book.ID,
		book.Title,
		book.Slug,
		book.ISBN,
		book.AuthorID,
		book.PublisherID,
		book.CategoryID,
		book.Price,
		book.CompareAtPrice,
		book.CostPrice,
		book.Description,
		book.Pages,
		book.Language,
		book.PublishedYear,
		book.Format,
		book.Dimensions,
		book.WeightGrams,
		book.IsActive,
		book.IsFeatured,
		book.MetaTitle,
		book.MetaDescription,
		book.MetaKeywords,
		book.Version,
		book.CreatedAt,
		book.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create book: %w", err)
	}

	return nil
}

// FindBySlugWithTx tìm book by slug (trong transaction)
func (r *postgresRepository) FindBySlugWithTx(ctx context.Context, tx pgx.Tx, slug string) (*model.Book, error) {
	query := `
        SELECT id, title, slug
        FROM books
        WHERE slug = $1 AND deleted_at IS NULL
        LIMIT 1
    `

	var book model.Book
	err := tx.QueryRow(ctx, query, slug).Scan(
		&book.ID,
		&book.Title,
		&book.Slug,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find book: %w", err)
	}

	return &book, nil
}
