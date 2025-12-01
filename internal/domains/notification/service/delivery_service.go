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
// DELIVERY SERVICE IMPLEMENTATION
// ================================================

type deliveryService struct {
	notifRepo       repository.NotificationRepository
	deliveryLogRepo repository.DeliveryLogRepository

	// External service providers (will be injected)
	emailProvider EmailProvider
	smsProvider   SMSProvider
	pushProvider  PushProvider
}

// ================================================
// PROVIDER INTERFACES (External Services)
// ================================================

type EmailProvider interface {
	SendEmail(ctx context.Context, to, subject, body string) (messageID string, err error)
}

type SMSProvider interface {
	SendSMS(ctx context.Context, to, message string) (messageID string, err error)
}

type PushProvider interface {
	SendPush(ctx context.Context, deviceToken, title, body string, data map[string]interface{}) (messageID string, err error)
}

// ================================================
// CONSTRUCTOR
// ================================================

func NewDeliveryService(
	notifRepo repository.NotificationRepository,
	deliveryLogRepo repository.DeliveryLogRepository,
	emailProvider EmailProvider,
	smsProvider SMSProvider,
	pushProvider PushProvider,
) DeliveryService {
	return &deliveryService{
		notifRepo:       notifRepo,
		deliveryLogRepo: deliveryLogRepo,
		emailProvider:   emailProvider,
		smsProvider:     smsProvider,
		pushProvider:    pushProvider,
	}
}

// ================================================
// SEND EMAIL
// ================================================

func (s *deliveryService) SendEmail(ctx context.Context, notification *model.Notification, recipient string) error {
	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("recipient", recipient).
		Msg("[DeliveryService] SendEmail")

	// 1. CREATE DELIVERY LOG (QUEUED)
	deliveryLog := &model.DeliveryLog{
		NotificationID: notification.ID,
		Channel:        model.ChannelEmail,
		AttemptNumber:  1,
		Status:         model.DeliveryStatusQueued,
		Recipient:      recipient,
		Provider:       stringPtr("aws_ses"), // or your email provider
		MaxRetries:     3,
	}

	now := time.Now()
	deliveryLog.QueuedAt = &now

	if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
		return fmt.Errorf("create delivery log: %w", err)
	}

	// 2. UPDATE STATUS TO PROCESSING
	deliveryLog.Status = model.DeliveryStatusProcessing
	processingTime := time.Now()
	deliveryLog.ProcessingAt = &processingTime

	if err := s.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		log.Warn().Err(err).Msg("Failed to update delivery log to processing")
	}

	// 3. SEND EMAIL VIA PROVIDER
	messageID, err := s.emailProvider.SendEmail(ctx, recipient, notification.Title, notification.Message)

	if err != nil {
		// 4a. MARK AS FAILED
		log.Error().
			Err(err).
			Str("notification_id", notification.ID.String()).
			Str("recipient", recipient).
			Msg("Failed to send email")

		errCode := "EMAIL_SEND_FAILED"
		errMsg := err.Error()

		if err := s.deliveryLogRepo.MarkAsFailed(ctx, deliveryLog.ID, errCode, errMsg); err != nil {
			log.Error().Err(err).Msg("Failed to mark delivery log as failed")
		}

		// Update notification delivery status
		if err := s.notifRepo.UpdateDeliveryStatus(ctx, notification.ID, model.ChannelEmail, "failed"); err != nil {
			log.Error().Err(err).Msg("Failed to update notification delivery status")
		}

		return fmt.Errorf("send email: %w", err)
	}

	// 4b. MARK AS SENT
	sentTime := time.Now()
	deliveryLog.Status = model.DeliveryStatusSent
	deliveryLog.SentAt = &sentTime
	deliveryLog.ProviderMessageID = &messageID

	if err := s.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		log.Warn().Err(err).Msg("Failed to update delivery log to sent")
	}

	// Update notification delivery status
	if err := s.notifRepo.UpdateDeliveryStatus(ctx, notification.ID, model.ChannelEmail, "sent"); err != nil {
		log.Error().Err(err).Msg("Failed to update notification delivery status")
	}

	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("message_id", messageID).
		Msg("[DeliveryService] Email sent successfully")

	return nil
}

// ================================================
// SEND SMS
// ================================================

