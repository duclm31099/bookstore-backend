package service

import (
	"context"
	"time"

	cart "bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/internal/domains/promotion/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ServiceInterface interface {
	ValidatePromotion(ctx context.Context, req *model.ValidatePromotionRequest) (*model.ValidationResult, error)
	ApplyPromotionToCart(ctx context.Context, userID uuid.UUID, code string) (*cart.CartResponse, error)
	RemovePromotionFromCart(ctx context.Context, userID uuid.UUID) (*cart.CartResponse, error)
	ListActivePromotions(ctx context.Context, categoryID *uuid.UUID, page, limit int) ([]*model.Promotion, int, error)

	// Admin methods
	CreatePromotion(ctx context.Context, req *model.CreatePromotionRequest) (*model.Promotion, error)
	UpdatePromotion(ctx context.Context, id uuid.UUID, req *model.UpdatePromotionRequest) (*model.Promotion, error)
	GetPromotionByID(ctx context.Context, id uuid.UUID) (*model.PromotionDetailResponse, error)
	ListPromotions(ctx context.Context, filter *model.ListPromotionsFilter) ([]*model.PromotionListItem, int, error)
	UpdatePromotionStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	DeletePromotion(ctx context.Context, id uuid.UUID) error
	GetUsageHistory(ctx context.Context, promoID uuid.UUID, startDate, endDate *time.Time, userID *uuid.UUID, page, limit int) (*model.UsageHistoryResponse, error)
	// Internal methods (called by Order service)
	RecordUsage(ctx context.Context, orderID, promoID, userID uuid.UUID, discountAmount interface{}) error
	CalculateDiscount(promo *model.Promotion, subtotal decimal.Decimal) decimal.Decimal
}
