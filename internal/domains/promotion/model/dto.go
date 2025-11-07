package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Request DTOs

// CreatePromotionDTO represents the request to create a new promotion
type CreatePromotionDTO struct {
	Code                 string           `json:"code" validate:"required,min=3,max=50"`
	Name                 string           `json:"name" validate:"required,min=3,max=100"`
	Description          *string          `json:"description,omitempty"`
	DiscountType        string           `json:"discount_type" validate:"required,oneof=percentage fixed"`
	DiscountValue       decimal.Decimal   `json:"discount_value" validate:"required,gt=0"`
	MaxDiscountAmount   *decimal.Decimal  `json:"max_discount_amount,omitempty" validate:"omitempty,gt=0"`
	MinOrderAmount      decimal.Decimal   `json:"min_order_amount" validate:"required,gte=0"`
	ApplicableCategoryIDs []uuid.UUID     `json:"applicable_category_ids,omitempty"`
	FirstOrderOnly      bool             `json:"first_order_only"`
	MaxUses            *int             `json:"max_uses,omitempty" validate:"omitempty,gt=0"`
	MaxUsesPerUser     int              `json:"max_uses_per_user" validate:"required,gt=0"`
	StartsAt           time.Time        `json:"starts_at" validate:"required"`
	ExpiresAt          time.Time        `json:"expires_at" validate:"required,gtfield=StartsAt"`
	IsActive           bool             `json:"is_active"`
}

// UpdatePromotionDTO represents the request to update an existing promotion
type UpdatePromotionDTO struct {
	Name                *string           `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	Description         *string           `json:"description,omitempty"`
	DiscountValue      *decimal.Decimal   `json:"discount_value,omitempty" validate:"omitempty,gt=0"`
	MaxDiscountAmount  *decimal.Decimal   `json:"max_discount_amount,omitempty" validate:"omitempty,gt=0"`
	MinOrderAmount     *decimal.Decimal   `json:"min_order_amount,omitempty" validate:"omitempty,gte=0"`
	ApplicableCategoryIDs []uuid.UUID     `json:"applicable_category_ids,omitempty"`
	MaxUses           *int              `json:"max_uses,omitempty" validate:"omitempty,gt=0"`
	MaxUsesPerUser    *int              `json:"max_uses_per_user,omitempty" validate:"omitempty,gt=0"`
	ExpiresAt         *time.Time        `json:"expires_at,omitempty"`
	IsActive          *bool             `json:"is_active,omitempty"`
}

// ValidatePromotionDTO represents the request to validate a promotion code
type ValidatePromotionDTO struct {
	Code       string          `json:"code" validate:"required"`
	OrderTotal decimal.Decimal `json:"order_total" validate:"required,gt=0"`
	UserID     uuid.UUID      `json:"user_id" validate:"required"`
	IsFirstOrder bool         `json:"is_first_order"`
}

// Response DTOs

// PromotionDetailsDTO represents the promotion data sent back to clients
type PromotionDetailsDTO struct {
	ID                  uuid.UUID        `json:"id"`
	Code                string           `json:"code"`
	Name                string           `json:"name"`
	Description         *string          `json:"description,omitempty"`
	DiscountType        string           `json:"discount_type"`
	DiscountValue       decimal.Decimal   `json:"discount_value"`
	MaxDiscountAmount   *decimal.Decimal  `json:"max_discount_amount,omitempty"`
	MinOrderAmount      decimal.Decimal   `json:"min_order_amount"`
	ApplicableCategoryIDs []uuid.UUID     `json:"applicable_category_ids,omitempty"`
	FirstOrderOnly      bool             `json:"first_order_only"`
	MaxUses            *int             `json:"max_uses,omitempty"`
	MaxUsesPerUser     int              `json:"max_uses_per_user"`
	CurrentUses        int              `json:"current_uses"`
	StartsAt           time.Time        `json:"starts_at"`
	ExpiresAt          time.Time        `json:"expires_at"`
	IsActive           bool             `json:"is_active"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

// PromotionListDTO represents a paginated list of promotions
type PromotionListDTO struct {
	Promotions []PromotionDetailsDTO `json:"promotions"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
}

// PromotionValidationResultDTO represents the result of validating/applying a promotion
type PromotionValidationResultDTO struct {
	PromotionID     uuid.UUID       `json:"promotion_id"`
	Code            string          `json:"code"`
	DiscountType    string          `json:"discount_type"`
	DiscountValue   decimal.Decimal  `json:"discount_value"`
	DiscountAmount  decimal.Decimal  `json:"discount_amount"`
	FinalAmount     decimal.Decimal  `json:"final_amount"`
}