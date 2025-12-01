package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/repository"
)

// ================================================
// CAMPAIGN SERVICE IMPLEMENTATION
// ================================================

type campaignService struct {
	campaignRepo repository.CampaignRepository
	templateRepo repository.TemplateRepository
	notifRepo    repository.NotificationRepository

	// Dependencies
	notificationService NotificationService
	templateService     TemplateService
}

func NewCampaignService(
	campaignRepo repository.CampaignRepository,
	templateRepo repository.TemplateRepository,
	notifRepo repository.NotificationRepository,
) CampaignService {
	return &campaignService{
		campaignRepo: campaignRepo,
		templateRepo: templateRepo,
		notifRepo:    notifRepo,
	}
}

// SetDependencies sets circular dependencies
func (s *campaignService) SetDependencies(
	notificationService NotificationService,
	templateService TemplateService,
) {
	s.notificationService = notificationService
	s.templateService = templateService
}

// ================================================
// CREATE CAMPAIGN (Admin)
// ================================================

func (s *campaignService) CreateCampaign(ctx context.Context, adminID uuid.UUID, req model.CreateCampaignRequest) (*model.CampaignResponse, error) {
	log.Info().
		Str("admin_id", adminID.String()).
		Str("name", req.Name).
		Msg("[CampaignService] CreateCampaign")

	// 1. VALIDATE TEMPLATE EXISTS
	template, err := s.templateRepo.GetByCode(ctx, req.TemplateCode)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	if !template.IsActive {
		return nil, model.ErrTemplateInactive
	}

	// 2. VALIDATE TEMPLATE DATA (variables)
	if err := s.templateService.ValidateTemplateVariables(ctx, req.TemplateCode, req.TemplateData); err != nil {
		return nil, fmt.Errorf("validate template data: %w", err)
	}

	// 3. VALIDATE TARGET TYPE
	if err := s.validateTargetType(req); err != nil {
		return nil, err
	}

	// 4. SET DEFAULTS
	batchSize := 1000
	if req.BatchSize != nil {
		batchSize = *req.BatchSize
	}

	batchDelay := 5
	if req.BatchDelaySeconds != nil {
		batchDelay = *req.BatchDelaySeconds
	}

	// 5. DETERMINE STATUS
	status := model.CampaignStatusDraft
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now()) {
		status = model.CampaignStatusScheduled
	}

	// 6. CREATE CAMPAIGN
	campaign := &model.Campaign{
		Name:              req.Name,
		Description:       req.Description,
		TemplateCode:      &req.TemplateCode,
		TargetType:        req.TargetType,
		TargetSegment:     req.TargetSegment,
		TargetUserIDs:     req.TargetUserIDs,
		TargetFilters:     req.TargetFilters,
		ScheduledAt:       req.ScheduledAt,
		Status:            status,
		BatchSize:         batchSize,
		BatchDelaySeconds: batchDelay,
		TemplateData:      req.TemplateData,
		Channels:          req.Channels,
		CreatedBy:         &adminID,
		ProcessedCount:    0,
		SentCount:         0,
		DeliveredCount:    0,
		FailedCount:       0,
	}

	// 7. CALCULATE TOTAL RECIPIENTS (estimate)
	totalRecipients, err := s.estimateRecipients(ctx, campaign)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to estimate recipients")
	} else {
		campaign.TotalRecipients = &totalRecipients
	}

	// 8. SAVE CAMPAIGN
	if err := s.campaignRepo.Create(ctx, campaign); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	log.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("status", campaign.Status).
		Msg("[CampaignService] Campaign created successfully")

	return s.toResponse(campaign), nil
}

// ================================================
// GET CAMPAIGN BY ID
// ================================================

func (s *campaignService) GetCampaignByID(ctx context.Context, campaignID uuid.UUID) (*model.CampaignResponse, error) {
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return s.toResponse(campaign), nil
}

// ================================================
// LIST CAMPAIGNS (Admin)
// ================================================

func (s *campaignService) ListCampaigns(ctx context.Context, status *string, createdBy *uuid.UUID, page, pageSize int) ([]model.CampaignResponse, int64, error) {
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

	campaigns, total, err := s.campaignRepo.List(ctx, status, createdBy, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}

	responses := make([]model.CampaignResponse, len(campaigns))
	for i, c := range campaigns {
		responses[i] = *s.toResponse(&c)
	}

	return responses, total, nil
}

// ================================================
// START CAMPAIGN (Manual)
// ================================================

