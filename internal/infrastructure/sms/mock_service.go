package sms

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// ================================================
// MOCK SMS SERVICE (for development)
// ================================================

type MockSMSService struct{}

func NewMockSMSService() *MockSMSService {
	return &MockSMSService{}
}

// SendSMS implements notification DeliveryService.SMSProvider interface
func (s *MockSMSService) SendSMS(ctx context.Context, to, message string) (messageID string, err error) {
	log.Info().
		Str("to", to).
		Str("message", message).
		Msg("[MOCK] SMS sent successfully")

	// Simulate success
	messageID = fmt.Sprintf("mock-sms-%d", time.Now().Unix())
	return messageID, nil
}

// ================================================
// TODO: TWILIO SMS SERVICE (for production)
// ================================================

type TwilioSMSService struct {
	accountSID string
	authToken  string
	fromNumber string
}

func NewTwilioSMSService(accountSID, authToken, fromNumber string) *TwilioSMSService {
	return &TwilioSMSService{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
	}
}

func (s *TwilioSMSService) SendSMS(ctx context.Context, to, message string) (messageID string, err error) {
	// TODO: Implement Twilio API call
	log.Warn().Msg("Twilio SMS not implemented yet, using mock")
	return NewMockSMSService().SendSMS(ctx, to, message)
}
