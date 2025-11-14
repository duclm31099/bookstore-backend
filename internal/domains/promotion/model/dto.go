package model

import (
	"errors"
	"regexp"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ValidatePromotionRequest - Request để validate promotion code
type ValidatePromotionRequest struct {
	Code      string          `json:"code"`
	CartItems []CartItem      `json:"cart_items"`
	Subtotal  decimal.Decimal `json:"subtotal"`
	UserID    *uuid.UUID      `json:"-"` // Từ JWT token, không nhận từ request body
}

// CartItem đại diện cho một item trong giỏ hàng
type CartItem struct {
	BookID     uuid.UUID       `json:"book_id"`
	CategoryID uuid.UUID       `json:"category_id"`
	Price      decimal.Decimal `json:"price"`
	Quantity   int             `json:"quantity"`
}

// Validate validates ValidatePromotionRequest
func (r ValidatePromotionRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Code,
			validation.Required.Error("Mã khuyến mãi không được để trống"),
			validation.Length(3, 50).Error("Mã khuyến mãi phải từ 3-50 ký tự"),
		),
		validation.Field(&r.CartItems,
			validation.Required.Error("Giỏ hàng không được trống"),
			validation.Length(1, 100).Error("Giỏ hàng có từ 1-100 sản phẩm"),
		),
		validation.Field(&r.Subtotal,
			validation.Required.Error("Tổng tiền không được để trống"),
			validation.Min(decimal.NewFromInt(0)).Error("Tổng tiền phải >= 0"),
		),
	)
}

// NormalizeCode chuyển code về uppercase
func (r *ValidatePromotionRequest) NormalizeCode() {
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
}

// CalculateSubtotal tính lại subtotal từ cart items
func (r ValidatePromotionRequest) CalculateSubtotal() decimal.Decimal {
	total := decimal.Zero
	for _, item := range r.CartItems {
		itemTotal := item.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		total = total.Add(itemTotal)
	}
	return total
}

// Validate validates CartItem
func (c CartItem) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.BookID, validation.Required, is.UUIDv4),
		validation.Field(&c.CategoryID, validation.Required, is.UUIDv4),
		validation.Field(&c.Price, validation.Min(decimal.NewFromInt(0))),
		validation.Field(&c.Quantity, validation.Min(1), validation.Max(100)),
	)
}

// -------------------------------------------------------------------
// ADMIN REQUESTS
// -------------------------------------------------------------------

// CreatePromotionRequest - Request để tạo promotion mới
type CreatePromotionRequest struct {
	Code                  string      `json:"code"`
	Name                  string      `json:"name"`
	Description           *string     `json:"description"`
	DiscountType          string      `json:"discount_type"`
	DiscountValue         float64     `json:"discount_value"`
	MaxDiscountAmount     *float64    `json:"max_discount_amount"`
	MinOrderAmount        float64     `json:"min_order_amount"`
	ApplicableCategoryIDs []uuid.UUID `json:"applicable_category_ids"`
	FirstOrderOnly        bool        `json:"first_order_only"`
	MaxUses               *int        `json:"max_uses"`
	MaxUsesPerUser        int         `json:"max_uses_per_user"`
	StartsAt              string      `json:"starts_at"` // RFC3339 format
	ExpiresAt             string      `json:"expires_at"`
	IsActive              bool        `json:"is_active"`
}

