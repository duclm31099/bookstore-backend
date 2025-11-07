package repository

import (
	"context"

	"bookstore-backend/internal/domains/promotion/model"

	"github.com/google/uuid"
)

// PromotionRepository defines the interface for promotion data access
type PromotionRepository interface {
	// Create creates a new promotion
	Create(ctx context.Context, promotion *model.PromotionEntity) (uuid.UUID, error)

	// Update updates an existing promotion
	Update(ctx context.Context, promotion *model.PromotionEntity) error

	// GetByID retrieves a promotion by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.PromotionEntity, error)

	// GetByCode retrieves a promotion by its code
	GetByCode(ctx context.Context, code string) (*model.PromotionEntity, error)

	// List retrieves a paginated list of active promotions
	List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*model.PromotionEntity, int64, error)

	// Delete soft deletes a promotion by setting isActive to false
	Delete(ctx context.Context, id uuid.UUID) error

	// GetActivePromotions retrieves all currently active and valid promotions
	GetActivePromotions(ctx context.Context) ([]*model.PromotionEntity, error)

	// IncrementUsage increments the usage count of a promotion
	IncrementUsage(ctx context.Context, id uuid.UUID) error

	// GetUserPromotionUsage gets the number of times a user has used a promotion
	GetUserPromotionUsage(ctx context.Context, promotionID, userID uuid.UUID) (int, error)

	// RecordPromotionUsage records a usage of a promotion in an order
	RecordPromotionUsage(ctx context.Context, usage *model.PromotionUsageEntity) error

	// GetPromotionUsageHistory gets the usage history of a promotion
	GetPromotionUsageHistory(ctx context.Context, promotionID uuid.UUID, page, pageSize int) ([]*model.PromotionUsageEntity, int64, error)
}
