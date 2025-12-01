package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// RATE LIMIT REPOSITORY IMPLEMENTATION
// ================================================

type rateLimitRepository struct {
	db *pgxpool.Pool
}

func NewRateLimitRepository(db *pgxpool.Pool) RateLimitRepository {
	return &rateLimitRepository{db: db}
}

// Create creates a new rate limit record
func (r *rateLimitRepository) Create(ctx context.Context, rateLimit *model.RateLimit) error {
	query := `
		INSERT INTO notification_rate_limits (
			id, scope, scope_id, max_notifications, window_minutes,
			current_count, window_start
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING created_at, updated_at
	`

	if rateLimit.ID == uuid.Nil {
		rateLimit.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		rateLimit.ID,
		rateLimit.Scope,
		rateLimit.ScopeID,
		rateLimit.MaxNotifications,
		rateLimit.WindowMinutes,
		rateLimit.CurrentCount,
		rateLimit.WindowStart,
	).Scan(&rateLimit.CreatedAt, &rateLimit.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create rate limit: %w", err)
	}

	return nil
}

// GetByScope retrieves rate limit by scope
func (r *rateLimitRepository) GetByScope(ctx context.Context, scope, scopeID string, windowMinutes int) (*model.RateLimit, error) {
	query := `
		SELECT 
			id, scope, scope_id, max_notifications, window_minutes,
			current_count, window_start, created_at, updated_at
		FROM notification_rate_limits
		WHERE scope = $1 
		AND scope_id = $2 
		AND window_minutes = $3
	`

	var rl model.RateLimit
	err := r.db.QueryRow(ctx, query, scope, scopeID, windowMinutes).Scan(
		&rl.ID,
		&rl.Scope,
		&rl.ScopeID,
		&rl.MaxNotifications,
		&rl.WindowMinutes,
		&rl.CurrentCount,
		&rl.WindowStart,
		&rl.CreatedAt,
		&rl.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found is not an error for rate limits
		}
		return nil, fmt.Errorf("get rate limit by scope: %w", err)
	}

	return &rl, nil
}

// Update updates rate limit
func (r *rateLimitRepository) Update(ctx context.Context, rateLimit *model.RateLimit) error {
	query := `
		UPDATE notification_rate_limits
		SET 
			max_notifications = $2,
			current_count = $3,
			window_start = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		rateLimit.ID,
		rateLimit.MaxNotifications,
		rateLimit.CurrentCount,
		rateLimit.WindowStart,
	).Scan(&rateLimit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("rate limit not found")
		}
		return fmt.Errorf("update rate limit: %w", err)
	}

	return nil
}

// Upsert creates or updates rate limit
func (r *rateLimitRepository) Upsert(ctx context.Context, rateLimit *model.RateLimit) error {
	query := `
		INSERT INTO notification_rate_limits (
			id, scope, scope_id, max_notifications, window_minutes,
			current_count, window_start
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (scope, scope_id, window_minutes)
		DO UPDATE SET
			max_notifications = EXCLUDED.max_notifications,
			current_count = EXCLUDED.current_count,
			window_start = EXCLUDED.window_start,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`

	if rateLimit.ID == uuid.Nil {
		rateLimit.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		rateLimit.ID,
		rateLimit.Scope,
		rateLimit.ScopeID,
		rateLimit.MaxNotifications,
		rateLimit.WindowMinutes,
		rateLimit.CurrentCount,
		rateLimit.WindowStart,
	).Scan(&rateLimit.CreatedAt, &rateLimit.UpdatedAt)

	if err != nil {
		return fmt.Errorf("upsert rate limit: %w", err)
	}

	return nil
}

// CheckLimit checks if request is within rate limit
func (r *rateLimitRepository) CheckLimit(ctx context.Context, scope, scopeID string, maxNotifications, windowMinutes int) (bool, error) {
	// Use database function for atomic check
	query := `SELECT check_rate_limit($1, $2, $3, $4)`

	var allowed bool
	err := r.db.QueryRow(ctx, query, scope, scopeID, maxNotifications, windowMinutes).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check rate limit: %w", err)
	}

	return allowed, nil
}

// IncrementCount increments the counter for rate limit
func (r *rateLimitRepository) IncrementCount(ctx context.Context, scope, scopeID string, windowMinutes int) error {
	// Use database function for atomic increment
	query := `SELECT increment_rate_limit($1, $2, $3)`

	_, err := r.db.Exec(ctx, query, scope, scopeID, windowMinutes)
	if err != nil {
		return fmt.Errorf("increment rate limit: %w", err)
	}

	return nil
}

// ResetExpiredWindows resets rate limits where window has expired
func (r *rateLimitRepository) ResetExpiredWindows(ctx context.Context) (int, error) {
	query := `
		UPDATE notification_rate_limits
		SET 
			current_count = 0,
			window_start = NOW(),
			updated_at = NOW()
		WHERE window_start + (window_minutes || ' minutes')::INTERVAL < NOW()
		AND current_count > 0
	`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("reset expired windows: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// ResetByScope resets rate limit for specific scope
func (r *rateLimitRepository) ResetByScope(ctx context.Context, scope, scopeID string) error {
	query := `
		UPDATE notification_rate_limits
		SET 
			current_count = 0,
			window_start = NOW(),
			updated_at = NOW()
		WHERE scope = $1 AND scope_id = $2
	`

	result, err := r.db.Exec(ctx, query, scope, scopeID)
	if err != nil {
		return fmt.Errorf("reset by scope: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("rate limit not found")
	}

	return nil
}