// Validate validates CreatePromotionRequest
func (r CreatePromotionRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Code,
			validation.Required.Error("Mã khuyến mãi bắt buộc"),
			validation.Length(3, 50).Error("Mã khuyến mãi phải từ 3-50 ký tự"),
			validation.Match(regexp.MustCompile("^[A-Z0-9]+$")).Error("Mã chỉ được chứa chữ hoa và số"),
		),
		validation.Field(&r.Name,
			validation.Required.Error("Tên khuyến mãi bắt buộc"),
			validation.Length(3, 200).Error("Tên phải từ 3-200 ký tự"),
		),
		validation.Field(&r.Description,
			validation.When(r.Description != nil,
				validation.Length(0, 1000).Error("Mô tả không được vượt quá 1000 ký tự"),
			),
		),
		validation.Field(&r.DiscountType,
			validation.Required.Error("Loại giảm giá bắt buộc"),
			validation.In("percentage", "fixed").Error("Loại giảm giá phải là 'percentage' hoặc 'fixed'"),
		),
		validation.Field(&r.DiscountValue,
			validation.Required.Error("Giá trị giảm giá bắt buộc"),
			validation.Min(0.01).Error("Giá trị giảm giá phải > 0"),
			validation.By(r.validateDiscountValue),
		),
		validation.Field(&r.MaxDiscountAmount,
			validation.When(r.MaxDiscountAmount != nil,
				validation.Min(0.01).Error("Giá trị giảm tối đa phải > 0"),
			),
		),
		validation.Field(&r.MinOrderAmount,
			validation.Min(0.0).Error("Giá trị đơn hàng tối thiểu phải >= 0"),
		),
		validation.Field(&r.MaxUses,
			validation.When(r.MaxUses != nil,
				validation.Min(1).Error("Số lượt sử dụng tối đa phải >= 1"),
			),
		),
		validation.Field(&r.MaxUsesPerUser,
			validation.Min(1).Error("Số lượt sử dụng/user phải >= 1"),
		),
		validation.Field(&r.StartsAt,
			validation.Required.Error("Thời gian bắt đầu bắt buộc"),
			validation.Date("2006-01-02T15:04:05Z07:00").Error("Định dạng thời gian không hợp lệ (RFC3339)"),
		),
		validation.Field(&r.ExpiresAt,
			validation.Required.Error("Thời gian kết thúc bắt buộc"),
			validation.Date("2006-01-02T15:04:05Z07:00").Error("Định dạng thời gian không hợp lệ (RFC3339)"),
			validation.By(r.validateDateRange),
		),
	)
}

// validateDiscountValue kiểm tra percentage không vượt 100
func (r CreatePromotionRequest) validateDiscountValue(value interface{}) error {
	if r.DiscountType == "percentage" {
		if r.DiscountValue > 100 {
			return errors.New("giảm giá phần trăm không được vượt quá 100")
		}
	}
	return nil
}

// validateDateRange kiểm tra expires_at phải sau starts_at
func (r CreatePromotionRequest) validateDateRange(value interface{}) error {
	startsAt, err := time.Parse(time.RFC3339, r.StartsAt)
	if err != nil {
		return nil // Lỗi format đã được validate ở Field StartsAt
	}

	expiresAt, err := time.Parse(time.RFC3339, r.ExpiresAt)
	if err != nil {
		return nil // Lỗi format đã được validate ở Field ExpiresAt
	}

	if expiresAt.Before(startsAt) || expiresAt.Equal(startsAt) {
		return errors.New("thời gian kết thúc phải sau thời gian bắt đầu")
	}

	return nil
}

// NormalizeCode chuyển code về uppercase
func (r *CreatePromotionRequest) NormalizeCode() {
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
}

// UpdatePromotionRequest - Request để update promotion
type UpdatePromotionRequest struct {
	Name              *string          `json:"name"`
	Description       *string          `json:"description"`
	MaxDiscountAmount *decimal.Decimal `json:"max_discount_amount"`
	MinOrderAmount    *decimal.Decimal `json:"min_order_amount"`
	MaxUses           *int             `json:"max_uses"`
	MaxUsesPerUser    *int             `json:"max_uses_per_user"`
	StartsAt          *string          `json:"starts_at"`
	ExpiresAt         *string          `json:"expires_at"`
	IsActive          *bool            `json:"is_active"`
}

