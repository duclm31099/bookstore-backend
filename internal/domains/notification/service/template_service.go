package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/repository"
)

// ================================================
// TEMPLATE SERVICE IMPLEMENTATION
// ================================================

type templateService struct {
	templateRepo repository.TemplateRepository
}

func NewTemplateService(templateRepo repository.TemplateRepository) TemplateService {
	return &templateService{
		templateRepo: templateRepo,
	}
}

// ================================================
// CREATE TEMPLATE (Admin)
// ================================================

func (s *templateService) CreateTemplate(ctx context.Context, adminID uuid.UUID, req model.CreateTemplateRequest) (*model.TemplateResponse, error) {
	log.Info().
		Str("admin_id", adminID.String()).
		Str("code", req.Code).
		Msg("[TemplateService] CreateTemplate")

	// 1. VALIDATE TEMPLATE CODE FORMAT (lowercase, underscores only)
	if !isValidTemplateCode(req.Code) {
		return nil, fmt.Errorf("invalid template code format: use lowercase letters, numbers, and underscores only")
	}

	// 2. CHECK IF CODE ALREADY EXISTS
	existing, err := s.templateRepo.GetByCode(ctx, req.Code)
	if err == nil && existing != nil {
		return nil, model.ErrTemplateCodeExists
	}

	// 3. VALIDATE AT LEAST ONE CHANNEL HAS CONTENT
	hasContent := false
	for _, channel := range req.DefaultChannels {
		switch channel {
		case model.ChannelEmail:
			if req.EmailSubject != nil && req.EmailBodyHTML != nil {
				hasContent = true
			}
		case model.ChannelSMS:
			if req.SMSBody != nil {
				hasContent = true
			}
		case model.ChannelPush:
			if req.PushTitle != nil && req.PushBody != nil {
				hasContent = true
			}
		case model.ChannelInApp:
			if req.InAppTitle != nil && req.InAppBody != nil {
				hasContent = true
			}
		}
	}

	if !hasContent {
		return nil, fmt.Errorf("template must have content for at least one default channel")
	}

	// 4. CREATE TEMPLATE
	template := &model.NotificationTemplate{
		Code:              req.Code,
		Name:              req.Name,
		Description:       req.Description,
		Category:          req.Category,
		EmailSubject:      req.EmailSubject,
		EmailBodyHTML:     req.EmailBodyHTML,
		EmailBodyText:     req.EmailBodyText,
		SMSBody:           req.SMSBody,
		PushTitle:         req.PushTitle,
		PushBody:          req.PushBody,
		InAppTitle:        req.InAppTitle,
		InAppBody:         req.InAppBody,
		InAppActionURL:    req.InAppActionURL,
		RequiredVariables: req.RequiredVariables,
		Language:          req.Language,
		DefaultChannels:   req.DefaultChannels,
		DefaultPriority:   req.DefaultPriority,
		ExpiresAfterHours: req.ExpiresAfterHours,
		Version:           1,
		IsActive:          true,
		CreatedBy:         &adminID,
		UpdatedBy:         &adminID,
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}

	log.Info().
		Str("template_id", template.ID.String()).
		Str("code", template.Code).
		Msg("[TemplateService] Template created successfully")

	return s.toResponse(template), nil
}

// ================================================
// GET TEMPLATE BY ID
// ================================================

func (s *templateService) GetTemplateByID(ctx context.Context, templateID uuid.UUID) (*model.TemplateResponse, error) {
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	return s.toResponse(template), nil
}

// ================================================
// GET TEMPLATE BY CODE
// ================================================

func (s *templateService) GetTemplateByCode(ctx context.Context, code string) (*model.TemplateResponse, error) {
	template, err := s.templateRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return s.toResponse(template), nil
}

// ================================================
// UPDATE TEMPLATE (Admin)
// ================================================

