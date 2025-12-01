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
// TEMPLATE REPOSITORY IMPLEMENTATION
// ================================================

type templateRepository struct {
	db *pgxpool.Pool
}

func NewTemplateRepository(db *pgxpool.Pool) TemplateRepository {
	return &templateRepository{db: db}
}

// Create creates a new notification template
func (r *templateRepository) Create(ctx context.Context, template *model.NotificationTemplate) error {
	query := `
		INSERT INTO notification_templates (
			id, code, name, description, category,
			email_subject, email_body_html, email_body_text,
			sms_body, push_title, push_body,
			in_app_title, in_app_body, in_app_action_url,
			required_variables, language, default_channels,
			default_priority, expires_after_hours, version, is_active,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		RETURNING created_at, updated_at
	`

	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		template.ID, template.Code, template.Name, template.Description, template.Category,
		template.EmailSubject, template.EmailBodyHTML, template.EmailBodyText,
		template.SMSBody, template.PushTitle, template.PushBody,
		template.InAppTitle, template.InAppBody, template.InAppActionURL,
		template.RequiredVariables, template.Language, template.DefaultChannels,
		template.DefaultPriority, template.ExpiresAfterHours, template.Version, template.IsActive,
		template.CreatedBy, template.UpdatedBy,
	).Scan(&template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}

	return nil
}

// GetByID retrieves template by ID
func (r *templateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationTemplate, error) {
	query := `
		SELECT 
			id, code, name, description, category,
			email_subject, email_body_html, email_body_text,
			sms_body, push_title, push_body,
			in_app_title, in_app_body, in_app_action_url,
			required_variables, language, default_channels,
			default_priority, expires_after_hours, version, is_active,
			created_by, updated_by, created_at, updated_at
		FROM notification_templates
		WHERE id = $1
	`

	var t model.NotificationTemplate
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Code, &t.Name, &t.Description, &t.Category,
		&t.EmailSubject, &t.EmailBodyHTML, &t.EmailBodyText,
		&t.SMSBody, &t.PushTitle, &t.PushBody,
		&t.InAppTitle, &t.InAppBody, &t.InAppActionURL,
		&t.RequiredVariables, &t.Language, &t.DefaultChannels,
		&t.DefaultPriority, &t.ExpiresAfterHours, &t.Version, &t.IsActive,
		&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get template by id: %w", err)
	}

	return &t, nil
}

// GetByCode retrieves template by code
func (r *templateRepository) GetByCode(ctx context.Context, code string) (*model.NotificationTemplate, error) {
	query := `
		SELECT 
			id, code, name, description, category,
			email_subject, email_body_html, email_body_text,
			sms_body, push_title, push_body,
			in_app_title, in_app_body, in_app_action_url,
			required_variables, language, default_channels,
			default_priority, expires_after_hours, version, is_active,
			created_by, updated_by, created_at, updated_at
		FROM notification_templates
		WHERE code = $1
	`

	var t model.NotificationTemplate
	err := r.db.QueryRow(ctx, query, code).Scan(
		&t.ID, &t.Code, &t.Name, &t.Description, &t.Category,
		&t.EmailSubject, &t.EmailBodyHTML, &t.EmailBodyText,
		&t.SMSBody, &t.PushTitle, &t.PushBody,
		&t.InAppTitle, &t.InAppBody, &t.InAppActionURL,
		&t.RequiredVariables, &t.Language, &t.DefaultChannels,
		&t.DefaultPriority, &t.ExpiresAfterHours, &t.Version, &t.IsActive,
		&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get template by code: %w", err)
	}

	return &t, nil
}

