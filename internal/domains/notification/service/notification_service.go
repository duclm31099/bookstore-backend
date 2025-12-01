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
// NOTIFICATION SERVICE IMPLEMENTATION
// ================================================

type notificationService struct {
	notifRepo       repository.NotificationRepository
	prefsRepo       repository.PreferencesRepository
	templateRepo    repository.TemplateRepository
	rateLimitRepo   repository.RateLimitRepository
	deliveryLogRepo repository.DeliveryLogRepository

	// Dependencies
	prefsService    PreferencesService
	templateService TemplateService
	deliveryService DeliveryService
}

func NewNotificationService(
	notifRepo repository.NotificationRepository,
	prefsRepo repository.PreferencesRepository,
	templateRepo repository.TemplateRepository,
	rateLimitRepo repository.RateLimitRepository,
	deliveryLogRepo repository.DeliveryLogRepository,
) NotificationService {
	return &notificationService{
		notifRepo:       notifRepo,
		prefsRepo:       prefsRepo,
		templateRepo:    templateRepo,
		rateLimitRepo:   rateLimitRepo,
		deliveryLogRepo: deliveryLogRepo,
	}
}

// SetDependencies sets circular dependencies (called after all services created)
func (s *notificationService) SetDependencies(
	prefsService PreferencesService,
	templateService TemplateService,
	deliveryService DeliveryService,
) {
	s.prefsService = prefsService
	s.templateService = templateService
	s.deliveryService = deliveryService
}

// ================================================
// SEND NOTIFICATION (Main Entry Point)
// ================================================

func (s *notificationService) SendNotification(ctx context.Context, req model.SendNotificationRequest) (*model.Notification, error) {
	log.Info().
		Str("user_id", req.UserID.String()).
		Str("template_code", req.TemplateCode).
		Msg("[NotificationService] SendNotification started")

	// 1. VALIDATE TEMPLATE EXISTS AND ACTIVE
	template, err := s.templateRepo.GetByCode(ctx, req.TemplateCode)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	if !template.IsActive {
		return nil, model.ErrTemplateInactive
	}

	// 2. VALIDATE TEMPLATE VARIABLES
	if err := s.templateService.ValidateTemplateVariables(ctx, req.TemplateCode, req.Data); err != nil {
		return nil, fmt.Errorf("validate template variables: %w", err)
	}

	// 3. DETERMINE CHANNELS (use request channels or template defaults)
	channels := req.Channels
	if len(channels) == 0 {
		channels = template.DefaultChannels
	}

	// 4. FILTER CHANNELS BASED ON USER PREFERENCES
	allowedChannels := []string{}
	for _, channel := range channels {
		allowed, reason, err := s.prefsService.CanSendNotification(ctx, req.UserID, template.Code, channel)
		if err != nil {
			log.Warn().Err(err).Str("channel", channel).Msg("Error checking channel permission")
			continue
		}
		if allowed {
			allowedChannels = append(allowedChannels, channel)
		} else {
			log.Info().
				Str("user_id", req.UserID.String()).
				Str("channel", channel).
				Str("reason", reason).
				Msg("Channel blocked by user preference")
		}
	}

	if len(allowedChannels) == 0 {
		log.Warn().
			Str("user_id", req.UserID.String()).
			Str("template_code", req.TemplateCode).
			Msg("All channels blocked by user preferences")
		return nil, fmt.Errorf("no available channels: all blocked by user preferences")
	}

	// 5. CHECK RATE LIMIT
	allowed, err := s.rateLimitRepo.CheckLimit(
		ctx,
		model.RateLimitScopeUser,
		req.UserID.String(),
		10, // Max 10 notifications per window
		60, // 60 minutes window
	)
	if err != nil {
		log.Error().Err(err).Msg("Error checking rate limit")
		// Don't fail, just log and continue
	} else if !allowed {
		return nil, model.ErrRateLimitExceeded
	}

	// 6. RENDER TEMPLATES FOR EACH CHANNEL
	renderedContent := make(map[string]string)
	for _, channel := range allowedChannels {
		title, body, err := s.templateService.RenderTemplate(ctx, req.TemplateCode, channel, req.Data)
		if err != nil {
			log.Error().Err(err).Str("channel", channel).Msg("Error rendering template")
			continue
		}

		if channel == model.ChannelInApp {
			renderedContent["title"] = title
			renderedContent["message"] = body
		}
		renderedContent[channel] = body
	}

	// 7. GENERATE IDEMPOTENCY KEY
	idempotencyKey := s.generateIdempotencyKey(req.TemplateCode, req.ReferenceID, req.UserID)

	// 8. CHECK FOR DUPLICATE (Idempotency)
	existing, err := s.notifRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil && existing != nil {
		log.Info().
			Str("idempotency_key", idempotencyKey).
			Msg("Duplicate notification detected, returning existing")
		return existing, nil
	}

	// 9. DETERMINE PRIORITY
	priority := template.DefaultPriority
	if req.Priority != nil {
		priority = *req.Priority
	}

	// 10. CALCULATE EXPIRATION
	var expiresAt *time.Time
	if template.ExpiresAfterHours != nil && *template.ExpiresAfterHours > 0 {
		expires := time.Now().Add(time.Duration(*template.ExpiresAfterHours) * time.Hour)
		expiresAt = &expires
	}

	// 11. CREATE NOTIFICATION RECORD
	notification := &model.Notification{
		UserID:          req.UserID,
		Type:            template.Code,
		Title:           renderedContent["title"],
		Message:         renderedContent["message"],
		Data:            req.Data,
		Channels:        allowedChannels,
		DeliveryStatus:  model.JSONB{},
		ReferenceType:   req.ReferenceType,
		ReferenceID:     req.ReferenceID,
		IdempotencyKey:  &idempotencyKey,
		Priority:        priority,
		ExpiresAt:       expiresAt,
		TemplateCode:    &req.TemplateCode,
		TemplateVersion: &template.Version,
		TemplateData:    req.Data,
		IsRead:          false,
		IsSent:          false,
	}

	if err := s.notifRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	// 12. INCREMENT RATE LIMIT COUNTER
	if err := s.rateLimitRepo.IncrementCount(ctx, model.RateLimitScopeUser, req.UserID.String(), 60); err != nil {
		log.Warn().Err(err).Msg("Error incrementing rate limit counter")
	}

	// 13. SEND VIA DELIVERY SERVICE (Async - Non-blocking)
	// This will be handled by background worker via queue
	// For now, we just create the notification and worker will pick it up

	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("user_id", req.UserID.String()).
		Strs("channels", allowedChannels).
		Msg("[NotificationService] Notification created successfully")

	return notification, nil
}

