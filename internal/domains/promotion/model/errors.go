package model

import "errors"

var (
	ErrPromotionNotFound       = errors.New("promotion not found")
	ErrPromotionExpired       = errors.New("promotion has expired")
	ErrPromotionInactive      = errors.New("promotion is not active")
	ErrPromotionExhausted     = errors.New("promotion usage limit reached")
	ErrPromotionFirstOrderOnly = errors.New("promotion is for first order only")
	ErrOrderAmountTooLow      = errors.New("order amount is below minimum required")
	ErrUserLimitExceeded      = errors.New("user has exceeded maximum uses for this promotion")
	ErrInvalidDiscountType    = errors.New("invalid discount type")
)