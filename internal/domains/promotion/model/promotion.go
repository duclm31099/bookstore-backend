package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DiscountType định nghĩa loại giảm giá
type DiscountType string

const (
	DiscountTypePercentage DiscountType = "percentage" // Giảm theo %
	DiscountTypeFixed      DiscountType = "fixed"      // Giảm số tiền cố định
)

// Promotion đại diện cho một chương trình khuyến mãi
type Promotion struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Code        string    `db:"code" json:"code"`                     // Mã khuyến mãi (SUMMER20)
	Name        string    `db:"name" json:"name"`                     // Tên hiển thị
	Description *string   `db:"description" json:"description,omitempty"` // Mô tả chi tiết
	
	// Cấu hình giảm giá
	DiscountType       DiscountType     `db:"discount_type" json:"discount_type"`
	DiscountValue      decimal.Decimal  `db:"discount_value" json:"discount_value"`           // 20 (cho 20%)
	MaxDiscountAmount  *decimal.Decimal `db:"max_discount_amount" json:"max_discount_amount,omitempty"` // Cap tối đa
	
	// Điều kiện áp dụng
	MinOrderAmount        decimal.Decimal `db:"min_order_amount" json:"min_order_amount"`
	ApplicableCategoryIDs []uuid.UUID     `db:"applicable_category_ids" json:"applicable_category_ids,omitempty"` // NULL = tất cả category
	FirstOrderOnly        bool            `db:"first_order_only" json:"first_order_only"`
	
	// Giới hạn sử dụng
	MaxUses        *int `db:"max_uses" json:"max_uses,omitempty"`             // NULL = không giới hạn
	MaxUsesPerUser int  `db:"max_uses_per_user" json:"max_uses_per_user"`     // Mặc định: 1
	CurrentUses    int  `db:"current_uses" json:"current_uses"`               // Số lần đã dùng
	
	// Thời gian hiệu lực
	StartsAt  time.Time `db:"starts_at" json:"starts_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	
	// Trạng thái
	IsActive bool `db:"is_active" json:"is_active"`
	Version  int  `db:"version" json:"version"` // Optimistic locking
	
	// Audit
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// IsExpired kiểm tra promotion đã hết hạn chưa
func (p *Promotion) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// IsNotStarted kiểm tra promotion chưa bắt đầu
func (p *Promotion) IsNotStarted() bool {
	return time.Now().Before(p.StartsAt)
}

// IsUsageLimitReached kiểm tra đã hết lượt sử dụng
func (p *Promotion) IsUsageLimitReached() bool {
	if p.MaxUses == nil {
		return false // Không giới hạn
	}
	return p.CurrentUses >= *p.MaxUses
}

// IsValidTimeWindow kiểm tra promotion đang trong thời gian hiệu lực
func (p *Promotion) IsValidTimeWindow() bool {
	now := time.Now()
	return now.After(p.StartsAt) && now.Before(p.ExpiresAt)
}

// RemainingUses tính số lượt còn lại (nil nếu không giới hạn)
func (p *Promotion) RemainingUses() *int {
	if p.MaxUses == nil {
		return nil
	}
	remaining := *p.MaxUses - p.CurrentUses
	if remaining < 0 {
		remaining = 0
	}
	return &remaining
}