// ================================================
// CREATE NOTIFICATION (Without Template)
// ================================================

func (s *notificationService) CreateNotification(ctx context.Context, req model.CreateNotificationRequest) (*model.Notification, error) {
	log.Info().
		Str("user_id", req.UserID.String()).
		Str("type", req.Type).
		Msg("[NotificationService] CreateNotification started")

	// 1. VALIDATE NOTIFICATION TYPE
	validTypes := []string{
		model.NotificationTypePromotionRemoved,
		model.NotificationTypeOrderStatus,
		model.NotificationTypePayment,
		model.NotificationTypeNewPromotion,
		model.NotificationTypeReviewResponse,
		model.NotificationTypeSystemAlert,
	}

	isValid := false
	for _, vt := range validTypes {
		if req.Type == vt {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, model.ErrInvalidNotificationType
	}

	// 2. VALIDATE CHANNELS
	for _, channel := range req.Channels {
		if channel != model.ChannelInApp &&
			channel != model.ChannelEmail &&
			channel != model.ChannelPush &&
			channel != model.ChannelSMS {
			return nil, model.ErrInvalidChannel
		}
	}

	// 3. CHECK USER PREFERENCES FOR EACH CHANNEL
	allowedChannels := []string{}
	for _, channel := range req.Channels {
		allowed, reason, err := s.prefsService.CanSendNotification(ctx, req.UserID, req.Type, channel)
		if err != nil {
			log.Warn().Err(err).Str("channel", channel).Msg("Error checking channel permission")
			continue
		}
		if allowed {
			allowedChannels = append(allowedChannels, channel)
		} else {
			log.Info().
				Str("channel", channel).
				Str("reason", reason).
				Msg("Channel blocked by user preference")
		}
	}

	if len(allowedChannels) == 0 {
		return nil, fmt.Errorf("no available channels: all blocked by user preferences")
	}

	// 4. CHECK RATE LIMIT
	allowed, err := s.rateLimitRepo.CheckLimit(
		ctx,
		model.RateLimitScopeUser,
		req.UserID.String(),
		10,
		60,
	)
	if err != nil {
		log.Warn().Err(err).Msg("Error checking rate limit")
	} else if !allowed {
		return nil, model.ErrRateLimitExceeded
	}

	// 5. SET DEFAULT PRIORITY
	priority := model.PriorityMedium
	if req.Priority != nil {
		priority = *req.Priority
	}

	// 6. CREATE NOTIFICATION
	notification := &model.Notification{
		UserID:         req.UserID,
		Type:           req.Type,
		Title:          req.Title,
		Message:        req.Message,
		Data:           req.Data,
		Channels:       allowedChannels,
		DeliveryStatus: model.JSONB{},
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		Priority:       priority,
		ExpiresAt:      req.ExpiresAt,
		TemplateCode:   req.TemplateCode,
		TemplateData:   req.TemplateData,
		IsRead:         false,
		IsSent:         false,
	}

	// 7. GENERATE IDEMPOTENCY KEY IF REFERENCE PROVIDED
	if req.ReferenceID != nil && req.ReferenceType != nil {
		key := s.generateIdempotencyKey(req.Type, req.ReferenceID, req.UserID)
		notification.IdempotencyKey = &key

		// Check for duplicate
		existing, err := s.notifRepo.GetByIdempotencyKey(ctx, key)
		if err == nil && existing != nil {
			log.Info().Str("idempotency_key", key).Msg("Duplicate notification")
			return existing, nil
		}
	}

	if err := s.notifRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	// 8. INCREMENT RATE LIMIT
	if err := s.rateLimitRepo.IncrementCount(ctx, model.RateLimitScopeUser, req.UserID.String(), 60); err != nil {
		log.Warn().Err(err).Msg("Error incrementing rate limit")
	}

	log.Info().
		Str("notification_id", notification.ID.String()).
		Msg("[NotificationService] Notification created")

	return notification, nil
}

// ================================================
// GET NOTIFICATION BY ID
// ================================================

func (s *notificationService) GetNotificationByID(ctx context.Context, userID, notificationID uuid.UUID) (*model.NotificationResponse, error) {
	notification, err := s.notifRepo.GetByID(ctx, notificationID)
	if err != nil {
		return nil, err
	}

	// Security: Ensure notification belongs to user
	if notification.UserID != userID {
		return nil, model.ErrNotificationNotFound
	}

	// Check if expired
	if notification.ExpiresAt != nil && notification.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrNotificationExpired
	}

	return s.toResponse(notification), nil
}

