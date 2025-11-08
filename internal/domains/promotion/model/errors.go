package model

import "errors"

var (
	ErrPromotionInactive       = errors.New("promotion is not active")
	ErrPromotionExhausted      = errors.New("promotion usage limit reached")
	ErrPromotionFirstOrderOnly = errors.New("promotion is for first order only")
	ErrOrderAmountTooLow       = errors.New("order amount is below minimum required")
	ErrUserLimitExceeded       = errors.New("user has exceeded maximum uses for this promotion")
	ErrInvalidDiscountType     = errors.New("invalid discount type")
)

type ErrorCode string

const (
	// Promotion validation errors (400)
	ErrCodePromoNotFound              ErrorCode = "PROMO_NOT_FOUND"               // 404
	ErrCodePromoNotStarted            ErrorCode = "PROMO_NOT_STARTED"             // 400
	ErrCodePromoExpired               ErrorCode = "PROMO_EXPIRED"                 // 400
	ErrCodePromoUsageLimitExceeded    ErrorCode = "PROMO_USAGE_LIMIT_EXCEEDED"    // 400
	ErrCodePromoUserLimitExceeded     ErrorCode = "PROMO_USER_LIMIT_EXCEEDED"     // 400
	ErrCodePromoMinOrderNotMet        ErrorCode = "PROMO_MIN_ORDER_NOT_MET"       // 400
	ErrCodePromoCategoryNotApplicable ErrorCode = "PROMO_CATEGORY_NOT_APPLICABLE" // 400
	ErrCodePromoFirstOrderOnly        ErrorCode = "PROMO_FIRST_ORDER_ONLY"        // 400

	// Admin operation errors
	ErrCodePromoDuplicateCode  ErrorCode = "VAL_DUPLICATE_CODE"           // 400
	ErrCodePromoUpdateConflict ErrorCode = "BIZ_UPDATE_CONFLICT"          // 409
	ErrCodePromoCannotDelete   ErrorCode = "BIZ_CANNOT_DELETE_USED_PROMO" // 400
	ErrCodePromoDuplicateUsage ErrorCode = "BIZ_DUPLICATE_USAGE"          // 409

	// Validation errors (400)
	ErrCodeValidationFailed ErrorCode = "VAL_INVALID_INPUT"    // 400
	ErrCodeInvalidSubtotal  ErrorCode = "VAL_INVALID_SUBTOTAL" // 400

	// System errors (500)
	ErrCodeInternalError ErrorCode = "SYS_INTERNAL_ERROR" // 500
)

type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Predefined errors
var (
	ErrPromotionNotFound = &AppError{
		Code:       ErrCodePromoNotFound,
		Message:    "Mã giảm giá không tồn tại hoặc đã bị vô hiệu hóa",
		HTTPStatus: 404,
	}

	ErrPromotionExpired = &AppError{
		Code:       ErrCodePromoExpired,
		Message:    "Mã giảm giá đã hết hạn",
		HTTPStatus: 400,
	}

	// ... định nghĩa các errors khác
)
