package repository

import (
	"bookstore-backend/internal/domains/publisher/model"
	"bookstore-backend/pkg/cache"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresRepository implements publisher.Repository
// Uses pgxpool for PostgreSQL connection management
type postgresRepository struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewPostgresRepository creates a new publisher repository instance
// Dependency injection pattern - receives pool from container
func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) RepositoryInterface {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}

// Create inserts a new publisher record
func (r *postgresRepository) Create(ctx context.Context, pub *model.Publisher) (*model.Publisher, error) {
	query := `
    INSERT INTO publishers (name, slug, website, email, phone, address, description)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, name, slug, website, email, phone, address, description
  `
	args := []interface{}{
		pub.Name, pub.Slug, pub.Website, pub.Email, pub.Phone, pub.Address, pub.Description,
	}
	row := r.pool.QueryRow(ctx, query, args...)

	var createdPub model.Publisher
	err := row.Scan(
		&createdPub.ID,
		&createdPub.Name,
		&createdPub.Slug,
		&createdPub.Website,
		&createdPub.Email,
		&createdPub.Phone,
		&createdPub.Address,
		&createdPub.Description,
	)
	if err != nil {
		fmt.Println("error", err)
		if strings.Contains(err.Error(), "unique constraint") {
			return nil, model.NewPublisherSlugAlreadyExists(pub.Slug)
		}
		return nil, model.NewCreatePublisherError(err)
	}
	return &createdPub, nil
}

// GetByID retrieves a publisher by ID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Publisher, error) {
	query := `
    SELECT id, name, slug, 
			COALESCE(website, '') AS website,
      COALESCE(email, '')   AS email,
      COALESCE(phone, '')   AS phone,
			is_active,
			description,
			address,
			created_at, updated_at
    FROM publishers
    WHERE id = $1
  `

	row := r.pool.QueryRow(ctx, query, id)

	var pub model.Publisher
	err := row.Scan(
		&pub.ID,
		&pub.Name,
		&pub.Slug,
		&pub.Website,
		&pub.Email,
		&pub.Phone,
		&pub.IsActive,
		&pub.Description,
		&pub.Address,
		&pub.CreatedAt,
		&pub.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get publisher by id: %w", err)
	}

	return &pub, nil
}

// GetBySlug retrieves a publisher by slug
func (r *postgresRepository) GetBySlug(ctx context.Context, slug string) (*model.Publisher, error) {
	query := `
    SELECT id, name, slug,
			COALESCE(website, '') AS website,
      COALESCE(email, '')   AS email,
      COALESCE(phone, '')   AS phone,
			is_active, description, address,
			created_at, updated_at
    FROM publishers
    WHERE slug = $1
  `

	row := r.pool.QueryRow(ctx, query, slug)

	var pub model.Publisher
	err := row.Scan(
		&pub.ID,
		&pub.Name,
		&pub.Slug,
		&pub.Website,
		&pub.Email,
		&pub.Phone,
		&pub.IsActive,
		&pub.Description,
		&pub.Address,
		&pub.CreatedAt,
		&pub.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get publisher by slug: %w", err)
	}

	return &pub, nil
}

// List retrieves all publishers with pagination
func (r *postgresRepository) List(ctx context.Context, offset, limit int) ([]*model.Publisher, error) {
	query := `
    SELECT id, name, slug, 
		  COALESCE(website, '') AS website,
      COALESCE(email, '')   AS email,
      COALESCE(phone, '')   AS phone
    FROM publishers
    ORDER BY created_at DESC
    LIMIT $1 OFFSET $2
  `
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list publishers: %w", err)
	}
	defer rows.Close()
	var publishers []*model.Publisher

	for rows.Next() {
		var pub model.Publisher
		err := rows.Scan(
			&pub.ID,
			&pub.Name,
			&pub.Slug,
			&pub.Website,
			&pub.Email,
			&pub.Phone,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan publisher row: %w", err)
		}
		publishers = append(publishers, &pub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating publisher rows: %w", err)
	}

	return publishers, nil
}

// Count returns total number of publishers
func (r *postgresRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM publishers`

	row := r.pool.QueryRow(ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count publishers: %w", err)
	}

	return count, nil
}

// Update updates publisher information (except slug & id)
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, pub *model.Publisher) (*model.Publisher, error) {
	query := `
    UPDATE publishers
    SET name = $1, website = $2, email = $3, phone = $4, updated_at = NOW()
    WHERE id = $5
    RETURNING id, name, slug, 
			COALESCE(website, '') AS website,
      COALESCE(email, '')   AS email,
      COALESCE(phone, '')   AS phone, 
			created_at, updated_at
  `

	row := r.pool.QueryRow(ctx, query, pub.Name, pub.Website, pub.Email, pub.Phone, id)

	var updatedPub model.Publisher
	err := row.Scan(
		&updatedPub.ID,
		&updatedPub.Name,
		&updatedPub.Slug,
		&updatedPub.Website,
		&updatedPub.Email,
		&updatedPub.Phone,
		&updatedPub.CreatedAt,
		&updatedPub.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("publisher not found")
		}
		return nil, fmt.Errorf("failed to update publisher: %w", err)
	}

	return &updatedPub, nil
}

// Delete removes a publisher record
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First check if publisher has books
	countQuery := `SELECT COUNT(*) FROM books WHERE publisher_id = $1`
	row := r.pool.QueryRow(ctx, countQuery, id)

	var bookCount int
	err := row.Scan(&bookCount)
	if err != nil {
		return model.NewDeletePublisherError(err)
	}

	if bookCount > 0 {
		return model.NewPublisherHasBooks(id.String())
	}

	// Delete publisher
	query := `
		UPDATE publishers 
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id)

	if err != nil {
		return model.NewDeletePublisherError(err)
	}

	if result.RowsAffected() == 0 {
		return model.NewPublisherNotFound()
	}

	return nil
}

