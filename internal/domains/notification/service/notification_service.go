package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/repository"
	user "bookstore-backend/internal/domains/user"
	"bookstore-backend/pkg/logger"
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
	userRepository  user.Repository

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
	userRepository user.Repository,
) NotificationService {
	return &notificationService{
		notifRepo:       notifRepo,
		prefsRepo:       prefsRepo,
		templateRepo:    templateRepo,
		rateLimitRepo:   rateLimitRepo,
		deliveryLogRepo: deliveryLogRepo,
		userRepository:  userRepository,
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
	logger.Info("[NotificationService] SendNotification started", map[string]interface{}{
		"user_id":       req.UserID.String(),
		"template_code": req.TemplateCode,
	})

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
			logger.Error("Error checking channel permission", err)
			continue
		}
		if allowed {
			allowedChannels = append(allowedChannels, channel)
		} else {
			logger.Info("Channel blocked by user preference", map[string]interface{}{
				"user_id": req.UserID.String(),
				"channel": channel,
				"reason":  reason,
			})
		}
	}

	if len(allowedChannels) == 0 {
		logger.Info("All channels blocked by user preferences", map[string]interface{}{
			"user_id":       req.UserID.String(),
			"template_code": req.TemplateCode,
		})
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
		logger.Error("Error checking rate limit", err)
		// Don't fail, just log and continue
	} else if !allowed {
		return nil, model.ErrRateLimitExceeded
	}

	// 6. RENDER TEMPLATES FOR EACH CHANNEL
	renderedContent := make(map[string]string)
	for _, channel := range allowedChannels {
		title, body, err := s.templateService.RenderTemplate(ctx, req.TemplateCode, channel, req.Data)
		if err != nil {
			logger.Error("Error rendering template", err)
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
		logger.Info("Duplicate notification detected, returning existing", map[string]interface{}{
			"idempotency_key": idempotencyKey,
		})
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
		logger.Error("Error incrementing rate limit counter", err)
	}

	// 13. SEND VIA DELIVERY SERVICE (Async - Non-blocking)
	// This will be handled by background worker via queue
	// For now, we just create the notification and worker will pick it up

	logger.Info("[NotificationService] Notification created successfully", map[string]interface{}{
		"notification_id": notification.ID.String(),
		"user_id":         req.UserID.String(),
		"channels":        allowedChannels,
	})

	return notification, nil
}

// ================================================
// CREATE NOTIFICATION (Without Template)
// ================================================

