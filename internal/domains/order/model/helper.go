package model

import (
	"github.com/shopspring/decimal"
)

// =====================================================
// CALCULATION HELPERS
// =====================================================

// CalculateOrderAmounts calculates all order amounts
// Returns: subtotal, discount, shipping, cod_fee, tax, total
func CalculateOrderAmounts(
	itemsSubtotal decimal.Decimal,
	discountPercent decimal.Decimal,
	maxDiscount decimal.Decimal,
	fixedDiscount decimal.Decimal,
	promoType string, // "percentage" or "fixed"
	isCOD bool,
) (subtotal, discount, shipping, codFee, tax, total decimal.Decimal) {

	subtotal = itemsSubtotal

	// Calculate discount based on promo type
	if promoType == "percentage" {
		discount = subtotal.Mul(discountPercent).Div(decimal.NewFromInt(100))
		if maxDiscount.GreaterThan(decimal.Zero) && discount.GreaterThan(maxDiscount) {
			discount = maxDiscount
		}
	} else if promoType == "fixed" {
		discount = fixedDiscount
	} else {
		discount = decimal.Zero
	}

	// Shipping fee (fixed 15,000 VND)
	shipping = decimal.NewFromInt(ShippingFee)

	// COD fee (15,000 VND if payment method is COD)
	if isCOD {
		codFee = decimal.NewFromInt(CODFee)
	} else {
		codFee = decimal.Zero
	}

	// Tax (0% for now)
	tax = decimal.Zero

	// Total = subtotal - discount + shipping + cod_fee + tax
	total = subtotal.Sub(discount).Add(shipping).Add(codFee).Add(tax)

	// Ensure total is not negative
	if total.LessThan(decimal.Zero) {
		total = decimal.Zero
	}

	return subtotal, discount, shipping, codFee, tax, total
}

// GetWarehouseCodeByProvince returns warehouse code based on province
func GetWarehouseCodeByProvince(province string) string {
	if code, exists := ProvinceWarehouseMap[province]; exists {
		return code
	}
	return DefaultWarehouseCode
}
