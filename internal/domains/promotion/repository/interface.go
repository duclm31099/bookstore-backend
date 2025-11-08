package repository

import (
	"context"
	"time"

	"bookstore-backend/internal/domains/promotion/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository định nghĩa interface cho promotion data access
type PromotionRepository interface {
	// Read operations
	FindByID(ctx context.Context, id uuid.UUID) (*model.Promotion, error)
	FindByCode(ctx context.Context, code string) (*model.Promotion, error)
	FindByCodeActive(ctx context.Context, code string) (*model.Promotion, error)
	GetUserUsageCount(ctx context.Context, promoID, userID uuid.UUID) (int, error)
	ListActive(ctx context.Context, categoryID *uuid.UUID, page, limit int) ([]*model.Promotion, int, error)
	ListAdmin(ctx context.Context, filter *model.ListPromotionsFilter) ([]*model.PromotionListItem, int, error)

	// Write operations
	Create(ctx context.Context, promo *model.Promotion) error
	Update(ctx context.Context, promo *model.Promotion) error
	UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// Usage tracking
	CreateUsage(ctx context.Context, tx pgx.Tx, usage *model.PromotionUsage) error
	GetUsageHistory(ctx context.Context, promoID uuid.UUID, startDate, endDate *time.Time, userID *uuid.UUID, page, limit int) ([]*model.PromotionUsageWithDetails, int, error)
	GetUsageStats(ctx context.Context, promoID uuid.UUID, startDate, endDate *time.Time) (*model.UsageStats, error)

	// Utility
	CheckCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error)
}
