package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bookstore-backend/internal/domains/notification/model"
)

// ================================================
// CAMPAIGN REPOSITORY IMPLEMENTATION
// ================================================

type campaignRepository struct {
	db *pgxpool.Pool
}

func NewCampaignRepository(db *pgxpool.Pool) CampaignRepository {
	return &campaignRepository{db: db}
}

// Create creates a new campaign
func (r *campaignRepository) Create(ctx context.Context, campaign *model.Campaign) error {
	query := `
		INSERT INTO notification_campaigns (
			id, name, description, template_code,
			target_type, target_segment, target_user_ids, target_filters,
			scheduled_at, status, batch_size, batch_delay_seconds,
			total_recipients, processed_count, sent_count, delivered_count, failed_count,
			template_data, channels, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
		RETURNING created_at, updated_at
	`

	if campaign.ID == uuid.Nil {
		campaign.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		campaign.ID, campaign.Name, campaign.Description, campaign.TemplateCode,
		campaign.TargetType, campaign.TargetSegment, campaign.TargetUserIDs, campaign.TargetFilters,
		campaign.ScheduledAt, campaign.Status, campaign.BatchSize, campaign.BatchDelaySeconds,
		campaign.TotalRecipients, campaign.ProcessedCount, campaign.SentCount,
		campaign.DeliveredCount, campaign.FailedCount,
		campaign.TemplateData, campaign.Channels, campaign.CreatedBy,
	).Scan(&campaign.CreatedAt, &campaign.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}

	return nil
}

// GetByID retrieves campaign by ID
func (r *campaignRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Campaign, error) {
	query := `
		SELECT 
			id, name, description, template_code,
			target_type, target_segment, target_user_ids, target_filters,
			scheduled_at, started_at, completed_at, cancelled_at,
			status, batch_size, batch_delay_seconds,
			total_recipients, processed_count, sent_count, delivered_count, failed_count,
			template_data, channels, created_by, created_at, updated_at
		FROM notification_campaigns
		WHERE id = $1
	`
	var targetFilterByte, templateData []byte

	var c model.Campaign
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Description, &c.TemplateCode,
		&c.TargetType, &c.TargetSegment, &c.TargetUserIDs, &targetFilterByte,
		&c.ScheduledAt, &c.StartedAt, &c.CompletedAt, &c.CancelledAt,
		&c.Status, &c.BatchSize, &c.BatchDelaySeconds,
		&c.TotalRecipients, &c.ProcessedCount, &c.SentCount, &c.DeliveredCount, &c.FailedCount,
		&templateData, &c.Channels, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCampaignNotFound
		}
		return nil, fmt.Errorf("get campaign by id: %w", err)
	}
	if len(targetFilterByte) > 0 {
		if err := json.Unmarshal(targetFilterByte, &c.TargetFilters); err != nil {
			return nil, fmt.Errorf("get campaign by id: %w", err)
		}
	}
	if len(templateData) > 0 {
		if err := json.Unmarshal(templateData, &c.TemplateData); err != nil {
			return nil, fmt.Errorf("get campaign by id: %w", err)
		}
	}
	return &c, nil
}

// Update updates campaign
func (r *campaignRepository) Update(ctx context.Context, campaign *model.Campaign) error {
	query := `
		UPDATE notification_campaigns
		SET 
			name = $2,
			description = $3,
			template_code = $4,
			target_type = $5,
			target_segment = $6,
			target_user_ids = $7,
			target_filters = $8,
			scheduled_at = $9,
			status = $10,
			batch_size = $11,
			batch_delay_seconds = $12,
			template_data = $13,
			channels = $14,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		campaign.ID, campaign.Name, campaign.Description, campaign.TemplateCode,
		campaign.TargetType, campaign.TargetSegment, campaign.TargetUserIDs, campaign.TargetFilters,
		campaign.ScheduledAt, campaign.Status, campaign.BatchSize, campaign.BatchDelaySeconds,
		campaign.TemplateData, campaign.Channels,
	).Scan(&campaign.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrCampaignNotFound
		}
		return fmt.Errorf("update campaign: %w", err)
	}

	return nil
}

// Delete deletes campaign
func (r *campaignRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notification_campaigns WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrCampaignNotFound
	}

	return nil
}

// List retrieves campaigns with filters and pagination
func (r *campaignRepository) List(ctx context.Context, status *string, createdBy *uuid.UUID, limit, offset int) ([]model.Campaign, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if status != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *status)
	}

	if createdBy != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND created_by = $%d", argCount)
		args = append(args, *createdBy)
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM notification_campaigns " + whereClause
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count campaigns: %w", err)
	}

	// Query campaigns
	query := fmt.Sprintf(`
		SELECT 
			id, name, description, template_code,
			target_type, target_segment, target_user_ids, target_filters,
			scheduled_at, started_at, completed_at, cancelled_at,
			status, batch_size, batch_delay_seconds,
			total_recipients, processed_count, sent_count, delivered_count, failed_count,
			template_data, channels, created_by, created_at, updated_at
		FROM notification_campaigns
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount+1, argCount+2)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()
	var targetFilterBytes, templateDataBytes []byte
	var campaigns []model.Campaign
	for rows.Next() {
		var c model.Campaign
		err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.TemplateCode,
			&c.TargetType, &c.TargetSegment, &c.TargetUserIDs, &targetFilterBytes,
			&c.ScheduledAt, &c.StartedAt, &c.CompletedAt, &c.CancelledAt,
			&c.Status, &c.BatchSize, &c.BatchDelaySeconds,
			&c.TotalRecipients, &c.ProcessedCount, &c.SentCount, &c.DeliveredCount, &c.FailedCount,
			&templateDataBytes, &c.Channels, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan campaign: %w", err)
		}
		if targetFilterBytes != nil && string(targetFilterBytes) != "" {
			if err := json.Unmarshal(targetFilterBytes, &c.TargetFilters); err != nil {
				return nil, 0, fmt.Errorf("unmarshal target filters: %w", err)
			}
		}

		if err := json.Unmarshal(templateDataBytes, &c.TemplateData); err != nil {
			return nil, 0, fmt.Errorf("unmarshal template data: %w", err)
		}
		campaigns = append(campaigns, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return campaigns, total, nil
}

