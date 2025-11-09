package repository

import (
	"bookstore-backend/internal/domains/author/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/pkg/cache"
)

// postgresRepository implements author.Repository interface
// Uses pgxpool for PostgreSQL and Redis for caching
type postgresRepository struct {
	pool  *pgxpool.Pool // PostgreSQL connection pool
	cache cache.Cache   // Redis cache layer (injected dependency)
}

// NewPostgresRepository creates a new author repository instance
// Dependency injection pattern - receives pool and cache from container
func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) RepositoryInterface {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}

// Cache key constants
const (
	authorCacheKeyPrefix = "author:"
	authorSlugKeyPrefix  = "author:slug:"
	authorListKeyPrefix  = "authors:list:"
	cacheTTL             = 15 * time.Minute
)

// Create inserts new author with generated ID and timestamps
func (r *postgresRepository) Create(ctx context.Context, a *model.Author) (*model.Author, error) {
	query := `
        INSERT INTO authors (name, slug, bio, photo_url, version)
        VALUES ($1, $2, $3, $4, 0)
        RETURNING id, name, slug, bio, photo_url, version, created_at, updated_at
    `

	var created model.Author
	err := r.pool.QueryRow(
		ctx,
		query,
		a.Name,
		a.Slug,
		a.Bio,
		a.PhotoURL,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Slug,
		&created.Bio,
		&created.PhotoURL,
		&created.Version,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation on slug
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				if strings.Contains(pgErr.Message, "slug") {
					return nil, model.ErrDuplicateSlug
				}
			}
		}
		return nil, fmt.Errorf("failed to create author: %w", err)
	}

	// Invalidate list cache after creation
	r.invalidateListCache(ctx)

	return &created, nil
}

// GetByID retrieves author by UUID with caching
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Author, error) {
	// Try cache first
	cacheKey := authorCacheKeyPrefix + id.String()

	var a model.Author
	cached, err := r.cache.Get(ctx, cacheKey, &a)
	fmt.Println("CACHED cacheKey: ", cacheKey)
	if err == nil && cached {
		// Cache hit
		return &a, nil
	}

	// Cache miss - query database
	query := `
        SELECT id, name, slug, bio, photo_url, version, created_at, updated_at
        FROM authors
        WHERE id = $1
    `

	err = r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID,
		&a.Name,
		&a.Slug,
		&a.Bio,
		&a.PhotoURL,
		&a.Version,
		&a.CreatedAt,
		&a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrAuthorNotFound
		}
		return nil, fmt.Errorf("failed to get author by id: %w", err)
	}

	// Store in cache for next time
	if data, err := json.Marshal(a); err == nil {
		r.cache.Set(ctx, cacheKey, string(data), cacheTTL)
	}

	return &a, nil
}

// GetBySlug retrieves author by URL slug with caching
func (r *postgresRepository) GetBySlug(ctx context.Context, slug string) (*model.Author, error) {
	// Try cache first
	cacheKey := authorSlugKeyPrefix + slug

	var a model.Author
	cached, err := r.cache.Get(ctx, cacheKey, &a)
	if err == nil && cached {
		// Cache hit
		return &a, nil
	}

	// Cache miss - query database
	query := `
        SELECT id, name, slug, bio, photo_url, version, created_at, updated_at
        FROM authors
        WHERE slug = $1
    `

	err = r.pool.QueryRow(ctx, query, slug).Scan(
		&a.ID,
		&a.Name,
		&a.Slug,
		&a.Bio,
		&a.PhotoURL,
		&a.Version,
		&a.CreatedAt,
		&a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrAuthorNotFound
		}
		return nil, fmt.Errorf("failed to get author by slug: %w", err)
	}

	// Store in cache
	if data, err := json.Marshal(a); err == nil {
		// Cache both by ID and slug
		r.cache.Set(ctx, cacheKey, string(data), cacheTTL)
		r.cache.Set(ctx, authorCacheKeyPrefix+a.ID.String(), string(data), cacheTTL)
	}

	return &a, nil
}

