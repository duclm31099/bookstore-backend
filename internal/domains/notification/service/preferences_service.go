package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/repository"
	"bookstore-backend/pkg/logger"
)

// ================================================
// PREFERENCES SERVICE IMPLEMENTATION
// ================================================

type preferencesService struct {
	prefsRepo repository.PreferencesRepository
}

func NewPreferencesService(prefsRepo repository.PreferencesRepository) PreferencesService {
	return &preferencesService{
		prefsRepo: prefsRepo,
	}
}

// ================================================
// GET USER PREFERENCES
// ================================================

func (s *preferencesService) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*model.PreferencesResponse, error) {
	prefs, err := s.prefsRepo.GetByUserID(ctx, userID)
	if err != nil {
		// If preferences not found, create default preferences
		if err == model.ErrPreferencesNotFound {
			logger.Info("Creating default preferences", map[string]interface{}{
				"user_id": userID.String(),
			})

			defaultPrefs := s.createDefaultPreferences(userID)
			if err := s.prefsRepo.Create(ctx, defaultPrefs); err != nil {
				return nil, fmt.Errorf("create default preferences: %w", err)
			}

			return s.toResponse(defaultPrefs), nil
		}
		return nil, fmt.Errorf("get user preferences: %w", err)
	}

	return s.toResponse(prefs), nil
}

// ================================================
// UPDATE USER PREFERENCES
// ================================================

func (s *preferencesService) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, req model.UpdatePreferencesRequest) (*model.PreferencesResponse, error) {
	logger.Info("[PreferencesService] UpdateUserPreferences", map[string]interface{}{
		"user_id": userID.String(),
	})

	// 1. GET EXISTING PREFERENCES (or create if not exists)
	existing, err := s.prefsRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == model.ErrPreferencesNotFound {
			// Create default and then update
			existing = s.createDefaultPreferences(userID)
			if err := s.prefsRepo.Create(ctx, existing); err != nil {
				return nil, fmt.Errorf("create preferences: %w", err)
			}
		} else {
			return nil, fmt.Errorf("get preferences: %w", err)
		}
	}

	// 2. UPDATE PREFERENCES (merge with existing)
	if req.Preferences != nil {
		// Convert to JSONB
		prefsMap := make(model.JSONB)
		for notifType, channels := range req.Preferences {
			prefsMap[notifType] = map[string]interface{}{
				"in_app": channels.InApp,
				"email":  channels.Email,
				"push":   channels.Push,
			}
		}
		existing.Preferences = prefsMap
	}

	// 3. UPDATE DO NOT DISTURB
	if req.DoNotDisturb != nil {
		existing.DoNotDisturb = *req.DoNotDisturb
	}

	// 4. UPDATE QUIET HOURS
	if req.QuietHoursStart != nil {
		startTime, err := time.Parse("15:04", *req.QuietHoursStart)
		if err != nil {
			return nil, fmt.Errorf("invalid quiet_hours_start format: %w", err)
		}
		existing.QuietHoursStart = &startTime
	}

	if req.QuietHoursEnd != nil {
		endTime, err := time.Parse("15:04", *req.QuietHoursEnd)
		if err != nil {
			return nil, fmt.Errorf("invalid quiet_hours_end format: %w", err)
		}
		existing.QuietHoursEnd = &endTime
	}

	// 5. SAVE UPDATED PREFERENCES
	if err := s.prefsRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update preferences: %w", err)
	}

	logger.Info("[PreferencesService] Preferences updated successfully", map[string]interface{}{
		"user_id": userID.String(),
	})

	return s.toResponse(existing), nil
}

// ================================================
// CAN SEND NOTIFICATION (Check Permission)
// ================================================

