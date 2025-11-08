package service

import (
	"bookstore-backend/internal/domains/promotion/model"

	"github.com/shopspring/decimal"
)

// DiscountCalculator xử lý logic tính toán discount
type DiscountCalculator struct{}

// NewDiscountCalculator tạo instance mới
func NewDiscountCalculator() *DiscountCalculator {
	return &DiscountCalculator{}
}

// Calculate tính toán số tiền giảm giá dựa trên promotion và subtotal
//
// Business Logic:
// 1. Percentage Discount:
//   - Tính: discount = subtotal × (discount_value / 100)
//   - Nếu có max_discount_amount: discount = min(discount, max_discount_amount)
//   - Làm tròn đến VND (không có phần thập phân)
//
// 2. Fixed Discount:
//   - discount = discount_value
//   - Không được vượt quá subtotal: discount = min(discount, subtotal)
//
// Returns: Số tiền giảm giá (đã làm tròn)
func (c *DiscountCalculator) Calculate(promo *model.Promotion, subtotal decimal.Decimal) decimal.Decimal {
	var discount decimal.Decimal

	switch promo.DiscountType {
	case model.DiscountTypePercentage:
		// Tính % giảm giá
		// VD: 400,000 × 20 / 100 = 80,000
		discount = subtotal.Mul(promo.DiscountValue).Div(decimal.NewFromInt(100))

		// Áp dụng cap tối đa nếu có
		if promo.MaxDiscountAmount != nil {
			if discount.GreaterThan(*promo.MaxDiscountAmount) {
				discount = *promo.MaxDiscountAmount
			}
		}

	case model.DiscountTypeFixed:
		// Giảm số tiền cố định
		discount = promo.DiscountValue

		// Không được vượt quá tổng tiền đơn hàng
		// VD: Đơn 50k, discount 100k → chỉ giảm 50k
		if discount.GreaterThan(subtotal) {
			discount = subtotal
		}

	default:
		// Không hỗ trợ discount type này
		return decimal.Zero
	}

	// Làm tròn đến VND (không có xu)
	// Round mode: ROUND_HALF_UP (>= 0.5 làm tròn lên)
	return discount.Round(0)
}

// CalculateWithBreakdown tính toán chi tiết từng bước (dùng cho debugging/logging)
func (c *DiscountCalculator) CalculateWithBreakdown(promo *model.Promotion, subtotal decimal.Decimal) DiscountBreakdown {
	breakdown := DiscountBreakdown{
		Subtotal:     subtotal,
		DiscountType: string(promo.DiscountType),
	}

	switch promo.DiscountType {
	case model.DiscountTypePercentage:
		rawDiscount := subtotal.Mul(promo.DiscountValue).Div(decimal.NewFromInt(100))
		breakdown.RawDiscount = rawDiscount

		if promo.MaxDiscountAmount != nil && rawDiscount.GreaterThan(*promo.MaxDiscountAmount) {
			breakdown.FinalDiscount = *promo.MaxDiscountAmount
			breakdown.Capped = true
			breakdown.CapReason = "max_discount_amount"
		} else {
			breakdown.FinalDiscount = rawDiscount
		}

	case model.DiscountTypeFixed:
		breakdown.RawDiscount = promo.DiscountValue

		if promo.DiscountValue.GreaterThan(subtotal) {
			breakdown.FinalDiscount = subtotal
			breakdown.Capped = true
			breakdown.CapReason = "exceeds_subtotal"
		} else {
			breakdown.FinalDiscount = promo.DiscountValue
		}
	}

	breakdown.FinalDiscount = breakdown.FinalDiscount.Round(0)
	return breakdown
}

// DiscountBreakdown chứa chi tiết tính toán (dùng cho logging/debugging)
type DiscountBreakdown struct {
	Subtotal      decimal.Decimal `json:"subtotal"`
	DiscountType  string          `json:"discount_type"`
	RawDiscount   decimal.Decimal `json:"raw_discount"`         // Trước khi cap
	FinalDiscount decimal.Decimal `json:"final_discount"`       // Sau khi cap
	Capped        bool            `json:"capped"`               // Có bị cap không
	CapReason     string          `json:"cap_reason,omitempty"` // Lý do cap
}