// GetAll retrieves paginated list with filtering and sorting
func (r *postgresRepository) GetAll(ctx context.Context, filter model.AuthorFilter) ([]model.Author, int64, error) {
	// Build dynamic query
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
        SELECT id, name, slug, bio, photo_url, version, created_at, updated_at
        FROM authors
        WHERE 1=1
    `)

	args := []interface{}{}
	argPos := 1

	// Add search filter if provided
	if filter.Search != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND name ILIKE $%d", argPos))
		args = append(args, "%"+filter.Search+"%")
		argPos++
	}

	// Add sorting
	sortColumn := "created_at" // default
	switch filter.SortBy {
	case "name":
		sortColumn = "name"
	case "updated_at":
		sortColumn = "updated_at"
	}

	sortOrder := "DESC" // default
	if filter.Order == "asc" {
		sortOrder = "ASC"
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortColumn, sortOrder))

	// Add pagination
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, filter.Limit, filter.Offset)

	// Execute query
	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query authors: %w", err)
	}
	defer rows.Close()

	// Scan results
	var authors []model.Author
	for rows.Next() {
		var a model.Author
		err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Slug,
			&a.Bio,
			&a.PhotoURL,
			&a.Version,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan author: %w", err)
		}
		authors = append(authors, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating authors: %w", err)
	}

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM authors WHERE 1=1`
	countArgs := []interface{}{}

	if filter.Search != "" {
		countQuery += " AND name ILIKE $1"
		countArgs = append(countArgs, "%"+filter.Search+"%")
	}

	var total int64
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count authors: %w", err)
	}

	return authors, total, nil
}

// Update updates author with optimistic locking
func (r *postgresRepository) Update(ctx context.Context, a *model.Author, currentVersion int) (*model.Author, error) {
	// Critical: WHERE clause includes version check
	query := `
        UPDATE authors
        SET 
            name = $1,
            slug = $2,
            bio = $3,
            photo_url = $4,
            version = version + 1,
            updated_at = NOW()
        WHERE id = $5 AND version = $6
        RETURNING id, name, slug, bio, photo_url, version, created_at, updated_at
    `

	var updated model.Author
	err := r.pool.QueryRow(
		ctx,
		query,
		a.Name,
		a.Slug,
		a.Bio,
		a.PhotoURL,
		a.ID,
		currentVersion,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Slug,
		&updated.Bio,
		&updated.PhotoURL,
		&updated.Version,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if author exists or version mismatch
			exists, checkErr := r.ExistsByID(ctx, a.ID)
			if checkErr != nil {
				return nil, checkErr
			}

			if !exists {
				return nil, model.ErrAuthorNotFound
			}

			// Author exists but version doesn't match = conflict
			return nil, model.ErrVersionMismatch
		}

		// Check for duplicate slug
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				if strings.Contains(pgErr.Message, "slug") {
					return nil, model.ErrDuplicateSlug
				}
			}
		}

		return nil, fmt.Errorf("failed to update author: %w", err)
	}

	// Invalidate caches
	r.invalidateAuthorCache(ctx, a.ID, a.Slug)
	r.invalidateListCache(ctx)

	return &updated, nil
}

