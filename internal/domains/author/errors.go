package author

import "errors"

var (
	// Validation Errors
	ErrInvalidName = errors.New("author name is invalid")
	ErrNameTooLong = errors.New("author name exceeds maximum length")
	ErrInvalidSlug = errors.New("author slug is invalid")
	ErrBioTooLong  = errors.New("biography exceeds maximum length")

	// Business Rule Errors
	ErrAuthorNotFound  = errors.New("author not found")
	ErrDuplicateSlug   = errors.New("author with this slug already exists")
	ErrAuthorHasBooks  = errors.New("cannot delete author with linked books")
	ErrVersionMismatch = errors.New("author version mismatch - conflict detected")

	// Database Errors
	ErrDatabaseConnection = errors.New("database connection error")
	ErrDatabaseQuery      = errors.New("database query error")
)

// ErrorResponse represents API error response format
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ToErrorCode converts error to API error code
func ToErrorCode(err error) string {
	switch err {
	case ErrAuthorNotFound:
		return "AUTHOR_NOT_FOUND"
	case ErrDuplicateSlug:
		return "DUPLICATE_SLUG"
	case ErrAuthorHasBooks:
		return "AUTHOR_HAS_BOOKS"
	case ErrVersionMismatch:
		return "VERSION_CONFLICT"
	case ErrInvalidName:
		return "INVALID_NAME"
	default:
		return "INTERNAL_ERROR"
	}
}

// ToHTTPStatus converts error to HTTP status code
func ToHTTPStatus(err error) int {
	switch err {
	case ErrAuthorNotFound:
		return 404
	case ErrDuplicateSlug:
		return 409
	case ErrVersionMismatch:
		return 409
	case ErrAuthorHasBooks:
		return 409
	case ErrInvalidName, ErrNameTooLong, ErrInvalidSlug, ErrBioTooLong:
		return 400
	default:
		return 500
	}
}