func (s *campaignService) StartCampaign(ctx context.Context, campaignID uuid.UUID) error {
	log.Info().
		Str("campaign_id", campaignID.String()).
		Msg("[CampaignService] StartCampaign")

	// 1. GET CAMPAIGN
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}

	// 2. VALIDATE STATUS
	if campaign.Status != model.CampaignStatusDraft && campaign.Status != model.CampaignStatusScheduled {
		return fmt.Errorf("campaign cannot be started: status is %s", campaign.Status)
	}

	// 3. MARK AS RUNNING
	if err := s.campaignRepo.MarkAsStarted(ctx, campaignID); err != nil {
		return fmt.Errorf("mark as started: %w", err)
	}

	// 4. START PROCESSING (this will be handled by background worker)
	// For now, just mark as running and let worker pick it up
	log.Info().
		Str("campaign_id", campaignID.String()).
		Msg("[CampaignService] Campaign started, will be processed by worker")

	return nil
}

// ================================================
// CANCEL CAMPAIGN
// ================================================

func (s *campaignService) CancelCampaign(ctx context.Context, campaignID uuid.UUID) error {
	log.Info().
		Str("campaign_id", campaignID.String()).
		Msg("[CampaignService] CancelCampaign")

	// 1. GET CAMPAIGN
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}

	// 2. VALIDATE STATUS
	if campaign.Status == model.CampaignStatusCompleted || campaign.Status == model.CampaignStatusCancelled {
		return fmt.Errorf("campaign cannot be cancelled: status is %s", campaign.Status)
	}

	// 3. MARK AS CANCELLED
	if err := s.campaignRepo.MarkAsCancelled(ctx, campaignID); err != nil {
		return fmt.Errorf("mark as cancelled: %w", err)
	}

	log.Info().
		Str("campaign_id", campaignID.String()).
		Msg("[CampaignService] Campaign cancelled successfully")

	return nil
}

// ================================================
// PROCESS SCHEDULED CAMPAIGNS (Background Job)
// ================================================

func (s *campaignService) ProcessScheduledCampaigns(ctx context.Context) error {
	log.Info().Msg("[Background] Processing scheduled campaigns")

	// Get campaigns scheduled to run now
	campaigns, err := s.campaignRepo.ListScheduled(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("list scheduled campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		log.Info().Msg("[Background] No scheduled campaigns to process")
		return nil
	}

	for _, campaign := range campaigns {
		log.Info().
			Str("campaign_id", campaign.ID.String()).
			Str("name", campaign.Name).
			Msg("[Background] Starting scheduled campaign")

		// Mark as started
		if err := s.campaignRepo.MarkAsStarted(ctx, campaign.ID); err != nil {
			log.Error().
				Err(err).
				Str("campaign_id", campaign.ID.String()).
				Msg("Failed to mark campaign as started")
			continue
		}

		// Process campaign (send notifications)
		// This will be handled by ProcessCampaignBatch in a separate worker
		log.Info().
			Str("campaign_id", campaign.ID.String()).
			Msg("[Background] Campaign marked as running, worker will process batches")
	}

	return nil
}

// ================================================
// PROCESS CAMPAIGN BATCH (Worker)
// ================================================

func (s *campaignService) ProcessCampaignBatch(ctx context.Context, campaignID uuid.UUID, batchNumber int) error {
	log.Info().
		Str("campaign_id", campaignID.String()).
		Int("batch_number", batchNumber).
		Msg("[Worker] ProcessCampaignBatch")

	// 1. GET CAMPAIGN
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}

	// 2. CHECK IF CAMPAIGN IS STILL RUNNING
	if campaign.Status != model.CampaignStatusRunning {
		log.Warn().
			Str("campaign_id", campaignID.String()).
			Str("status", campaign.Status).
			Msg("Campaign is not running, skipping batch")
		return nil
	}

	// 3. GET TARGET USERS FOR THIS BATCH
	targetUsers, err := s.getTargetUsers(ctx, campaign, batchNumber)
	if err != nil {
		return fmt.Errorf("get target users: %w", err)
	}

	if len(targetUsers) == 0 {
		// No more users to process, mark campaign as completed
		log.Info().
			Str("campaign_id", campaignID.String()).
			Msg("No more users to process, marking campaign as completed")

		if err := s.campaignRepo.MarkAsCompleted(ctx, campaignID); err != nil {
			return fmt.Errorf("mark as completed: %w", err)
		}

		return nil
	}

	// 4. SEND NOTIFICATIONS TO USERS IN BATCH
	sentCount := 0
	failedCount := 0

	for _, userID := range targetUsers {
		// Create notification request
		notifReq := model.SendNotificationRequest{
			UserID:        userID,
			TemplateCode:  *campaign.TemplateCode,
			Channels:      campaign.Channels,
			Data:          campaign.TemplateData,
			ReferenceType: stringPtr("campaign"),
			ReferenceID:   &campaign.ID,
		}

		// Send notification
		_, err := s.notificationService.SendNotification(ctx, notifReq)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID.String()).
				Str("campaign_id", campaignID.String()).
				Msg("Failed to send campaign notification")
			failedCount++
		} else {
			sentCount++
		}
	}

	// 5. UPDATE CAMPAIGN PROGRESS
	if err := s.campaignRepo.UpdateProgress(ctx, campaignID, sentCount, sentCount, failedCount); err != nil {
		log.Error().
			Err(err).
			Str("campaign_id", campaignID.String()).
			Msg("Failed to update campaign progress")
	}

	// 6. DELAY BEFORE NEXT BATCH
	time.Sleep(time.Duration(campaign.BatchDelaySeconds) * time.Second)

	log.Info().
		Str("campaign_id", campaignID.String()).
		Int("batch_number", batchNumber).
		Int("sent", sentCount).
		Int("failed", failedCount).
		Msg("[Worker] Batch processed successfully")

	return nil
}

