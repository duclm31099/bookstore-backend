package model

// import (
// 	"errors"
// 	"strings"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/lib/pq"
// 	"github.com/shopspring/decimal"
// )

// // DiscountType represents valid discount types
// type DiscountType string

// const (
// 	DiscountTypePercentage DiscountType = "percentage"
// 	DiscountTypeFixed      DiscountType = "fixed"
// )

// func (dt DiscountType) IsValid() bool {
// 	switch dt {
// 	case DiscountTypePercentage, DiscountTypeFixed:
// 		return true
// 	}
// 	return false
// }

// func (dt DiscountType) String() string {
// 	return string(dt)
// }

// // Promotion represents promotional codes and discount campaigns
// type Promotion struct {
// 	ID          uuid.UUID `json:"id" db:"id"`
// 	Code        string    `json:"code" db:"code"`
// 	Name        string    `json:"name" db:"name"`
// 	Description *string   `json:"description" db:"description"`

// 	// Discount configuration
// 	DiscountType      string           `json:"discount_type" db:"discount_type"`
// 	DiscountValue     decimal.Decimal  `json:"discount_value" db:"discount_value"`
// 	MaxDiscountAmount *decimal.Decimal `json:"max_discount_amount" db:"max_discount_amount"`

// 	// Conditions
// 	MinOrderAmount        decimal.Decimal `json:"min_order_amount" db:"min_order_amount"`
// 	ApplicableCategoryIDs pq.StringArray  `json:"applicable_category_ids" db:"applicable_category_ids"`
// 	FirstOrderOnly        bool            `json:"first_order_only" db:"first_order_only"`

// 	// Usage limits
// 	MaxUses        *int `json:"max_uses" db:"max_uses"`
// 	MaxUsesPerUser int  `json:"max_uses_per_user" db:"max_uses_per_user"`
// 	CurrentUses    int  `json:"current_uses" db:"current_uses"`

// 	// Validity
// 	StartsAt  time.Time `json:"starts_at" db:"starts_at"`
// 	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`

// 	IsActive  bool      `json:"is_active" db:"is_active"`
// 	CreatedAt time.Time `json:"created_at" db:"created_at"`
// 	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
// }

// // PromotionUsage tracks which users used which promotions in which orders
// type PromotionUsage struct {
// 	ID             uuid.UUID       `json:"id" db:"id"`
// 	PromotionID    uuid.UUID       `json:"promotion_id" db:"promotion_id"`
// 	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
// 	OrderID        uuid.UUID       `json:"order_id" db:"order_id"`
// 	DiscountAmount decimal.Decimal `json:"discount_amount" db:"discount_amount"`
// 	UsedAt         time.Time       `json:"used_at" db:"used_at"`
// }

// // PromotionRequest represents request to create/update promotion
// type PromotionRequest struct {
// 	Code        string  `json:"code" validate:"required,min=3,max=50,alphanum"`
// 	Name        string  `json:"name" validate:"required,min=3,max=200"`
// 	Description *string `json:"description" validate:"omitempty,max=1000"`

// 	DiscountType      string   `json:"discount_type" validate:"required,oneof=percentage fixed"`
// 	DiscountValue     float64  `json:"discount_value" validate:"required,gt=0"`
// 	MaxDiscountAmount *float64 `json:"max_discount_amount" validate:"omitempty,gt=0"`

// 	MinOrderAmount        float64  `json:"min_order_amount" validate:"omitempty,gte=0"`
// 	ApplicableCategoryIDs []string `json:"applicable_category_ids" validate:"omitempty,dive,uuid"`
// 	FirstOrderOnly        bool     `json:"first_order_only"`

// 	MaxUses        *int `json:"max_uses" validate:"omitempty,gt=0"`
// 	MaxUsesPerUser int  `json:"max_uses_per_user" validate:"required,gte=1"`

// 	StartsAt  time.Time `json:"starts_at" validate:"required"`
// 	ExpiresAt time.Time `json:"expires_at" validate:"required,gtfield=StartsAt"`

// 	IsActive bool `json:"is_active"`
// }

// // ValidatePromoRequest represents request to validate promo code
// type ValidatePromoRequest struct {
// 	PromoCode   string          `json:"promo_code" validate:"required,min=3,max=50"`
// 	UserID      *uuid.UUID      `json:"user_id" validate:"omitempty"`
// 	OrderAmount decimal.Decimal `json:"order_amount" validate:"required,gt=0"`
// 	CategoryIDs []uuid.UUID     `json:"category_ids" validate:"omitempty,dive,uuid"`
// }

// // PromotionResponse represents promotion response
// type PromotionResponse struct {
// 	ID          uuid.UUID `json:"id"`
// 	Code        string    `json:"code"`
// 	Name        string    `json:"name"`
// 	Description *string   `json:"description,omitempty"`

