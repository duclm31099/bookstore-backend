package service

import (
	"bookstore-backend/internal/domains/payment/model"
	"context"

	"github.com/google/uuid"
)

type RefundInterface interface {
	// User endpoints
	RequestRefund(ctx context.Context, userID uuid.UUID, paymentID uuid.UUID, req model.CreateRefundRequestDTO) (*model.RefundRequestResponse, error)
	GetRefundStatus(ctx context.Context, userID uuid.UUID, paymentID uuid.UUID) (*model.RefundRequestResponse, error)

	// Admin endpoints
	ListPendingRefunds(ctx context.Context, page, limit int) ([]*model.RefundRequest, int, error)
	GetRefundDetail(ctx context.Context, refundID uuid.UUID) (*model.RefundRequest, map[string]interface{}, error)
	ApproveRefund(ctx context.Context, adminID uuid.UUID, refundID uuid.UUID, req model.ApproveRefundRequestDTO) (*model.RefundRequestResponse, error)
	RejectRefund(ctx context.Context, adminID uuid.UUID, refundID uuid.UUID, req model.RejectRefundRequestDTO) error
}