// ListScheduled retrieves campaigns scheduled before a specific time
func (r *campaignRepository) ListScheduled(ctx context.Context, before time.Time) ([]model.Campaign, error) {
	query := `
		SELECT 
			id, name, description, template_code,
			target_type, target_segment, target_user_ids, target_filters,
			scheduled_at, started_at, completed_at, cancelled_at,
			status, batch_size, batch_delay_seconds,
			total_recipients, processed_count, sent_count, delivered_count, failed_count,
			template_data, channels, created_by, created_at, updated_at
		FROM notification_campaigns
		WHERE status = $1
		AND scheduled_at IS NOT NULL
		AND scheduled_at <= $2
		ORDER BY scheduled_at ASC
	`

	rows, err := r.db.Query(ctx, query, model.CampaignStatusScheduled, before)
	if err != nil {
		return nil, fmt.Errorf("list scheduled campaigns: %w", err)
	}
	defer rows.Close()
	var targetFilterBytes, templateDataBytes []byte
	var campaigns []model.Campaign
	for rows.Next() {
		var c model.Campaign
		err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.TemplateCode,
			&c.TargetType, &c.TargetSegment, &c.TargetUserIDs, &targetFilterBytes,
			&c.ScheduledAt, &c.StartedAt, &c.CompletedAt, &c.CancelledAt,
			&c.Status, &c.BatchSize, &c.BatchDelaySeconds,
			&c.TotalRecipients, &c.ProcessedCount, &c.SentCount, &c.DeliveredCount, &c.FailedCount,
			&templateDataBytes, &c.Channels, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan campaign: %w", err)
		}
		if targetFilterBytes != nil && string(targetFilterBytes) != "" {
			if err := json.Unmarshal(targetFilterBytes, &c.TargetFilters); err != nil {
				return nil, fmt.Errorf("unmarshal target filters: %w", err)
			}
		}
		if templateDataBytes != nil && string(templateDataBytes) != "" {
			if err := json.Unmarshal(templateDataBytes, &c.TemplateData); err != nil {
				return nil, fmt.Errorf("unmarshal template data: %w", err)
			}
		}
		campaigns = append(campaigns, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return campaigns, nil
}

// UpdateStatus updates campaign status
func (r *campaignRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE notification_campaigns
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("update campaign status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrCampaignNotFound
	}

	return nil
}

// UpdateProgress updates campaign progress counters
func (r *campaignRepository) UpdateProgress(ctx context.Context, id uuid.UUID, sent, delivered, failed int) error {
	query := `
		UPDATE notification_campaigns
		SET 
			processed_count = processed_count + $2,
			sent_count = sent_count + $3,
			delivered_count = delivered_count + $4,
			failed_count = failed_count + $5,
			updated_at = NOW()
		WHERE id = $1
	`

	total := sent + failed
	result, err := r.db.Exec(ctx, query, id, total, sent, delivered, failed)
	if err != nil {
		return fmt.Errorf("update campaign progress: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrCampaignNotFound
	}

	return nil
}

// MarkAsStarted marks campaign as started
func (r *campaignRepository) MarkAsStarted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notification_campaigns
		SET 
			status = $2,
			started_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, model.CampaignStatusRunning)
	if err != nil {
		return fmt.Errorf("mark campaign as started: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrCampaignNotFound
	}

	return nil
}

// MarkAsCompleted marks campaign as completed
func (r *campaignRepository) MarkAsCompleted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notification_campaigns
		SET 
			status = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, model.CampaignStatusCompleted)
	if err != nil {
		return fmt.Errorf("mark campaign as completed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrCampaignNotFound
	}

	return nil
}

// MarkAsCancelled marks campaign as cancelled
func (r *campaignRepository) MarkAsCancelled(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notification_campaigns
		SET 
			status = $2,
			cancelled_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, model.CampaignStatusCancelled)
	if err != nil {
		return fmt.Errorf("mark campaign as cancelled: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrCampaignNotFound
	}

	return nil
}