func (s *templateService) UpdateTemplate(ctx context.Context, adminID, templateID uuid.UUID, req model.UpdateTemplateRequest) (*model.TemplateResponse, error) {
	log.Info().
		Str("admin_id", adminID.String()).
		Str("template_id", templateID.String()).
		Msg("[TemplateService] UpdateTemplate")

	// 1. GET EXISTING TEMPLATE
	existing, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// 2. UPDATE FIELDS (only if provided in request)
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.EmailSubject != nil {
		existing.EmailSubject = req.EmailSubject
	}
	if req.EmailBodyHTML != nil {
		existing.EmailBodyHTML = req.EmailBodyHTML
	}
	if req.EmailBodyText != nil {
		existing.EmailBodyText = req.EmailBodyText
	}
	if req.SMSBody != nil {
		existing.SMSBody = req.SMSBody
	}
	if req.PushTitle != nil {
		existing.PushTitle = req.PushTitle
	}
	if req.PushBody != nil {
		existing.PushBody = req.PushBody
	}
	if req.InAppTitle != nil {
		existing.InAppTitle = req.InAppTitle
	}
	if req.InAppBody != nil {
		existing.InAppBody = req.InAppBody
	}
	if req.InAppActionURL != nil {
		existing.InAppActionURL = req.InAppActionURL
	}
	if req.RequiredVariables != nil {
		existing.RequiredVariables = req.RequiredVariables
	}
	if req.DefaultChannels != nil {
		existing.DefaultChannels = req.DefaultChannels
	}
	if req.DefaultPriority != nil {
		existing.DefaultPriority = *req.DefaultPriority
	}
	if req.ExpiresAfterHours != nil {
		existing.ExpiresAfterHours = req.ExpiresAfterHours
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	existing.UpdatedBy = &adminID

	// 3. INCREMENT VERSION (if content changed)
	if s.isContentChanged(req) {
		if err := s.templateRepo.IncrementVersion(ctx, templateID); err != nil {
			log.Warn().Err(err).Msg("Failed to increment version")
		}
	}

	// 4. UPDATE TEMPLATE
	if err := s.templateRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}

	log.Info().
		Str("template_id", templateID.String()).
		Msg("[TemplateService] Template updated successfully")

	return s.toResponse(existing), nil
}

// ================================================
// DELETE TEMPLATE (Admin)
// ================================================

func (s *templateService) DeleteTemplate(ctx context.Context, templateID uuid.UUID) error {
	log.Info().
		Str("template_id", templateID.String()).
		Msg("[TemplateService] DeleteTemplate")

	// Check if template exists
	if _, err := s.templateRepo.GetByID(ctx, templateID); err != nil {
		return err
	}

	// Delete template
	if err := s.templateRepo.Delete(ctx, templateID); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	log.Info().
		Str("template_id", templateID.String()).
		Msg("[TemplateService] Template deleted successfully")

	return nil
}

// ================================================
// LIST TEMPLATES (Admin)
// ================================================

func (s *templateService) ListTemplates(ctx context.Context, category *string, isActive *bool, page, pageSize int) ([]model.TemplateResponse, int64, error) {
	// Set defaults
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	templates, total, err := s.templateRepo.List(ctx, category, isActive, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}

	responses := make([]model.TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = *s.toResponse(&t)
	}

	return responses, total, nil
}

// ================================================
// RENDER TEMPLATE
// ================================================

