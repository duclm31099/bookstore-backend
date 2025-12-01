package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// PREFERENCES REPOSITORY IMPLEMENTATION
// ================================================

type preferencesRepository struct {
	db *pgxpool.Pool
}

func NewPreferencesRepository(db *pgxpool.Pool) PreferencesRepository {
	return &preferencesRepository{db: db}
}

// Create creates new notification preferences
func (r *preferencesRepository) Create(ctx context.Context, prefs *model.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			id, user_id, preferences, do_not_disturb,
			quiet_hours_start, quiet_hours_end
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		RETURNING created_at, updated_at
	`

	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		prefs.ID,
		prefs.UserID,
		prefs.Preferences,
		prefs.DoNotDisturb,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
	).Scan(&prefs.CreatedAt, &prefs.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create preferences: %w", err)
	}

	return nil
}

// GetByUserID retrieves preferences by user ID
func (r *preferencesRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error) {
	query := `
		SELECT 
			id, user_id, preferences, do_not_disturb,
			quiet_hours_start, quiet_hours_end,
			created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`

	var prefs model.NotificationPreferences
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&prefs.ID,
		&prefs.UserID,
		&prefs.Preferences,
		&prefs.DoNotDisturb,
		&prefs.QuietHoursStart,
		&prefs.QuietHoursEnd,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPreferencesNotFound
		}
		return nil, fmt.Errorf("get preferences by user id: %w", err)
	}

	return &prefs, nil
}

// Update updates notification preferences
func (r *preferencesRepository) Update(ctx context.Context, prefs *model.NotificationPreferences) error {
	query := `
		UPDATE notification_preferences
		SET 
			preferences = $2,
			do_not_disturb = $3,
			quiet_hours_start = $4,
			quiet_hours_end = $5,
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		prefs.UserID,
		prefs.Preferences,
		prefs.DoNotDisturb,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
	).Scan(&prefs.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrPreferencesNotFound
		}
		return fmt.Errorf("update preferences: %w", err)
	}

	return nil
}

// Upsert creates or updates preferences
func (r *preferencesRepository) Upsert(ctx context.Context, prefs *model.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			id, user_id, preferences, do_not_disturb,
			quiet_hours_start, quiet_hours_end
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT (user_id) 
		DO UPDATE SET
			preferences = EXCLUDED.preferences,
			do_not_disturb = EXCLUDED.do_not_disturb,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`

	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		prefs.ID,
		prefs.UserID,
		prefs.Preferences,
		prefs.DoNotDisturb,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
	).Scan(&prefs.CreatedAt, &prefs.UpdatedAt)

	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}

	return nil
}

// IsChannelEnabled checks if channel is enabled for notification type
func (r *preferencesRepository) IsChannelEnabled(ctx context.Context, userID uuid.UUID, notificationType, channel string) (bool, error) {
	query := `
		SELECT 
			COALESCE(
				(preferences->$2->>$3)::boolean,
				true  -- Default to true if not set
			) as is_enabled
		FROM notification_preferences
		WHERE user_id = $1
	`

	var isEnabled bool
	err := r.db.QueryRow(ctx, query, userID, notificationType, channel).Scan(&isEnabled)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// If no preferences found, default to enabled
			return true, nil
		}
		return false, fmt.Errorf("check channel enabled: %w", err)
	}

	return isEnabled, nil
}

// IsInQuietHours checks if current time is in user's quiet hours
func (r *preferencesRepository) IsInQuietHours(ctx context.Context, userID uuid.UUID, checkTime time.Time) (bool, error) {
	query := `
		SELECT 
			quiet_hours_start,
			quiet_hours_end
		FROM notification_preferences
		WHERE user_id = $1
	`

	var start, end *time.Time
	err := r.db.QueryRow(ctx, query, userID).Scan(&start, &end)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No quiet hours set
			return false, nil
		}
		return false, fmt.Errorf("check quiet hours: %w", err)
	}

	// If quiet hours not set, not in quiet hours
	if start == nil || end == nil {
		return false, nil
	}

	// Extract time components
	checkHour := checkTime.Hour()
	checkMinute := checkTime.Minute()
	checkTimeInMinutes := checkHour*60 + checkMinute

	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()

	// Handle case where quiet hours span midnight
	if startMinutes > endMinutes {
		// Example: 22:00 - 08:00
		return checkTimeInMinutes >= startMinutes || checkTimeInMinutes < endMinutes, nil
	}

	// Normal case: 22:00 - 23:00
	return checkTimeInMinutes >= startMinutes && checkTimeInMinutes < endMinutes, nil
}

// IsDoNotDisturb checks if user has enabled do not disturb
func (r *preferencesRepository) IsDoNotDisturb(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT do_not_disturb
		FROM notification_preferences
		WHERE user_id = $1
	`

	var dnd bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&dnd)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Default to false if no preferences
			return false, nil
		}
		return false, fmt.Errorf("check do not disturb: %w", err)
	}

	return dnd, nil
}
