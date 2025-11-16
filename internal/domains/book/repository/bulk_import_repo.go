package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/book/model"
)

// BulkImportRepository interface cho tracking bulk import jobs
type BulkImportRepoI interface {
	CreateJob(ctx context.Context, job *model.BulkImportJob) error
	UpdateJobStatus(ctx context.Context, jobID, status string) error
	UpdateJobProgress(ctx context.Context, jobID string, processed, success, failed int) error
	UpdateJobErrors(ctx context.Context, jobID string, errors []model.ImportValidationError) error
	GetJobByID(ctx context.Context, jobID string) (*model.BulkImportJob, error)
	ListJobsByUser(ctx context.Context, userID string, limit, offset int) ([]*model.BulkImportJob, error)
}

type bulkImportRepository struct {
	pool *pgxpool.Pool
}

// NewBulkImportRepository tạo repository instance
func NewBulkImportRepository(pool *pgxpool.Pool) BulkImportRepoI {
	return &bulkImportRepository{pool: pool}
}

// CreateJob tạo một bulk import job mới
func (r *bulkImportRepository) CreateJob(ctx context.Context, job *model.BulkImportJob) error {
	query := `
        INSERT INTO bulk_import_jobs (
            id, user_id, file_name, file_url, file_size_bytes,
            total_rows, status, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		job.ID,
		job.UserID,
		job.FileName,
		job.FileURL,
		job.FileSizeBytes,
		job.TotalRows,
		job.Status,
		job.CreatedAt,
		job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create bulk import job: %w", err)
	}

	return nil
}

// UpdateJobStatus cập nhật status của job
func (r *bulkImportRepository) UpdateJobStatus(ctx context.Context, jobID, status string) error {
	query := `
        UPDATE bulk_import_jobs
        SET status = $1,
            updated_at = NOW(),
            started_at = CASE 
                WHEN $1 = 'processing' AND started_at IS NULL THEN NOW()
                ELSE started_at
            END,
            completed_at = CASE 
                WHEN $1 IN ('completed', 'failed') THEN NOW()
                ELSE completed_at
            END
        WHERE id = $2
    `

	_, err := r.pool.Exec(ctx, query, status, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// UpdateJobProgress cập nhật progress counters
func (r *bulkImportRepository) UpdateJobProgress(ctx context.Context, jobID string, processed, success, failed int) error {
	query := `
        UPDATE bulk_import_jobs
        SET processed_rows = $1,
            success_rows = $2,
            failed_rows = $3,
            updated_at = NOW()
        WHERE id = $4
    `

	_, err := r.pool.Exec(ctx, query, processed, success, failed, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// UpdateJobErrors lưu errors vào JSONB
func (r *bulkImportRepository) UpdateJobErrors(ctx context.Context, jobID string, errors []model.ImportValidationError) error {
	errorsJSON, err := json.Marshal(errors)
	if err != nil {
		return fmt.Errorf("failed to marshal errors: %w", err)
	}

	query := `
        UPDATE bulk_import_jobs
        SET errors = $1,
            updated_at = NOW()
        WHERE id = $2
    `

	_, err = r.pool.Exec(ctx, query, errorsJSON, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job errors: %w", err)
	}

	return nil
}

// GetJobByID lấy job info by ID
func (r *bulkImportRepository) GetJobByID(ctx context.Context, jobID string) (*model.BulkImportJob, error) {
	query := `
        SELECT id, user_id, file_name, file_url, file_size_bytes,
               total_rows, processed_rows, success_rows, failed_rows,
               status, errors,
               started_at, completed_at, created_at, updated_at
        FROM bulk_import_jobs
        WHERE id = $1
    `

	var job model.BulkImportJob
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&job.ID,
		&job.UserID,
		&job.FileName,
		&job.FileURL,
		&job.FileSizeBytes,
		&job.TotalRows,
		&job.ProcessedRows,
		&job.SuccessRows,
		&job.FailedRows,
		&job.Status,
		&job.Errors,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// ListJobsByUser lấy danh sách jobs của user (pagination)
func (r *bulkImportRepository) ListJobsByUser(ctx context.Context, userID string, limit, offset int) ([]*model.BulkImportJob, error) {
	query := `
        SELECT id, user_id, file_name, file_url, file_size_bytes,
               total_rows, processed_rows, success_rows, failed_rows,
               status, errors,
               started_at, completed_at, created_at, updated_at
        FROM bulk_import_jobs
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.BulkImportJob
	for rows.Next() {
		var job model.BulkImportJob
		err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.FileName,
			&job.FileURL,
			&job.FileSizeBytes,
			&job.TotalRows,
			&job.ProcessedRows,
			&job.SuccessRows,
			&job.FailedRows,
			&job.Status,
			&job.Errors,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}