func (s *deliveryService) SendSMS(ctx context.Context, notification *model.Notification, recipient string) error {
	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("recipient", recipient).
		Msg("[DeliveryService] SendSMS")

	// 1. CREATE DELIVERY LOG (QUEUED)
	deliveryLog := &model.DeliveryLog{
		NotificationID: notification.ID,
		Channel:        model.ChannelSMS,
		AttemptNumber:  1,
		Status:         model.DeliveryStatusQueued,
		Recipient:      recipient,
		Provider:       stringPtr("twilio"), // or your SMS provider
		MaxRetries:     3,
	}

	now := time.Now()
	deliveryLog.QueuedAt = &now

	if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
		return fmt.Errorf("create delivery log: %w", err)
	}

	// 2. UPDATE STATUS TO PROCESSING
	deliveryLog.Status = model.DeliveryStatusProcessing
	processingTime := time.Now()
	deliveryLog.ProcessingAt = &processingTime

	if err := s.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		log.Warn().Err(err).Msg("Failed to update delivery log to processing")
	}

	// 3. SEND SMS VIA PROVIDER
	messageID, err := s.smsProvider.SendSMS(ctx, recipient, notification.Message)

	if err != nil {
		// 4a. MARK AS FAILED
		log.Error().
			Err(err).
			Str("notification_id", notification.ID.String()).
			Str("recipient", recipient).
			Msg("Failed to send SMS")

		errCode := "SMS_SEND_FAILED"
		errMsg := err.Error()

		if err := s.deliveryLogRepo.MarkAsFailed(ctx, deliveryLog.ID, errCode, errMsg); err != nil {
			log.Error().Err(err).Msg("Failed to mark delivery log as failed")
		}

		// Update notification delivery status
		if err := s.notifRepo.UpdateDeliveryStatus(ctx, notification.ID, model.ChannelSMS, "failed"); err != nil {
			log.Error().Err(err).Msg("Failed to update notification delivery status")
		}

		return fmt.Errorf("send sms: %w", err)
	}

	// 4b. MARK AS SENT
	sentTime := time.Now()
	deliveryLog.Status = model.DeliveryStatusSent
	deliveryLog.SentAt = &sentTime
	deliveryLog.ProviderMessageID = &messageID

	if err := s.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		log.Warn().Err(err).Msg("Failed to update delivery log to sent")
	}

	// Update notification delivery status
	if err := s.notifRepo.UpdateDeliveryStatus(ctx, notification.ID, model.ChannelSMS, "sent"); err != nil {
		log.Error().Err(err).Msg("Failed to update notification delivery status")
	}

	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("message_id", messageID).
		Msg("[DeliveryService] SMS sent successfully")

	return nil
}

// ================================================
// SEND PUSH NOTIFICATION
// ================================================

func (s *deliveryService) SendPush(ctx context.Context, notification *model.Notification, recipient string) error {
	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("recipient", recipient).
		Msg("[DeliveryService] SendPush")

	// 1. CREATE DELIVERY LOG (QUEUED)
	deliveryLog := &model.DeliveryLog{
		NotificationID: notification.ID,
		Channel:        model.ChannelPush,
		AttemptNumber:  1,
		Status:         model.DeliveryStatusQueued,
		Recipient:      recipient,        // FCM token or APNS device token
		Provider:       stringPtr("fcm"), // Firebase Cloud Messaging
		MaxRetries:     3,
	}

	now := time.Now()
	deliveryLog.QueuedAt = &now

	if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
		return fmt.Errorf("create delivery log: %w", err)
	}

	// 2. UPDATE STATUS TO PROCESSING
	deliveryLog.Status = model.DeliveryStatusProcessing
	processingTime := time.Now()
	deliveryLog.ProcessingAt = &processingTime

	if err := s.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		log.Warn().Err(err).Msg("Failed to update delivery log to processing")
	}

	// 3. SEND PUSH VIA PROVIDER
	messageID, err := s.pushProvider.SendPush(
		ctx,
		recipient,
		notification.Title,
		notification.Message,
		notification.Data,
	)

	if err != nil {
		// 4a. MARK AS FAILED
		log.Error().
			Err(err).
			Str("notification_id", notification.ID.String()).
			Str("recipient", recipient).
			Msg("Failed to send push notification")

		errCode := "PUSH_SEND_FAILED"
		errMsg := err.Error()

		if err := s.deliveryLogRepo.MarkAsFailed(ctx, deliveryLog.ID, errCode, errMsg); err != nil {
			log.Error().Err(err).Msg("Failed to mark delivery log as failed")
		}

		// Update notification delivery status
		if err := s.notifRepo.UpdateDeliveryStatus(ctx, notification.ID, model.ChannelPush, "failed"); err != nil {
			log.Error().Err(err).Msg("Failed to update notification delivery status")
		}

		return fmt.Errorf("send push: %w", err)
	}

	// 4b. MARK AS SENT
	sentTime := time.Now()
	deliveryLog.Status = model.DeliveryStatusSent
	deliveryLog.SentAt = &sentTime
	deliveryLog.ProviderMessageID = &messageID

	if err := s.deliveryLogRepo.Update(ctx, deliveryLog); err != nil {
		log.Warn().Err(err).Msg("Failed to update delivery log to sent")
	}

	// Update notification delivery status
	if err := s.notifRepo.UpdateDeliveryStatus(ctx, notification.ID, model.ChannelPush, "sent"); err != nil {
		log.Error().Err(err).Msg("Failed to update notification delivery status")
	}

	log.Info().
		Str("notification_id", notification.ID.String()).
		Str("message_id", messageID).
		Msg("[DeliveryService] Push notification sent successfully")

	return nil
}

