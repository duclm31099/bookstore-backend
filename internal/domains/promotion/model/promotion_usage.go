package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PromotionUsage ghi lại lịch sử sử dụng promotion
type PromotionUsage struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	PromotionID    uuid.UUID       `db:"promotion_id" json:"promotion_id"`
	UserID         uuid.UUID       `db:"user_id" json:"user_id"`
	OrderID        uuid.UUID       `db:"order_id" json:"order_id"`
	DiscountAmount decimal.Decimal `db:"discount_amount" json:"discount_amount"` // Số tiền đã giảm
	UsedAt         time.Time       `db:"used_at" json:"used_at"`
	Version        int             `db:"version" json:"version"`
}

// PromotionUsageWithDetails bao gồm thông tin chi tiết user và order
type PromotionUsageWithDetails struct {
	PromotionUsage

	// User info
	UserEmail    string `db:"user_email" json:"user_email"`
	UserFullName string `db:"user_full_name" json:"user_full_name"`

	// Order info
	OrderNumber string          `db:"order_number" json:"order_number"`
	OrderTotal  decimal.Decimal `db:"order_total" json:"order_total"`
	OrderStatus string          `db:"order_status" json:"order_status"`
}