// ================================================
// LIST NOTIFICATIONS
// ================================================

func (s *notificationService) ListNotifications(ctx context.Context, req model.ListNotificationsRequest) (*model.NotificationListResponse, error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Get notifications
	notifications, total, err := s.notifRepo.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	// Get unread count
	unreadCount, err := s.notifRepo.GetUnreadCount(ctx, req.UserID)
	if err != nil {
		log.Warn().Err(err).Msg("Error getting unread count")
		unreadCount = 0
	}

	// Convert to response
	responses := make([]model.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = *s.toResponse(&n)
	}

	// Calculate pagination
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &model.NotificationListResponse{
		Notifications: responses,
		Pagination: model.PaginationMeta{
			CurrentPage:  req.Page,
			PageSize:     req.PageSize,
			TotalPages:   totalPages,
			TotalRecords: total,
		},
		UnreadCount: unreadCount,
	}, nil
}

// ================================================
// GET UNREAD COUNT
// ================================================

func (s *notificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifRepo.GetUnreadCount(ctx, userID)
}

// ================================================
// MARK AS READ
// ================================================

func (s *notificationService) MarkAsRead(ctx context.Context, userID uuid.UUID, req model.MarkAsReadRequest) error {
	if len(req.NotificationIDs) == 0 {
		return fmt.Errorf("no notification IDs provided")
	}

	return s.notifRepo.MarkAsRead(ctx, req.NotificationIDs, userID)
}

// ================================================
// MARK ALL AS READ
// ================================================

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifRepo.MarkAllAsRead(ctx, userID)
}

// ================================================
// DELETE NOTIFICATION
// ================================================