// ================================================
// LOG DELIVERY ATTEMPT
// ================================================

func (s *deliveryService) LogDeliveryAttempt(ctx context.Context, notificationID uuid.UUID, channel, recipient, status string) error {
	log.Info().
		Str("notification_id", notificationID.String()).
		Str("channel", channel).
		Str("status", status).
		Msg("[DeliveryService] LogDeliveryAttempt")

	// Get existing logs to determine attempt number
	logs, err := s.deliveryLogRepo.ListByNotificationID(ctx, notificationID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get existing logs")
	}

	attemptNumber := 1
	for _, log := range logs {
		if log.Channel == channel && log.AttemptNumber >= attemptNumber {
			attemptNumber = log.AttemptNumber + 1
		}
	}

	// Create delivery log
	deliveryLog := &model.DeliveryLog{
		NotificationID: notificationID,
		Channel:        channel,
		AttemptNumber:  attemptNumber,
		Status:         status,
		Recipient:      recipient,
		MaxRetries:     3,
	}

	now := time.Now()
	deliveryLog.QueuedAt = &now

	if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
		return fmt.Errorf("create delivery log: %w", err)
	}

	return nil
}

// ================================================
// UPDATE DELIVERY STATUS
// ================================================

func (s *deliveryService) UpdateDeliveryStatus(ctx context.Context, notificationID uuid.UUID, channel, status string) error {
	log.Info().
		Str("notification_id", notificationID.String()).
		Str("channel", channel).
		Str("status", status).
		Msg("[DeliveryService] UpdateDeliveryStatus")

	// Update notification delivery status
	if err := s.notifRepo.UpdateDeliveryStatus(ctx, notificationID, channel, status); err != nil {
		return fmt.Errorf("update notification delivery status: %w", err)
	}

	return nil
}

// ================================================
// RETRY FAILED DELIVERIES (Background Job)
// ================================================

func (s *deliveryService) RetryFailedDeliveries(ctx context.Context, limit int) error {
	log.Info().Int("limit", limit).Msg("[Background] Retrying failed deliveries")

	// Get failed deliveries that need retry
	failedLogs, err := s.deliveryLogRepo.ListFailedRetries(ctx, limit)
	if err != nil {
		return fmt.Errorf("list failed retries: %w", err)
	}

	if len(failedLogs) == 0 {
		log.Info().Msg("[Background] No failed deliveries to retry")
		return nil
	}

	successCount := 0
	errorCount := 0

	for _, deliveryLog := range failedLogs {
		// Get notification
		notification, err := s.notifRepo.GetByID(ctx, deliveryLog.NotificationID)
		if err != nil {
			log.Error().
				Err(err).
				Str("notification_id", deliveryLog.NotificationID.String()).
				Msg("Failed to get notification for retry")
			errorCount++
			continue
		}

		// Retry delivery based on channel
		var retryErr error
		switch deliveryLog.Channel {
		case model.ChannelEmail:
			retryErr = s.SendEmail(ctx, notification, deliveryLog.Recipient)
		case model.ChannelSMS:
			retryErr = s.SendSMS(ctx, notification, deliveryLog.Recipient)
		case model.ChannelPush:
			retryErr = s.SendPush(ctx, notification, deliveryLog.Recipient)
		}

		if retryErr != nil {
			log.Error().
				Err(retryErr).
				Str("notification_id", notification.ID.String()).
				Str("channel", deliveryLog.Channel).
				Msg("Retry failed")
			errorCount++
		} else {
			successCount++
		}
	}

	log.Info().
		Int("success", successCount).
		Int("errors", errorCount).
		Msg("[Background] Finished retrying failed deliveries")

	return nil
}

// ================================================
// HELPER FUNCTIONS
// ================================================

func stringPtr(s string) *string {
	return &s
}