// 	DiscountType      string           `json:"discount_type"`
// 	DiscountValue     decimal.Decimal  `json:"discount_value"`
// 	MaxDiscountAmount *decimal.Decimal `json:"max_discount_amount,omitempty"`

// 	MinOrderAmount        decimal.Decimal `json:"min_order_amount"`
// 	ApplicableCategoryIDs []string        `json:"applicable_category_ids,omitempty"`
// 	FirstOrderOnly        bool            `json:"first_order_only"`

// 	MaxUses        *int `json:"max_uses,omitempty"`
// 	MaxUsesPerUser int  `json:"max_uses_per_user"`
// 	CurrentUses    int  `json:"current_uses"`
// 	RemainingUses  *int `json:"remaining_uses,omitempty"`

// 	StartsAt  time.Time `json:"starts_at"`
// 	ExpiresAt time.Time `json:"expires_at"`

// 	IsActive  bool      `json:"is_active"`
// 	IsValid   bool      `json:"is_valid"`
// 	CreatedAt time.Time `json:"created_at"`
// 	UpdatedAt time.Time `json:"updated_at"`
// }

// // PromotionListResponse represents simplified promotion for list views
// type PromotionListResponse struct {
// 	ID            uuid.UUID       `json:"id"`
// 	Code          string          `json:"code"`
// 	Name          string          `json:"name"`
// 	DiscountType  string          `json:"discount_type"`
// 	DiscountValue decimal.Decimal `json:"discount_value"`
// 	CurrentUses   int             `json:"current_uses"`
// 	MaxUses       *int            `json:"max_uses,omitempty"`
// 	StartsAt      time.Time       `json:"starts_at"`
// 	ExpiresAt     time.Time       `json:"expires_at"`
// 	IsActive      bool            `json:"is_active"`
// }

// // PromotionValidationResult represents result of promo validation
// type PromotionValidationResult struct {
// 	IsValid        bool            `json:"is_valid"`
// 	Promotion      *Promotion      `json:"promotion,omitempty"`
// 	DiscountAmount decimal.Decimal `json:"discount_amount"`
// 	ErrorMessage   string          `json:"error_message,omitempty"`
// }

// // PromotionUsageResponse represents promotion usage details
// type PromotionUsageResponse struct {
// 	ID             uuid.UUID       `json:"id"`
// 	UserID         uuid.UUID       `json:"user_id"`
// 	OrderID        uuid.UUID       `json:"order_id"`
// 	OrderNumber    string          `json:"order_number"`
// 	DiscountAmount decimal.Decimal `json:"discount_amount"`
// 	UsedAt         time.Time       `json:"used_at"`
// }

// // ToResponse converts Promotion to PromotionResponse
// func (p *Promotion) ToResponse() *PromotionResponse {
// 	resp := &PromotionResponse{
// 		ID:                    p.ID,
// 		Code:                  p.Code,
// 		Name:                  p.Name,
// 		Description:           p.Description,
// 		DiscountType:          p.DiscountType,
// 		DiscountValue:         p.DiscountValue,
// 		MaxDiscountAmount:     p.MaxDiscountAmount,
// 		MinOrderAmount:        p.MinOrderAmount,
// 		ApplicableCategoryIDs: []string(p.ApplicableCategoryIDs),
// 		FirstOrderOnly:        p.FirstOrderOnly,
// 		MaxUses:               p.MaxUses,
// 		MaxUsesPerUser:        p.MaxUsesPerUser,
// 		CurrentUses:           p.CurrentUses,
// 		StartsAt:              p.StartsAt,
// 		ExpiresAt:             p.ExpiresAt,
// 		IsActive:              p.IsActive,
// 		IsValid:               p.IsValidNow(),
// 		CreatedAt:             p.CreatedAt,
// 		UpdatedAt:             p.UpdatedAt,
// 	}

// 	// Calculate remaining uses
// 	if p.MaxUses != nil {
// 		remaining := *p.MaxUses - p.CurrentUses
// 		if remaining < 0 {
// 			remaining = 0
// 		}
// 		resp.RemainingUses = &remaining
// 	}

// 	return resp
// }

// // ToListResponse converts Promotion to PromotionListResponse
// func (p *Promotion) ToListResponse() *PromotionListResponse {
// 	return &PromotionListResponse{
// 		ID:            p.ID,
// 		Code:          p.Code,
// 		Name:          p.Name,
// 		DiscountType:  p.DiscountType,
// 		DiscountValue: p.DiscountValue,
// 		CurrentUses:   p.CurrentUses,
// 		MaxUses:       p.MaxUses,
// 		StartsAt:      p.StartsAt,
// 		ExpiresAt:     p.ExpiresAt,
// 		IsActive:      p.IsActive,
// 	}
// }