// Update updates notification template
func (r *templateRepository) Update(ctx context.Context, template *model.NotificationTemplate) error {
	query := `
		UPDATE notification_templates
		SET 
			name = $2,
			description = $3,
			category = $4,
			email_subject = $5,
			email_body_html = $6,
			email_body_text = $7,
			sms_body = $8,
			push_title = $9,
			push_body = $10,
			in_app_title = $11,
			in_app_body = $12,
			in_app_action_url = $13,
			required_variables = $14,
			language = $15,
			default_channels = $16,
			default_priority = $17,
			expires_after_hours = $18,
			is_active = $19,
			updated_by = $20,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		template.ID, template.Name, template.Description, template.Category,
		template.EmailSubject, template.EmailBodyHTML, template.EmailBodyText,
		template.SMSBody, template.PushTitle, template.PushBody,
		template.InAppTitle, template.InAppBody, template.InAppActionURL,
		template.RequiredVariables, template.Language, template.DefaultChannels,
		template.DefaultPriority, template.ExpiresAfterHours, template.IsActive,
		template.UpdatedBy,
	).Scan(&template.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrTemplateNotFound
		}
		return fmt.Errorf("update template: %w", err)
	}

	return nil
}

// Delete deletes notification template
func (r *templateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notification_templates WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrTemplateNotFound
	}

	return nil
}

// List retrieves templates with filters and pagination
func (r *templateRepository) List(ctx context.Context, category *string, isActive *bool, limit, offset int) ([]model.NotificationTemplate, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if category != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, *category)
	}

	if isActive != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *isActive)
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM notification_templates " + whereClause
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count templates: %w", err)
	}

	// Query templates
	query := fmt.Sprintf(`
		SELECT 
			id, code, name, description, category,
			email_subject, email_body_html, email_body_text,
			sms_body, push_title, push_body,
			in_app_title, in_app_body, in_app_action_url,
			required_variables, language, default_channels,
			default_priority, expires_after_hours, version, is_active,
			created_by, updated_by, created_at, updated_at
		FROM notification_templates
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount+1, argCount+2)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []model.NotificationTemplate
	for rows.Next() {
		var t model.NotificationTemplate
		err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description, &t.Category,
			&t.EmailSubject, &t.EmailBodyHTML, &t.EmailBodyText,
			&t.SMSBody, &t.PushTitle, &t.PushBody,
			&t.InAppTitle, &t.InAppBody, &t.InAppActionURL,
			&t.RequiredVariables, &t.Language, &t.DefaultChannels,
			&t.DefaultPriority, &t.ExpiresAfterHours, &t.Version, &t.IsActive,
			&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return templates, total, nil
}

// ListActive retrieves all active templates
func (r *templateRepository) ListActive(ctx context.Context) ([]model.NotificationTemplate, error) {
	query := `
		SELECT 
			id, code, name, description, category,
			email_subject, email_body_html, email_body_text,
			sms_body, push_title, push_body,
			in_app_title, in_app_body, in_app_action_url,
			required_variables, language, default_channels,
			default_priority, expires_after_hours, version, is_active,
			created_by, updated_by, created_at, updated_at
		FROM notification_templates
		WHERE is_active = TRUE
		ORDER BY code ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active templates: %w", err)
	}
	defer rows.Close()

	var templates []model.NotificationTemplate
	for rows.Next() {
		var t model.NotificationTemplate
		err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description, &t.Category,
			&t.EmailSubject, &t.EmailBodyHTML, &t.EmailBodyText,
			&t.SMSBody, &t.PushTitle, &t.PushBody,
			&t.InAppTitle, &t.InAppBody, &t.InAppActionURL,
			&t.RequiredVariables, &t.Language, &t.DefaultChannels,
			&t.DefaultPriority, &t.ExpiresAfterHours, &t.Version, &t.IsActive,
			&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return templates, nil
}

// IncrementVersion increments template version
func (r *templateRepository) IncrementVersion(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notification_templates
		SET version = version + 1, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("increment version: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrTemplateNotFound
	}

	return nil
}

// GetLatestVersion retrieves latest version of template by code
func (r *templateRepository) GetLatestVersion(ctx context.Context, code string) (*model.NotificationTemplate, error) {
	query := `
		SELECT 
			id, code, name, description, category,
			email_subject, email_body_html, email_body_text,
			sms_body, push_title, push_body,
			in_app_title, in_app_body, in_app_action_url,
			required_variables, language, default_channels,
			default_priority, expires_after_hours, version, is_active,
			created_by, updated_by, created_at, updated_at
		FROM notification_templates
		WHERE code = $1
		ORDER BY version DESC
		LIMIT 1
	`

	var t model.NotificationTemplate
	err := r.db.QueryRow(ctx, query, code).Scan(
		&t.ID, &t.Code, &t.Name, &t.Description, &t.Category,
		&t.EmailSubject, &t.EmailBodyHTML, &t.EmailBodyText,
		&t.SMSBody, &t.PushTitle, &t.PushBody,
		&t.InAppTitle, &t.InAppBody, &t.InAppActionURL,
		&t.RequiredVariables, &t.Language, &t.DefaultChannels,
		&t.DefaultPriority, &t.ExpiresAfterHours, &t.Version, &t.IsActive,
		&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get latest version: %w", err)
	}

	return &t, nil
}
