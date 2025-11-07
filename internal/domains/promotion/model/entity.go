package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

// PromotionEntity represents a promotional campaign or discount code in the database
type PromotionEntity struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	Code            string          `json:"code" db:"code"`
	Name            string          `json:"name" db:"name"`
	Description     *string         `json:"description,omitempty" db:"description"`
	
	// Discount details
	DiscountType     string          `json:"discount_type" db:"discount_type"`
	DiscountValue    decimal.Decimal  `json:"discount_value" db:"discount_value"`
	MaxDiscountAmount *decimal.Decimal `json:"max_discount_amount,omitempty" db:"max_discount_amount"`
	
	// Applicability rules
	MinOrderAmount       decimal.Decimal `json:"min_order_amount" db:"min_order_amount"`
	ApplicableCategoryIDs []uuid.UUID    `json:"applicable_category_ids,omitempty" db:"applicable_category_ids"`
	FirstOrderOnly       bool           `json:"first_order_only" db:"first_order_only"`
	
	// Usage limits
	MaxUses         *int `json:"max_uses,omitempty" db:"max_uses"`
	MaxUsesPerUser  int  `json:"max_uses_per_user" db:"max_uses_per_user"`
	CurrentUses     int  `json:"current_uses" db:"current_uses"`
	
	// Validity period
	StartsAt  time.Time `json:"starts_at" db:"starts_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	
	// Status
	IsActive bool `json:"is_active" db:"is_active"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// PromotionUsageEntity represents a record of promotion being used in an order
type PromotionUsageEntity struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	PromotionID    uuid.UUID      `json:"promotion_id" db:"promotion_id"`
	UserID         uuid.UUID      `json:"user_id" db:"user_id"`
	OrderID        uuid.UUID      `json:"order_id" db:"order_id"`
	DiscountAmount decimal.Decimal `json:"discount_amount" db:"discount_amount"`
	UsedAt         time.Time      `json:"used_at" db:"used_at"`
}

// IsValid checks if the promotion is currently valid
func (p *PromotionEntity) IsValid() bool {
	now := time.Now()
	return p.IsActive &&
		now.After(p.StartsAt) &&
		now.Before(p.ExpiresAt)
}

// IsAvailable checks if the promotion can still be used
func (p *PromotionEntity) IsAvailable() bool {
	if !p.IsValid() {
		return false
	}
	if p.MaxUses != nil && p.CurrentUses >= *p.MaxUses {
		return false
	}
	return true
}

// CanBeUsedByUser checks if a user can use this promotion
func (p *PromotionEntity) CanBeUsedByUser(userID uuid.UUID, isFirstOrder bool) (bool, error) {
	// Check if first order only restriction applies
	if p.FirstOrderOnly && !isFirstOrder {
		return false, ErrPromotionFirstOrderOnly
	}
	
	// TODO: Check user's usage count against MaxUsesPerUser
	// This would require repository access, so should be moved to service layer
	
	return true, nil
}

// CalculateDiscount calculates the discount amount for a given order total
func (p *PromotionEntity) CalculateDiscount(orderTotal decimal.Decimal) (decimal.Decimal, error) {
	// Check minimum order amount
	if orderTotal.LessThan(p.MinOrderAmount) {
		return decimal.Zero, ErrOrderAmountTooLow
	}

	var discount decimal.Decimal
	switch p.DiscountType {
	case "percentage":
		// Calculate percentage discount
		discount = orderTotal.Mul(p.DiscountValue).Div(decimal.NewFromInt(100))
		
		// Apply maximum discount cap if exists
		if p.MaxDiscountAmount != nil && discount.GreaterThan(*p.MaxDiscountAmount) {
			discount = *p.MaxDiscountAmount
		}
		
	case "fixed":
		discount = p.DiscountValue
		
		// Fixed discount cannot exceed order total
		if discount.GreaterThan(orderTotal) {
			discount = orderTotal
		}
		
	default:
		return decimal.Zero, ErrInvalidDiscountType
	}
	
	return discount, nil
}