func (s *notificationService) DeleteNotification(ctx context.Context, userID, notificationID uuid.UUID) error {
	// Security check: verify ownership
	notification, err := s.notifRepo.GetByID(ctx, notificationID)
	if err != nil {
		return err
	}

	if notification.UserID != userID {
		return model.ErrNotificationNotFound
	}

	return s.notifRepo.Delete(ctx, notificationID)
}

// ================================================
// BACKGROUND JOB: PROCESS UNSENT NOTIFICATIONS
// ================================================

func (s *notificationService) ProcessUnsentNotifications(ctx context.Context, limit int) error {
	log.Info().Int("limit", limit).Msg("[Background] Processing unsent notifications")

	notifications, err := s.notifRepo.GetUnsentNotifications(ctx, limit)
	if err != nil {
		return fmt.Errorf("get unsent notifications: %w", err)
	}

	if len(notifications) == 0 {
		log.Info().Msg("[Background] No unsent notifications")
		return nil
	}

	successCount := 0
	errorCount := 0

	for _, notification := range notifications {
		// Send via delivery service for each channel
		for _, channel := range notification.Channels {
			var recipient string

			// Get recipient based on channel
			// This would typically fetch from user profile
			// For now, we'll use placeholder logic

			var err error
			switch channel {
			case model.ChannelEmail:
				err = s.deliveryService.SendEmail(ctx, &notification, recipient)
			case model.ChannelSMS:
				err = s.deliveryService.SendSMS(ctx, &notification, recipient)
			case model.ChannelPush:
				err = s.deliveryService.SendPush(ctx, &notification, recipient)
			}

			if err != nil {
				log.Error().
					Err(err).
					Str("notification_id", notification.ID.String()).
					Str("channel", channel).
					Msg("Error sending notification")
				errorCount++
			} else {
				successCount++
			}
		}

		// Mark as sent
		if err := s.notifRepo.MarkAsSent(ctx, notification.ID); err != nil {
			log.Error().Err(err).Str("notification_id", notification.ID.String()).Msg("Error marking as sent")
		}
	}

	log.Info().
		Int("success", successCount).
		Int("errors", errorCount).
		Msg("[Background] Finished processing unsent notifications")

	return nil
}

// ================================================
// BACKGROUND JOB: CLEANUP EXPIRED
// ================================================

func (s *notificationService) CleanupExpiredNotifications(ctx context.Context) (int, error) {
	log.Info().Msg("[Background] Cleaning up expired notifications")

	count, err := s.notifRepo.DeleteExpired(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("delete expired: %w", err)
	}

	log.Info().Int("count", count).Msg("[Background] Expired notifications cleaned up")
	return count, nil
}

// ================================================
// BACKGROUND JOB: CLEANUP OLD READ
// ================================================

func (s *notificationService) CleanupOldReadNotifications(ctx context.Context, olderThan time.Duration) (int, error) {
	log.Info().Dur("older_than", olderThan).Msg("[Background] Cleaning up old read notifications")

	before := time.Now().Add(-olderThan)
	count, err := s.notifRepo.DeleteOldRead(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("delete old read: %w", err)
	}

	log.Info().Int("count", count).Msg("[Background] Old read notifications cleaned up")
	return count, nil
}

// ================================================
// HELPER METHODS
// ================================================

func (s *notificationService) generateIdempotencyKey(notificationType string, referenceID *uuid.UUID, userID uuid.UUID) string {
	if referenceID == nil {
		return fmt.Sprintf("%s:%s:%d", notificationType, userID.String(), time.Now().Unix())
	}
	return fmt.Sprintf("%s:%s:%s", notificationType, referenceID.String(), userID.String())
}

func (s *notificationService) toResponse(n *model.Notification) *model.NotificationResponse {
	return &model.NotificationResponse{
		ID:             n.ID,
		Type:           n.Type,
		Title:          n.Title,
		Message:        n.Message,
		Data:           n.Data,
		IsRead:         n.IsRead,
		ReadAt:         n.ReadAt,
		IsSent:         n.IsSent,
		SentAt:         n.SentAt,
		Channels:       n.Channels,
		DeliveryStatus: n.DeliveryStatus,
		ReferenceType:  n.ReferenceType,
		ReferenceID:    n.ReferenceID,
		Priority:       n.Priority,
		ExpiresAt:      n.ExpiresAt,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
	}
}
