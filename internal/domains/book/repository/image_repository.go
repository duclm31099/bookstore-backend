package repository

import (
	"bookstore-backend/internal/domains/book/model"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookImageRepository interface {
	Create(ctx context.Context, image *model.BookImage) error
	GetByID(ctx context.Context, id string) (*model.BookImage, error)
	GetByBookID(ctx context.Context, bookID string) ([]*model.BookImage, error)
	Update(ctx context.Context, image *model.BookImage) error
	UpdateVariants(ctx context.Context, id string, large, medium, thumbnail string) error
	UpdateStatus(ctx context.Context, id, status, errorMsg string) error
	Delete(ctx context.Context, id string) error
	DeleteByBookID(ctx context.Context, bookID string) error
	// NEW: Methods for bulk import
	CreateWithTx(ctx context.Context, tx pgx.Tx, image *model.BookImage) error
	CreateBatchWithTx(ctx context.Context, tx pgx.Tx, images []*model.BookImage) error
}

type bookImageRepository struct {
	pool *pgxpool.Pool
}

func NewBookImageRepository(pool *pgxpool.Pool) BookImageRepository {
	return &bookImageRepository{pool: pool}
}

// Create tạo mới một book image record
func (r *bookImageRepository) Create(ctx context.Context, image *model.BookImage) error {
	query := `
        INSERT INTO book_images (
            book_id, original_url, sort_order, is_cover, status,
            format, width, height, file_size_bytes
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at, updated_at
    `

	err := r.pool.QueryRow(
		ctx, query,
		image.BookID,
		image.OriginalURL,
		image.SortOrder,
		image.IsCover,
		image.Status,
		image.Format,
		image.Width,
		image.Height,
		image.FileSizeBytes,
	).Scan(&image.ID, &image.CreatedAt, &image.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create book image: %w", err)
	}

	return nil
}

// GetByID lấy một ảnh theo ID
func (r *bookImageRepository) GetByID(ctx context.Context, id string) (*model.BookImage, error) {
	query := `
        SELECT id, book_id, original_url, large_url, medium_url, thumbnail_url,
               sort_order, is_cover, status, error_message,
               format, width, height, file_size_bytes,
               created_at, updated_at
        FROM book_images
        WHERE id = $1
    `

	image := &model.BookImage{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&image.ID,
		&image.BookID,
		&image.OriginalURL,
		&image.LargeURL,
		&image.MediumURL,
		&image.ThumbnailURL,
		&image.SortOrder,
		&image.IsCover,
		&image.Status,
		&image.ErrorMessage,
		&image.Format,
		&image.Width,
		&image.Height,
		&image.FileSizeBytes,
		&image.CreatedAt,
		&image.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("image not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return image, nil
}

// GetByBookID lấy tất cả ảnh của một book, sắp xếp theo sort_order
func (r *bookImageRepository) GetByBookID(ctx context.Context, bookID string) ([]*model.BookImage, error) {
	query := `
        SELECT id, book_id, original_url, large_url, medium_url, thumbnail_url,
               sort_order, is_cover, status, error_message,
               format, width, height, file_size_bytes,
               created_at, updated_at
        FROM book_images
        WHERE book_id = $1
        ORDER BY sort_order ASC
    `

	rows, err := r.pool.Query(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to query images: %w", err)
	}
	defer rows.Close()

	var images []*model.BookImage
	for rows.Next() {
		image := &model.BookImage{}
		err := rows.Scan(
			&image.ID,
			&image.BookID,
			&image.OriginalURL,
			&image.LargeURL,
			&image.MediumURL,
			&image.ThumbnailURL,
			&image.SortOrder,
			&image.IsCover,
			&image.Status,
			&image.ErrorMessage,
			&image.Format,
			&image.Width,
			&image.Height,
			&image.FileSizeBytes,
			&image.CreatedAt,
			&image.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, image)
	}

	return images, nil
}

// Update cập nhật thông tin ảnh
func (r *bookImageRepository) Update(ctx context.Context, image *model.BookImage) error {
	query := `
        UPDATE book_images SET
            large_url = $1,
            medium_url = $2,
            thumbnail_url = $3,
            status = $4,
            error_message = $5,
            updated_at = NOW()
        WHERE id = $6
    `

	_, err := r.pool.Exec(
		ctx, query,
		image.LargeURL,
		image.MediumURL,
		image.ThumbnailURL,
		image.Status,
		image.ErrorMessage,
		image.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}

	return nil
}

// UpdateVariants cập nhật các variant URLs sau khi worker xử lý xong
func (r *bookImageRepository) UpdateVariants(ctx context.Context, id string, large, medium, thumbnail string) error {
	query := `
        UPDATE book_images SET
            large_url = $1,
            medium_url = $2,
            thumbnail_url = $3,
            status = $4,
            updated_at = NOW()
        WHERE id = $5
    `

	_, err := r.pool.Exec(
		ctx, query,
		large,
		medium,
		thumbnail,
		model.ImageStatusReady, // Đánh dấu ready sau khi có đủ variants
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update variants: %w", err)
	}

	return nil
}

// UpdateStatus cập nhật status của ảnh (dùng khi processing hoặc failed)
func (r *bookImageRepository) UpdateStatus(ctx context.Context, id, status, errorMsg string) error {
	query := `
        UPDATE book_images SET
            status = $1,
            error_message = $2,
            updated_at = NOW()
        WHERE id = $3
    `

	var errMsgPtr *string
	if errorMsg != "" {
		errMsgPtr = &errorMsg
	}

	_, err := r.pool.Exec(ctx, query, status, errMsgPtr, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// Delete xóa một ảnh
func (r *bookImageRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM book_images WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}

// DeleteByBookID xóa tất cả ảnh của một book (dùng khi xóa book)
func (r *bookImageRepository) DeleteByBookID(ctx context.Context, bookID string) error {
	query := `DELETE FROM book_images WHERE book_id = $1`
	_, err := r.pool.Exec(ctx, query, bookID)
	if err != nil {
		return fmt.Errorf("failed to delete images: %w", err)
	}
	return nil
}

// CreateWithTx tạo book_image trong transaction
func (r *bookImageRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, image *model.BookImage) error {
	query := `
        INSERT INTO book_images (
            id, book_id, original_url, sort_order, is_cover,
            status, format, file_size_bytes, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `

	now := time.Now()
	image.CreatedAt = now
	image.UpdatedAt = now

	_, err := tx.Exec(ctx, query,
		image.ID,
		image.BookID,
		image.OriginalURL,
		image.SortOrder,
		image.IsCover,
		image.Status,
		image.Format,
		image.FileSizeBytes,
		image.CreatedAt,
		image.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create book_image: %w", err)
	}

	return nil
}

// CreateBatchWithTx tạo nhiều book_images cùng lúc (efficient)
func (r *bookImageRepository) CreateBatchWithTx(ctx context.Context, tx pgx.Tx, images []*model.BookImage) error {
	if len(images) == 0 {
		return nil
	}

	// Build batch insert query
	query := `
        INSERT INTO book_images (
            id, book_id, original_url, sort_order, is_cover,
            status, format, file_size_bytes, created_at, updated_at
        ) VALUES
    `

	// Prepare values
	values := make([]interface{}, 0, len(images)*10)
	placeholders := make([]string, 0, len(images))

	now := time.Now()
	for i, image := range images {
		image.CreatedAt = now
		image.UpdatedAt = now

		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*10+1, i*10+2, i*10+3, i*10+4, i*10+5,
			i*10+6, i*10+7, i*10+8, i*10+9, i*10+10,
		))

		values = append(values,
			image.ID,
			image.BookID,
			image.OriginalURL,
			image.SortOrder,
			image.IsCover,
			image.Status,
			image.Format,
			image.FileSizeBytes,
			image.CreatedAt,
			image.UpdatedAt,
		)
	}

	query += strings.Join(placeholders, ", ")

	_, err := tx.Exec(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to batch create book_images: %w", err)
	}

	return nil
}