// Delete removes author by ID
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get slug first for cache invalidation
	var slug string
	err := r.pool.QueryRow(ctx, "SELECT slug FROM authors WHERE id = $1", id).Scan(&slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrAuthorNotFound
		}
	}

	query := `DELETE FROM authors WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		// Check for foreign key constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23503" { // foreign_key_violation
				return model.ErrAuthorHasBooks
			}
		}
		return fmt.Errorf("failed to delete author: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return model.ErrAuthorNotFound
	}

	// Invalidate caches
	r.invalidateAuthorCache(ctx, id, slug)
	r.invalidateListCache(ctx)

	return nil
}

// BulkDelete deletes multiple authors with transaction
func (r *postgresRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) (int, []model.BulkError, error) {
	if len(ids) == 0 {
		return 0, nil, nil
	}

	successCount := 0
	var bulkErrors []model.BulkError

	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete each author individually
	for _, id := range ids {
		query := `DELETE FROM authors WHERE id = $1`
		cmdTag, err := tx.Exec(ctx, query, id)

		if err != nil {
			bulkErrors = append(bulkErrors, model.BulkError{
				ID:      id,
				Message: err.Error(),
			})
			continue
		}

		if cmdTag.RowsAffected() == 0 {
			bulkErrors = append(bulkErrors, model.BulkError{
				ID:      id,
				Message: "author not found",
			})
			continue
		}

		successCount++
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate caches
	r.invalidateListCache(ctx)
	for _, id := range ids {
		r.cache.Delete(ctx, authorCacheKeyPrefix+id.String())
	}

	return successCount, bulkErrors, nil
}

// ExistsByID checks if author exists (lightweight query)
func (r *postgresRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM authors WHERE id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check author existence: %w", err)
	}

	return exists, nil
}

// ExistsBySlug checks if slug is taken
func (r *postgresRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM authors WHERE slug = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}

	return exists, nil
}

// GetBookCount returns number of books by this author
func (r *postgresRepository) GetBookCount(ctx context.Context, authorID uuid.UUID) (int, error) {
	query := `
        SELECT COUNT(*)
        FROM books
        WHERE author_id = $1
    `

	var count int
	err := r.pool.QueryRow(ctx, query, authorID).Scan(&count)
	if err != nil {
		// If table doesn't exist yet, return 0
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "42P01" { // undefined_table
				return 0, nil
			}
		}
		return 0, fmt.Errorf("failed to get book count: %w", err)
	}

	return count, nil
}

// Search performs full-text search with relevance ranking
// Search performs full-text search on slug using PostgreSQL engine
func (r *postgresRepository) Search(ctx context.Context, query string, filter model.AuthorFilter) ([]model.Author, int64, error) {
	sanitizedQuery := normalizeTSQuery(query)

	if sanitizedQuery == "" {
		return []model.Author{}, 0, nil
	}

	// Full-text search using to_tsvector và plainto_tsquery trên slug
	searchQuery := `
		SELECT 
			id, 
			name, 
			slug, 
			bio, 
			photo_url, 
			version, 
			created_at, 
			updated_at
		FROM (
			SELECT 
				id, 
				name, 
				slug, 
				bio, 
				photo_url, 
				version, 
				created_at, 
				updated_at,
				ts_rank(to_tsvector('simple', slug), plainto_tsquery('simple', $1)) as rank
			FROM authors
			WHERE to_tsvector('simple', slug) @@ plainto_tsquery('simple', $1)
		) search_results
		ORDER BY 
			rank DESC,
			CASE 
				WHEN slug = $2 THEN 0           -- Exact match
				WHEN slug LIKE $3 THEN 1        -- Starts with
				ELSE 2                          -- Contains
			END,
			slug ASC
		LIMIT $4 OFFSET $5
	`

	exactPattern := sanitizedQuery
	startsWithPattern := sanitizedQuery + "%"

	rows, err := r.pool.Query(
		ctx,
		searchQuery,
		sanitizedQuery,
		exactPattern,
		startsWithPattern,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search authors: %w", err)
	}
	defer rows.Close()

	var authors []model.Author
	for rows.Next() {
		var a model.Author
		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Slug,
			&a.Bio,
			&a.PhotoURL,
			&a.Version,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan author: %w", err)
		}
		authors = append(authors, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating search results: %w", err)
	}

	// Get total count with full-text search
	countQuery := `
		SELECT COUNT(*)
		FROM authors
		WHERE to_tsvector('simple', slug) @@ plainto_tsquery('simple', $1)
	`

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, sanitizedQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	return authors, total, nil
}

// normalizeTSQuery sanitizes input for PostgreSQL full-text search
func normalizeTSQuery(input string) string {
	input = strings.TrimSpace(input)

	// Replace multiple spaces with single space
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")

	// Remove special characters that can break ts_query
	specialChars := []string{"!", "&", "|", "(", ")", "<", ">", ":"}
	for _, char := range specialChars {
		input = strings.ReplaceAll(input, char, "")
	}

	return strings.ToLower(input)
}

// Cache helper methods

func (r *postgresRepository) invalidateAuthorCache(ctx context.Context, id uuid.UUID, slug string) {
	r.cache.Delete(ctx, authorCacheKeyPrefix+id.String())
	r.cache.Delete(ctx, authorSlugKeyPrefix+slug)
}

func (r *postgresRepository) invalidateListCache(ctx context.Context) {
	// Pattern-based deletion for list caches
	r.cache.DeletePattern(ctx, authorListKeyPrefix+"*")
}
