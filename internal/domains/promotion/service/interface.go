package service

// import (
// 	"context"

// 	"github.com/google/uuid"
// 	"github.com/shopspring/decimal"
// 	 "bookstore-backend/internal/domains/promotion/model"
// 	 cart  "bookstore-backend/internal/domains/cart/model"
// )

// type ServiceInterface interface {
// 	ValidatePromotion(ctx context.Context, req *model.ValidateRequest) (*model.ValidationResult, error)
// 	ApplyPromotionToCart(ctx context.Context, userID uuid.UUID, code string) (*cart.Cart, error)
// 	RemovePromotionFromCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error)
// 	ListActivePromotions(ctx context.Context, filters *model.PublicFilters) ([]*model.Promotion, *model.Pagination, error)

// 	// Admin methods
// 	CreatePromotion(ctx context.Context, req *model.CreatePromotionRequest) (*model.Promotion, error)
// 	UpdatePromotion(ctx context.Context, id uuid.UUID, req *model.UpdatePromotionRequest) (*model.Promotion, error)
// 	GetPromotionByID(ctx context.Context, id uuid.UUID) (*model.PromotionWithStats, error)
// 	ListPromotions(ctx context.Context, filters *model.AdminFilters) ([]*model.PromotionListItem, *model.Pagination, error)
// 	UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error
// 	DeletePromotion(ctx context.Context, id uuid.UUID) error
// 	GetUsageHistory(ctx context.Context, id uuid.UUID, filters *model.UsageFilters) (*model.UsageHistoryResponse, error)

// 	// Internal methods (called by Order service)
// 	RecordUsage(ctx context.Context, req *model.RecordUsageRequest) error
// 	CalculateDiscount(promo *model.Promotion, subtotal decimal.Decimal) decimal.Decimal
// }