// // IsValidNow checks if promotion is currently valid
// func (p *Promotion) IsValidNow() bool {
// 	now := time.Now()
// 	return p.IsActive &&
// 		now.After(p.StartsAt) &&
// 		now.Before(p.ExpiresAt) &&
// 		(p.MaxUses == nil || p.CurrentUses < *p.MaxUses)
// }

// // IsExpired checks if promotion has expired
// func (p *Promotion) IsExpired() bool {
// 	return time.Now().After(p.ExpiresAt)
// }

// // IsStarted checks if promotion has started
// func (p *Promotion) IsStarted() bool {
// 	return time.Now().After(p.StartsAt)
// }

// // HasReachedMaxUses checks if promotion has reached max uses
// func (p *Promotion) HasReachedMaxUses() bool {
// 	if p.MaxUses == nil {
// 		return false
// 	}
// 	return p.CurrentUses >= *p.MaxUses
// }

// // CalculateDiscount calculates discount amount for given order amount
// func (p *Promotion) CalculateDiscount(orderAmount decimal.Decimal) decimal.Decimal {
// 	if orderAmount.LessThan(p.MinOrderAmount) {
// 		return decimal.Zero
// 	}

// 	var discount decimal.Decimal

// 	if p.DiscountType == string(DiscountTypePercentage) {
// 		// Percentage discount
// 		discount = orderAmount.Mul(p.DiscountValue).Div(decimal.NewFromInt(100))

// 		// Apply max discount cap if set
// 		if p.MaxDiscountAmount != nil && discount.GreaterThan(*p.MaxDiscountAmount) {
// 			discount = *p.MaxDiscountAmount
// 		}
// 	} else if p.DiscountType == string(DiscountTypeFixed) {
// 		// Fixed amount discount
// 		discount = p.DiscountValue

// 		// Discount cannot exceed order amount
// 		if discount.GreaterThan(orderAmount) {
// 			discount = orderAmount
// 		}
// 	}

// 	return discount
// }

// // IsApplicableToCategories checks if promotion applies to given category IDs
// func (p *Promotion) IsApplicableToCategories(categoryIDs []uuid.UUID) bool {
// 	// If no category restriction, applicable to all
// 	if len(p.ApplicableCategoryIDs) == 0 {
// 		return true
// 	}

// 	// Check if any order category matches promotion categories
// 	applicableMap := make(map[string]bool)
// 	for _, catID := range p.ApplicableCategoryIDs {
// 		applicableMap[catID] = true
// 	}

// 	for _, catID := range categoryIDs {
// 		if applicableMap[catID.String()] {
// 			return true
// 		}
// 	}

// 	return false
// }

// // Validate validates promotion data
// func (p *Promotion) Validate() error {
// 	// Validate discount type
// 	discountType := DiscountType(p.DiscountType)
// 	if !discountType.IsValid() {
// 		return ErrInvalidDiscountType
// 	}

// 	// Validate discount value
// 	if p.DiscountValue.LessThanOrEqual(decimal.Zero) {
// 		return ErrInvalidDiscountValue
// 	}

// 	// For percentage, value must be <= 100
// 	if p.DiscountType == string(DiscountTypePercentage) &&
// 		p.DiscountValue.GreaterThan(decimal.NewFromInt(100)) {
// 		return ErrPercentageTooHigh
// 	}

// 	// Validate dates
// 	if !p.ExpiresAt.After(p.StartsAt) {
// 		return ErrInvalidDateRange
// 	}

// 	// Validate min order amount
// 	if p.MinOrderAmount.LessThan(decimal.Zero) {
// 		return ErrInvalidMinOrderAmount
// 	}

// 	return nil
// }

// // NormalizeCode normalizes promotion code to uppercase
// func (p *Promotion) NormalizeCode() {
// 	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
// }

// // Custom errors for promotion operations
// var (
//
//	ErrInvalidDiscountType        = errors.New("discount_type must be 'percentage' or 'fixed'")
//	ErrInvalidDiscountValue       = errors.New("discount_value must be > 0")
//	ErrPercentageTooHigh          = errors.New("percentage discount cannot exceed 100")
//	ErrInvalidDateRange           = errors.New("expires_at must be after starts_at")
//	ErrInvalidMinOrderAmount      = errors.New("min_order_amount must be >= 0")
//	ErrPromoCodeNotFound          = errors.New("promo code not found")
//	ErrPromoCodeExpired           = errors.New("promo code has expired")
//	ErrPromoCodeNotStarted        = errors.New("promo code is not yet active")
//	ErrPromoCodeMaxUsesReached    = errors.New("promo code has reached maximum uses")
//	ErrPromoCodeUserLimitReached  = errors.New("you have reached the usage limit for this promo code")
//	ErrPromoMinOrderNotMet        = errors.New("order amount does not meet minimum requirement")
//	ErrPromoNotApplicableCategory = errors.New("promo code is not applicable to items in this order")
//	ErrPromoFirstOrderOnly        = errors.New("promo code is only valid for first order")
//
// )