// ================================================
// HELPER METHODS
// ================================================

// validateTargetType validates campaign target configuration
func (s *campaignService) validateTargetType(req model.CreateCampaignRequest) error {
	switch req.TargetType {
	case model.TargetTypeAllUsers:
		// No additional validation needed
		return nil

	case model.TargetTypeSegment:
		if req.TargetSegment == nil || *req.TargetSegment == "" {
			return fmt.Errorf("target_segment is required for segment target type")
		}
		return nil

	case model.TargetTypeSpecificUsers:
		if req.TargetUserIDs == nil || len(req.TargetUserIDs) == 0 {
			return fmt.Errorf("target_user_ids is required for specific_users target type")
		}
		return nil

	default:
		return fmt.Errorf("invalid target type: %s", req.TargetType)
	}
}

// estimateRecipients estimates total recipients for campaign
func (s *campaignService) estimateRecipients(ctx context.Context, campaign *model.Campaign) (int, error) {
	switch campaign.TargetType {
	case model.TargetTypeSpecificUsers:
		if campaign.TargetUserIDs != nil {
			return len(campaign.TargetUserIDs), nil
		}
		return 0, nil

	case model.TargetTypeSegment:
		// Query database for segment count
		// This would require a user repository
		// For now, return estimate
		log.Warn().Msg("Segment recipient estimation not implemented")
		return 0, nil

	case model.TargetTypeAllUsers:
		// Query database for total active users
		// This would require a user repository
		// For now, return estimate
		log.Warn().Msg("All users recipient estimation not implemented")
		return 0, nil

	default:
		return 0, fmt.Errorf("unknown target type: %s", campaign.TargetType)
	}
}

// getTargetUsers gets users for specific batch
func (s *campaignService) getTargetUsers(ctx context.Context, campaign *model.Campaign, batchNumber int) ([]uuid.UUID, error) {
	offset := (batchNumber - 1) * campaign.BatchSize
	limit := campaign.BatchSize

	switch campaign.TargetType {
	case model.TargetTypeSpecificUsers:
		// Return slice of target users for this batch
		if campaign.TargetUserIDs == nil {
			return []uuid.UUID{}, nil
		}

		start := offset
		end := offset + limit

		if start >= len(campaign.TargetUserIDs) {
			return []uuid.UUID{}, nil
		}

		if end > len(campaign.TargetUserIDs) {
			end = len(campaign.TargetUserIDs)
		}

		return campaign.TargetUserIDs[start:end], nil

	case model.TargetTypeSegment:
		// Query users by segment with pagination
		// This would require integration with user service
		log.Warn().Msg("Segment user fetching not implemented")
		return []uuid.UUID{}, nil

	case model.TargetTypeAllUsers:
		// Query all active users with pagination
		// This would require integration with user service
		log.Warn().Msg("All users fetching not implemented")
		return []uuid.UUID{}, nil

	default:
		return []uuid.UUID{}, fmt.Errorf("unknown target type: %s", campaign.TargetType)
	}
}

// toResponse converts campaign entity to response DTO
func (s *campaignService) toResponse(c *model.Campaign) *model.CampaignResponse {
	return &model.CampaignResponse{
		ID:              c.ID,
		Name:            c.Name,
		Description:     c.Description,
		TemplateCode:    c.TemplateCode,
		Status:          c.Status,
		TotalRecipients: c.TotalRecipients,
		ProcessedCount:  c.ProcessedCount,
		SentCount:       c.SentCount,
		DeliveredCount:  c.DeliveredCount,
		FailedCount:     c.FailedCount,
		ScheduledAt:     c.ScheduledAt,
		StartedAt:       c.StartedAt,
		CompletedAt:     c.CompletedAt,
		CreatedBy:       c.CreatedBy,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}