func (s *templateService) RenderTemplate(ctx context.Context, templateCode, channel string, data map[string]interface{}) (string, string, error) {
	// 1. GET TEMPLATE
	template, err := s.templateRepo.GetByCode(ctx, templateCode)
	if err != nil {
		return "", "", fmt.Errorf("get template: %w", err)
	}

	if !template.IsActive {
		return "", "", model.ErrTemplateInactive
	}

	// 2. VALIDATE REQUIRED VARIABLES
	if err := s.validateVariables(template, data); err != nil {
		return "", "", err
	}

	// 3. RENDER BASED ON CHANNEL
	var title, body string

	switch channel {
	case model.ChannelEmail:
		if template.EmailSubject == nil || template.EmailBodyHTML == nil {
			return "", "", fmt.Errorf("email template not configured")
		}
		title = s.renderString(*template.EmailSubject, data)
		body = s.renderString(*template.EmailBodyHTML, data)

	case model.ChannelSMS:
		if template.SMSBody == nil {
			return "", "", fmt.Errorf("sms template not configured")
		}
		body = s.renderString(*template.SMSBody, data)

	case model.ChannelPush:
		if template.PushTitle == nil || template.PushBody == nil {
			return "", "", fmt.Errorf("push template not configured")
		}
		title = s.renderString(*template.PushTitle, data)
		body = s.renderString(*template.PushBody, data)

	case model.ChannelInApp:
		if template.InAppTitle == nil || template.InAppBody == nil {
			return "", "", fmt.Errorf("in-app template not configured")
		}
		title = s.renderString(*template.InAppTitle, data)
		body = s.renderString(*template.InAppBody, data)

	default:
		return "", "", fmt.Errorf("unsupported channel: %s", channel)
	}

	return title, body, nil
}

// ================================================
// VALIDATE TEMPLATE VARIABLES
// ================================================

func (s *templateService) ValidateTemplateVariables(ctx context.Context, templateCode string, data map[string]interface{}) error {
	template, err := s.templateRepo.GetByCode(ctx, templateCode)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	return s.validateVariables(template, data)
}

// ================================================
// HELPER METHODS
// ================================================

// renderString replaces {{variable}} with actual values
func (s *templateService) renderString(template string, data map[string]interface{}) string {
	result := template

	// Find all {{variable}} patterns
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		varName := match[1]
		placeholder := match[0] // {{variable}}

		// Get value from data
		if value, ok := data[varName]; ok {
			// Convert value to string
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			case int, int64, float64:
				strValue = fmt.Sprintf("%v", v)
			default:
				strValue = fmt.Sprintf("%v", v)
			}

			result = strings.Replace(result, placeholder, strValue, -1)
		} else {
			// Leave placeholder if variable not found (will be caught by validation)
			log.Warn().
				Str("variable", varName).
				Msg("Variable not found in data")
		}
	}

	return result
}

// validateVariables checks if all required variables are present in data
func (s *templateService) validateVariables(template *model.NotificationTemplate, data map[string]interface{}) error {
	if template.RequiredVariables == nil || len(template.RequiredVariables) == 0 {
		return nil
	}

	missingVars := []string{}
	for _, required := range template.RequiredVariables {
		if _, ok := data[required]; !ok {
			missingVars = append(missingVars, required)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missingVars, ", "))
	}

	return nil
}

// isValidTemplateCode checks if template code follows naming convention
func isValidTemplateCode(code string) bool {
	// Only lowercase letters, numbers, and underscores
	re := regexp.MustCompile(`^[a-z0-9_]+$`)
	return re.MatchString(code)
}

// isContentChanged checks if any content field was updated
func (s *templateService) isContentChanged(req model.UpdateTemplateRequest) bool {
	return req.EmailSubject != nil ||
		req.EmailBodyHTML != nil ||
		req.EmailBodyText != nil ||
		req.SMSBody != nil ||
		req.PushTitle != nil ||
		req.PushBody != nil ||
		req.InAppTitle != nil ||
		req.InAppBody != nil
}

// toResponse converts template entity to response DTO
func (s *templateService) toResponse(t *model.NotificationTemplate) *model.TemplateResponse {
	return &model.TemplateResponse{
		ID:                t.ID,
		Code:              t.Code,
		Name:              t.Name,
		Description:       t.Description,
		Category:          t.Category,
		RequiredVariables: t.RequiredVariables,
		Language:          t.Language,
		DefaultChannels:   t.DefaultChannels,
		DefaultPriority:   t.DefaultPriority,
		ExpiresAfterHours: t.ExpiresAfterHours,
		Version:           t.Version,
		IsActive:          t.IsActive,
		CreatedBy:         t.CreatedBy,
		UpdatedBy:         t.UpdatedBy,
		CreatedAt:         t.CreatedAt,
		UpdatedAt:         t.UpdatedAt,
	}
}