func (s *preferencesService) CanSendNotification(ctx context.Context, userID uuid.UUID, notificationType, channel string) (bool, string, error) {
	// 1. CHECK DO NOT DISTURB
	isDND, err := s.prefsRepo.IsDoNotDisturb(ctx, userID)
	if err != nil {
		logger.Error("Error checking do not disturb", err)
	}
	if isDND {
		return false, "User has enabled Do Not Disturb mode", nil
	}

	// 2. CHECK QUIET HOURS (only for email and push, not in-app)
	if channel == model.ChannelEmail || channel == model.ChannelPush {
		inQuietHours, err := s.prefsRepo.IsInQuietHours(ctx, userID, time.Now())
		if err != nil {
			logger.Error("Error checking quiet hours", err)
		}
		if inQuietHours {
			return false, "Currently in user's quiet hours", nil
		}
	}

	// 3. CHECK CHANNEL-SPECIFIC PREFERENCES
	isEnabled, err := s.prefsRepo.IsChannelEnabled(ctx, userID, notificationType, channel)
	if err != nil {
		logger.Error("Error checking channel enabled", err)
		// Default to enabled on error
		return true, "", nil
	}

	if !isEnabled {
		return false, fmt.Sprintf("User has disabled %s notifications for %s", channel, notificationType), nil
	}

	return true, "", nil
}

// ================================================
// IS IN QUIET HOURS
// ================================================

func (s *preferencesService) IsInQuietHours(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.prefsRepo.IsInQuietHours(ctx, userID, time.Now())
}

// ================================================
// HELPER METHODS
// ================================================

func (s *preferencesService) createDefaultPreferences(userID uuid.UUID) *model.NotificationPreferences {
	defaultPrefs := model.JSONB{
		model.NotificationTypePromotionRemoved: map[string]interface{}{
			"in_app": true,
			"email":  false,
			"push":   false,
		},
		model.NotificationTypeOrderStatus: map[string]interface{}{
			"in_app": true,
			"email":  true,
			"push":   true,
		},
		model.NotificationTypePayment: map[string]interface{}{
			"in_app": true,
			"email":  true,
			"push":   false,
		},
		model.NotificationTypeNewPromotion: map[string]interface{}{
			"in_app": true,
			"email":  false,
			"push":   false,
		},
		model.NotificationTypeReviewResponse: map[string]interface{}{
			"in_app": true,
			"email":  false,
			"push":   false,
		},
		model.NotificationTypeSystemAlert: map[string]interface{}{
			"in_app": true,
			"email":  true,
			"push":   false,
		},
	}
	startTime, _ := time.Parse("15:04", "22:00")
	endTime, _ := time.Parse("15:04", "07:00")

	return &model.NotificationPreferences{
		UserID:          userID,
		Preferences:     defaultPrefs,
		DoNotDisturb:    false,
		QuietHoursStart: &startTime,
		QuietHoursEnd:   &endTime,
	}
}

func (s *preferencesService) toResponse(prefs *model.NotificationPreferences) *model.PreferencesResponse {
	// Convert JSONB to map[string]PreferenceChannels
	prefsMap := make(map[string]model.PreferenceChannels)

	for key, value := range prefs.Preferences {
		if channelMap, ok := value.(map[string]interface{}); ok {
			prefsMap[key] = model.PreferenceChannels{
				InApp: getBoolFromMap(channelMap, "in_app", true),
				Email: getBoolFromMap(channelMap, "email", false),
				Push:  getBoolFromMap(channelMap, "push", false),
			}
		}
	}

	// Format quiet hours as HH:MM strings
	var quietStart, quietEnd *string
	if prefs.QuietHoursStart != nil {
		start := prefs.QuietHoursStart.Format("15:04")
		quietStart = &start
	}
	if prefs.QuietHoursEnd != nil {
		end := prefs.QuietHoursEnd.Format("15:04")
		quietEnd = &end
	}

	return &model.PreferencesResponse{
		UserID:          prefs.UserID,
		Preferences:     prefsMap,
		DoNotDisturb:    prefs.DoNotDisturb,
		QuietHoursStart: quietStart,
		QuietHoursEnd:   quietEnd,
		UpdatedAt:       prefs.UpdatedAt,
	}
}

// Helper to safely get bool from map
func getBoolFromMap(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultVal
}