// ListPromotionsFilter - Filter cho list promotions (Admin)
type ListPromotionsFilter struct {
	Status   string `form:"status"` // active, expired, upcoming, all
	Search   string `form:"search"` // Tìm kiếm theo code/name
	Sort     string `form:"sort"`   // created_at_desc, expires_at_asc, usage_desc
	IsActive *bool  `form:"is_active"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
}

// Validate validates ListPromotionsFilter
func (f *ListPromotionsFilter) Validate() error {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Status == "" {
		f.Status = "active"
	}
	return validation.ValidateStruct(f,
		validation.Field(&f.Status, validation.In("active", "expired", "upcoming", "all")),
		validation.Field(&f.Sort, validation.In("", "created_at_desc", "expires_at_asc", "usage_desc", "name_asc")),
	)
}

// ApplyPromoRequest - Request để apply promo vào cart
type ApplyPromoRequest struct {
	Code string `json:"code"`
}

// Validate validates ApplyPromoRequest
func (r ApplyPromoRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Code,
			validation.Required.Error("Mã khuyến mãi không được để trống"),
			validation.Length(3, 50).Error("Mã khuyến mãi phải từ 3-50 ký tự"),
		),
	)
}

// ValidationResult - Kết quả validation promotion
type ValidationResult struct {
	IsValid             bool            `json:"is_valid"`
	Promotion           *PromotionInfo  `json:"promotion,omitempty"`
	DiscountAmount      decimal.Decimal `json:"discount_amount"`
	FinalAmount         decimal.Decimal `json:"final_amount"`
	Message             string          `json:"message"`
	RemainingGlobalUses *int            `json:"remaining_global_uses,omitempty"`
	UserRemainingUses   int             `json:"user_remaining_uses"`
}

// PromotionInfo - Thông tin promotion cho response
type PromotionInfo struct {
	ID                uuid.UUID        `json:"id"`
	Code              string           `json:"code"`
	Name              string           `json:"name"`
	Description       *string          `json:"description,omitempty"`
	DiscountType      string           `json:"discount_type"`
	DiscountValue     decimal.Decimal  `json:"discount_value"`
	MaxDiscountAmount *decimal.Decimal `json:"max_discount_amount,omitempty"`
	MinOrderAmount    decimal.Decimal  `json:"min_order_amount"`
	ExpiresAt         time.Time        `json:"expires_at"`
}

// PromotionListItem - Item trong danh sách promotions (Admin)
type PromotionListItem struct {
	ID                uuid.UUID        `json:"id"`
	Code              string           `json:"code"`
	Name              string           `json:"name"`
	DiscountType      string           `json:"discount_type"`
	DiscountValue     decimal.Decimal  `json:"discount_value"`
	MaxDiscountAmount *decimal.Decimal `json:"max_discount_amount,omitempty"`
	CurrentUses       int              `json:"current_uses"`
	MaxUses           *int             `json:"max_uses,omitempty"`
	UsageRate         *float64         `json:"usage_rate,omitempty"` // % sử dụng
	StartsAt          time.Time        `json:"starts_at"`
	ExpiresAt         time.Time        `json:"expires_at"`
	IsActive          bool             `json:"is_active"`
	Status            string           `json:"status"` // active, expired, upcoming, exhausted
}

// PromotionDetailResponse - Chi tiết promotion (Admin)
type PromotionDetailResponse struct {
	ID                    uuid.UUID        `json:"id"`
	Code                  string           `json:"code"`
	Name                  string           `json:"name"`
	Description           *string          `json:"description,omitempty"`
	DiscountType          string           `json:"discount_type"`
	DiscountValue         decimal.Decimal  `json:"discount_value"`
	MaxDiscountAmount     *decimal.Decimal `json:"max_discount_amount,omitempty"`
	MinOrderAmount        decimal.Decimal  `json:"min_order_amount"`
	ApplicableCategoryIDs []uuid.UUID      `json:"applicable_category_ids,omitempty"`
	FirstOrderOnly        bool             `json:"first_order_only"`
	MaxUses               *int             `json:"max_uses,omitempty"`
	MaxUsesPerUser        int              `json:"max_uses_per_user"`
	CurrentUses           int              `json:"current_uses"`
	UsageRate             *float64         `json:"usage_rate,omitempty"`
	StartsAt              time.Time        `json:"starts_at"`
	ExpiresAt             time.Time        `json:"expires_at"`
	IsActive              bool             `json:"is_active"`
	Version               int              `json:"version"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
	Stats                 *UsageStats      `json:"stats,omitempty"`
}

// UsageStats - Thống kê sử dụng promotion
type UsageStats struct {
	TotalUses               int             `json:"total_uses"`
	TotalDiscountGiven      decimal.Decimal `json:"total_discount_given"`
	AverageDiscountPerOrder decimal.Decimal `json:"average_discount_per_order"`
	UniqueUsers             int             `json:"unique_users"`
	RevenueImpact           decimal.Decimal `json:"revenue_impact"` // Âm
}

// UsageHistoryResponse - Lịch sử sử dụng promotion
type UsageHistoryResponse struct {
	Promotion    PromotionInfo              `json:"promotion"`
	Statistics   UsageStats                 `json:"statistics"`
	UsageHistory []PromotionUsageDetailItem `json:"usage_history"`
}

// PromotionUsageDetailItem - Chi tiết một lần sử dụng
type PromotionUsageDetailItem struct {
	ID             uuid.UUID       `json:"id"`
	User           UserInfo        `json:"user"`
	Order          OrderInfo       `json:"order"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	UsedAt         time.Time       `json:"used_at"`
}

// UserInfo - Thông tin user trong usage history
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
}

// OrderInfo - Thông tin order trong usage history
type OrderInfo struct {
	ID          uuid.UUID       `json:"id"`
	OrderNumber string          `json:"order_number"`
	Subtotal    decimal.Decimal `json:"subtotal"`
	Status      string          `json:"status"`
}
