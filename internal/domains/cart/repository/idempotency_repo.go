// internal/cart/repository/idempotency_repository.go

package repository

import (
	"bookstore-backend/internal/domains/cart/model"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, userID uuid.UUID) (*model.IdempotencyRecord, error)
	Create(ctx context.Context, record *model.IdempotencyRecord) error
	UpdateCompleted(ctx context.Context, key string, orderID uuid.UUID, response interface{}) error
	UpdateFailed(ctx context.Context, key string, errorMsg string) error
	CleanupExpired(ctx context.Context) (int64, error)
}

type idempotencyRepository struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepository(db *pgxpool.Pool) IdempotencyRepository {
	return &idempotencyRepository{db: db}
}

func (r *idempotencyRepository) Get(ctx context.Context, key string, userID uuid.UUID) (*model.IdempotencyRecord, error) {
	query := `
        SELECT 
            idempotency_key, user_id, cart_id, order_id, status, 
            request_payload, response_data, error_message,
            created_at, updated_at, completed_at, expires_at
        FROM checkout_idempotency
        WHERE idempotency_key = $1 AND user_id = $2
    `

	var record model.IdempotencyRecord
	var requestPayload, responseData []byte

	err := r.db.QueryRow(ctx, query, key, userID).Scan(
		&record.IdempotencyKey,
		&record.UserID,
		&record.CartID,
		&record.OrderID,
		&record.Status,
		&requestPayload,
		&responseData,
		&record.ErrorMessage,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CompletedAt,
		&record.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(requestPayload) > 0 {
		json.Unmarshal(requestPayload, &record.RequestPayload)
	}
	if len(responseData) > 0 {
		json.Unmarshal(responseData, &record.ResponseData)
	}

	return &record, nil
}

func (r *idempotencyRepository) Create(ctx context.Context, record *model.IdempotencyRecord) error {
	requestPayload, _ := json.Marshal(record.RequestPayload)

	query := `
        INSERT INTO checkout_idempotency 
        (idempotency_key, user_id, cart_id, status, request_payload, created_at, updated_at, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (idempotency_key) DO NOTHING
    `

	_, err := r.db.Exec(
		ctx,
		query,
		record.IdempotencyKey,
		record.UserID,
		record.CartID,
		record.Status,
		requestPayload,
		record.CreatedAt,
		record.UpdatedAt,
		record.ExpiresAt,
	)

	return err
}

func (r *idempotencyRepository) UpdateCompleted(ctx context.Context, key string, orderID uuid.UUID, response interface{}) error {
	responseData, _ := json.Marshal(response)

	query := `
        UPDATE checkout_idempotency
        SET 
            status = 'completed',
            order_id = $2,
            response_data = $3,
            completed_at = $4,
            updated_at = $4
        WHERE idempotency_key = $1
    `

	now := time.Now()
	_, err := r.db.Exec(ctx, query, key, orderID, responseData, now)
	return err
}

func (r *idempotencyRepository) UpdateFailed(ctx context.Context, key string, errorMsg string) error {
	query := `
        UPDATE checkout_idempotency
        SET 
            status = 'failed',
            error_message = $2,
            completed_at = $3,
            updated_at = $3
        WHERE idempotency_key = $1
    `

	now := time.Now()
	_, err := r.db.Exec(ctx, query, key, errorMsg, now)
	return err
}

func (r *idempotencyRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `
        DELETE FROM checkout_idempotency
        WHERE expires_at < NOW()
    `

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