func (s *notificationService) CreateNotification(ctx context.Context, req model.CreateNotificationRequest) (*model.Notification, error) {
	logger.Info("[NotificationService] CreateNotification started", map[string]interface{}{
		"user_id": req.UserID.String(),
		"type":    req.Type,
	})

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
			logger.Error("Error checking channel permission", err)
			continue
		}
		if allowed {
			allowedChannels = append(allowedChannels, channel)
		} else {
			logger.Info("Channel blocked by user preference", map[string]interface{}{
				"channel": channel,
				"reason":  reason,
			})
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
		logger.Error("Error checking rate limit", err)
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
			logger.Info("Duplicate notification", map[string]interface{}{
				"idempotency_key": key,
			})
			return existing, nil
		}
	}

	if err := s.notifRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	// 8. INCREMENT RATE LIMIT
	if err := s.rateLimitRepo.IncrementCount(ctx, model.RateLimitScopeUser, req.UserID.String(), 60); err != nil {
		logger.Error("Error incrementing rate limit", err)
	}

	logger.Info("[NotificationService] Notification created", map[string]interface{}{
		"notification_id": notification.ID.String(),
	})

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
		logger.Error("Error getting unread count", err)
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
	logger.Info("[Background] Processing unsent notifications", map[string]interface{}{
		"limit": limit,
	})

	notifications, err := s.notifRepo.GetUnsentNotifications(ctx, limit)
	if err != nil {
		return fmt.Errorf("get unsent notifications: %w", err)
	}

	if len(notifications) == 0 {
		logger.Info("[Background] No unsent notifications", map[string]interface{}{})
		return nil
	}

	successCount := 0
	errorCount := 0

	for _, notification := range notifications {
		channelSuccessCount := 0
		channelErrorCount := 0

		// Send via delivery service for each channel
		for _, channel := range notification.Channels {
			// Get recipient based on channel
			var recipient string
			var err error

			switch channel {
			case model.ChannelEmail:
				user, err := s.userRepository.FindByID(ctx, notification.UserID)
				if err != nil {
					logger.Error("Failed to get user for email", err)
					channelErrorCount++
					continue
				}
				recipient = user.Email

			case model.ChannelSMS:
				user, err := s.userRepository.FindByID(ctx, notification.UserID)
				if err != nil {
					logger.Error("Failed to get user for SMS", err)
					channelErrorCount++
					continue
				}
				if user.Phone == nil {
					logger.Info("User has no phone number", map[string]interface{}{
						"user_id": notification.UserID,
					})
					channelErrorCount++
					continue
				}
				recipient = *user.Phone

				// case model.ChannelPush:
				// 	// Get device token from user_devices table
				// 	deviceToken, err := s.deviceRepo.GetActiveToken(ctx, notification.UserID)
				// 	if err != nil {
				// 		logger.Error("Failed to get device token", err)
				// 		channelErrorCount++
				// 		continue
				// 	}
				// 	recipient = deviceToken
			}

			if recipient == "" {
				logger.Info("Empty recipient, skipping channel", map[string]interface{}{
					"channel": channel,
					"user_id": notification.UserID,
				})
				channelErrorCount++
				continue
			}

			// Send notification via appropriate channel
			switch channel {
			case model.ChannelEmail:
				err = s.deliveryService.SendEmail(ctx, &notification, recipient)
			case model.ChannelSMS:
				err = s.deliveryService.SendSMS(ctx, &notification, recipient)
				// case model.ChannelPush:
				// 	err = s.deliveryService.SendPush(ctx, &notification, recipient)
			}

			if err != nil {
				logger.Error("Error sending notification", err)
				channelErrorCount++
			} else {
				channelSuccessCount++
			}
		}

		// Only mark as sent if at least one channel succeeded
		if channelSuccessCount > 0 {
			if err := s.notifRepo.MarkAsSent(ctx, notification.ID); err != nil {
				logger.Error("Error marking as sent", err)
			}
			successCount++
		} else {
			logger.Info("All channels failed for notification, not marking as sent", map[string]interface{}{
				"notification_id": notification.ID,
				"user_id":         notification.UserID,
				"channels":        notification.Channels,
			})
			errorCount++
		}
	}

	logger.Info("[Background] Finished processing unsent notifications", map[string]interface{}{
		"success": successCount,
		"errors":  errorCount,
	})

	return nil
}

// ================================================
// BACKGROUND JOB: CLEANUP EXPIRED
// ================================================

func (s *notificationService) CleanupExpiredNotifications(ctx context.Context) (int, error) {
	logger.Info("[Background] Cleaning up expired notifications", map[string]interface{}{})

	// Use batch delete to avoid long table locks
	// DELETE is atomic in PostgreSQL, so we don't need explicit transaction
	totalDeleted := 0
	batchSize := 1000

	for {
		count, err := s.notifRepo.DeleteExpired(ctx, time.Now())
		if err != nil {
			return totalDeleted, fmt.Errorf("delete expired (deleted %d so far): %w", totalDeleted, err)
		}

		totalDeleted += count

		// If deleted less than batch size, we're done
		if count < batchSize {
			break
		}
	}

	logger.Info("[Background] Expired notifications cleaned up", map[string]interface{}{
		"count": totalDeleted,
	})
	return totalDeleted, nil
}

// ================================================
// BACKGROUND JOB: CLEANUP OLD READ
// ================================================

func (s *notificationService) CleanupOldReadNotifications(ctx context.Context, olderThan time.Duration) (int, error) {
	logger.Info("[Background] Cleaning up old read notifications", map[string]interface{}{
		"older_than": olderThan.String(),
	})

	// Use batch delete to avoid long table locks
	// DELETE is atomic in PostgreSQL, so we don't need explicit transaction
	before := time.Now().Add(-olderThan)
	totalDeleted := 0
	batchSize := 1000

	for {
		count, err := s.notifRepo.DeleteOldRead(ctx, before)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete old read (deleted %d so far): %w", totalDeleted, err)
		}

		totalDeleted += count

		// If deleted less than batch size, we're done
		if count < batchSize {
			break
		}
	}

	logger.Info("[Background] Old read notifications cleaned up", map[string]interface{}{
		"count": totalDeleted,
	})
	return totalDeleted, nil
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