// GetWithBooks retrieves publisher with associated books
func (r *postgresRepository) GetWithBooks(ctx context.Context, id uuid.UUID) (*model.PublisherWithBooksResponse, error) {
	// First get publisher
	pubQuery := `
    SELECT id, name, slug, 
			COALESCE(website, '') AS website,
      COALESCE(email, '')   AS email,
      COALESCE(phone, '')   AS phone, 
			address, 
			description
    FROM publishers
    WHERE id = $1
  `

	row := r.pool.QueryRow(ctx, pubQuery, id)

	var pub model.PublisherWithBooksResponse
	err := row.Scan(
		&pub.ID,
		&pub.Name,
		&pub.Slug,
		&pub.Website,
		&pub.Email,
		&pub.Phone,
		&pub.Address,
		&pub.Description,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get publisher: %w", err)
	}

	// Get associated books
	booksQuery := `
    SELECT id, title, slug
    FROM books
    WHERE publisher_id = $1
    ORDER BY created_at DESC
  `

	rows, err := r.pool.Query(ctx, booksQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get books for publisher: %w", err)
	}
	defer rows.Close()

	var books []model.BookBasic

	for rows.Next() {
		var book model.BookBasic
		err := rows.Scan(&book.ID, &book.Title, &book.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to scan book row: %w", err)
		}
		books = append(books, book)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating book rows: %w", err)
	}

	pub.Books = books

	return &pub, nil
}

// ListWithBooks retrieves all publishers with their books (paginated)
func (r *postgresRepository) ListWithBooks(ctx context.Context, offset, limit int) ([]*model.PublisherWithBooksResponse, error) {
	// Get publishers with pagination
	pubQuery := `
    SELECT id, name, slug,
			COALESCE(website, '') AS website,
      COALESCE(email, '')   AS email,
      COALESCE(phone, '')   AS phone, 
		 	created_at, updated_at
    FROM publishers
    ORDER BY created_at DESC
    LIMIT $1 OFFSET $2
  `

	rows, err := r.pool.Query(ctx, pubQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list publishers: %w", err)
	}
	defer rows.Close()

	var publishers []*model.PublisherWithBooksResponse

	for rows.Next() {
		var pub model.PublisherWithBooksResponse
		err := rows.Scan(
			&pub.ID,
			&pub.Name,
			&pub.Slug,
			&pub.Website,
			&pub.Email,
			&pub.Phone,
			&pub.CreatedAt,
			&pub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan publisher row: %w", err)
		}

		publishers = append(publishers, &pub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating publisher rows: %w", err)
	}

	// Get books for each publisher
	for _, pub := range publishers {
		booksQuery := `
      SELECT id, title, slug
      FROM books
      WHERE publisher_id = $1
      ORDER BY created_at DESC
    `

		bookRows, err := r.pool.Query(ctx, booksQuery, pub.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get books for publisher: %w", err)
		}
		defer bookRows.Close()

		var books []model.BookBasic

		for bookRows.Next() {
			var book model.BookBasic
			err := bookRows.Scan(&book.ID, &book.Title, &book.Slug)
			if err != nil {
				return nil, fmt.Errorf("failed to scan book row: %w", err)
			}
			books = append(books, book)
		}

		if err = bookRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating book rows: %w", err)
		}

		pub.Books = books
	}

	return publishers, nil
}

// FindByNameCaseInsensitive tìm publisher by name
func (r *postgresRepository) FindByNameCaseInsensitive(ctx context.Context, name string) (*model.Publisher, error) {
	query := `
        SELECT id, name, slug, description, website,
               created_at, updated_at, deleted_at
        FROM publishers
        WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
        LIMIT 1
    `

	var publisher model.Publisher
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&publisher.ID,
		&publisher.Name,
		&publisher.Slug,
		&publisher.Description,
		&publisher.Website,
		&publisher.CreatedAt,
		&publisher.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("publisher not found")
		}
		return nil, fmt.Errorf("failed to find publisher: %w", err)
	}

	return &publisher, nil
}

// FindBySlugWithTx tìm publisher by slug (trong transaction)
func (r *postgresRepository) FindBySlugWithTx(ctx context.Context, tx pgx.Tx, slug string) (*model.Publisher, error) {
	query := `
        SELECT id, name, slug, description, website,
               created_at, updated_at
        FROM publishers
        WHERE slug = $1
        LIMIT 1
    `

	var publisher model.Publisher
	err := tx.QueryRow(ctx, query, slug).Scan(
		&publisher.ID,
		&publisher.Name,
		&publisher.Slug,
		&publisher.Description,
		&publisher.Website,
		&publisher.CreatedAt,
		&publisher.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find publisher: %w", err)
	}

	return &publisher, nil
}

// CreateWithTx tạo publisher trong transaction
func (r *postgresRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, publisher *model.Publisher) error {
	query := `
        INSERT INTO publishers (id, name, slug, description, website, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	now := time.Now()
	publisher.CreatedAt = now
	publisher.UpdatedAt = now

	_, err := tx.Exec(ctx, query,
		publisher.ID,
		publisher.Name,
		publisher.Slug,
		publisher.Description,
		publisher.Website,
		publisher.CreatedAt,
		publisher.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	return nil
}
