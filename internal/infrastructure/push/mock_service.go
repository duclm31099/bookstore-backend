package push

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// ================================================
// MOCK PUSH SERVICE (for development)
// ================================================

type MockPushService struct{}

func NewMockPushService() *MockPushService {
	return &MockPushService{}
}

// SendPush implements notification DeliveryService.PushProvider interface
func (s *MockPushService) SendPush(ctx context.Context, deviceToken, title, body string, data map[string]interface{}) (messageID string, err error) {
	log.Info().
		Str("device_token", deviceToken).
		Str("title", title).
		Str("body", body).
		Interface("data", data).
		Msg("[MOCK] Push notification sent successfully")

	// Simulate success
	messageID = fmt.Sprintf("mock-push-%d", time.Now().Unix())
	return messageID, nil
}

// ================================================
// TODO: FCM PUSH SERVICE (for production)
// ================================================

type FCMPushService struct {
	serverKey string
}

func NewFCMPushService(serverKey string) *FCMPushService {
	return &FCMPushService{
		serverKey: serverKey,
	}
}

func (s *FCMPushService) SendPush(ctx context.Context, deviceToken, title, body string, data map[string]interface{}) (messageID string, err error) {
	// TODO: Implement FCM API call
	log.Warn().Msg("FCM Push not implemented yet, using mock")
	return NewMockPushService().SendPush(ctx, deviceToken, title, body, data)
}
