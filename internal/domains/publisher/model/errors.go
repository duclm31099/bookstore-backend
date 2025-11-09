package model

import (
	"errors"
	"fmt"
	"net/http"
)

// PublisherError định nghĩa base error cho publisher domain
type PublisherError struct {
	Code    string // Error code duy nhất (VD: "PUBLISHER_NOT_FOUND")
	Message string // Human-readable message
	Err     error  // Underlying error
}

// Error implements error interface
func (e *PublisherError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap allows error wrapping compatibility
func (e *PublisherError) Unwrap() error {
	return e.Err
}

// ============================================
// DOMAIN-SPECIFIC ERROR DEFINITIONS
// ============================================

// ErrPublisherNotFound - Publisher không tìm thấy
var ErrPublisherNotFound = &PublisherError{
	Code:    "PUBLISHER_NOT_FOUND",
	Message: "Publisher not found",
}

// ErrPublisherSlugAlreadyExists - Slug đã tồn tại
var ErrPublisherSlugAlreadyExists = &PublisherError{
	Code:    "PUBLISHER_SLUG_ALREADY_EXISTS",
	Message: "Publisher slug already exists",
}

// ErrInvalidPublisherID - Publisher ID không hợp lệ
var ErrInvalidPublisherID = &PublisherError{
	Code:    "INVALID_PUBLISHER_ID",
	Message: "Invalid publisher ID format",
}

// ErrInvalidSlug - Slug không hợp lệ
var ErrInvalidSlug = &PublisherError{
	Code:    "INVALID_SLUG",
	Message: "Publisher slug is invalid or empty",
}

// ErrInvalidEmail - Email không hợp lệ
var ErrInvalidEmail = &PublisherError{
	Code:    "INVALID_EMAIL",
	Message: "Publisher email is invalid",
}

// ErrInvalidPublisherName - Publisher name không hợp lệ
var ErrInvalidPublisherName = &PublisherError{
	Code:    "INVALID_PUBLISHER_NAME",
	Message: "Publisher name is invalid or empty",
}

// ErrPublisherHasBooks - Publisher có books, không thể xóa
var ErrPublisherHasBooks = &PublisherError{
	Code:    "PUBLISHER_HAS_BOOKS",
	Message: "Cannot delete publisher with associated books",
}

// ErrInvalidPageParams - Page parameters không hợp lệ
var ErrInvalidPageParams = &PublisherError{
	Code:    "INVALID_PAGE_PARAMS",
	Message: "Invalid page or page size parameters",
}

// ErrCreatePublisher - Lỗi khi tạo publisher
var ErrCreatePublisher = &PublisherError{
	Code:    "CREATE_PUBLISHER_ERROR",
	Message: "Failed to create publisher",
}

// ErrUpdatePublisher - Lỗi khi update publisher
var ErrUpdatePublisher = &PublisherError{
	Code:    "UPDATE_PUBLISHER_ERROR",
	Message: "Failed to update publisher",
}

// ErrDeletePublisher - Lỗi khi xóa publisher
var ErrDeletePublisher = &PublisherError{
	Code:    "DELETE_PUBLISHER_ERROR",
	Message: "Failed to delete publisher",
}

// ============================================
// ERROR FACTORY FUNCTIONS
// ============================================

// NewPublisherNotFound tạo error "publisher not found"
func NewPublisherNotFound() *PublisherError {
	return &PublisherError{
		Code:    "PUBLISHER_NOT_FOUND",
		Message: "Publisher not found",
	}
}

// NewPublisherSlugAlreadyExists tạo error "slug already exists"
func NewPublisherSlugAlreadyExists(slug string) *PublisherError {
	return &PublisherError{
		Code:    "PUBLISHER_SLUG_ALREADY_EXISTS",
		Message: fmt.Sprintf("Publisher with slug '%s' already exists", slug),
	}
}

// NewInvalidPublisherID tạo error "invalid publisher ID"
func NewInvalidPublisherID(id string) *PublisherError {
	return &PublisherError{
		Code:    "INVALID_PUBLISHER_ID",
		Message: fmt.Sprintf("Invalid publisher ID: %s", id),
	}
}

// NewInvalidSlug tạo error "invalid slug"
func NewInvalidSlug(slug string) *PublisherError {
	return &PublisherError{
		Code:    "INVALID_SLUG",
		Message: fmt.Sprintf("Publisher slug is invalid: %s", slug),
	}
}

// NewInvalidEmail tạo error "invalid email"
func NewInvalidEmail(email string) *PublisherError {
	return &PublisherError{
		Code:    "INVALID_EMAIL",
		Message: fmt.Sprintf("Publisher email is invalid: %s", email),
	}
}

// NewInvalidPublisherName tạo error "invalid name"
func NewInvalidPublisherName(name string) *PublisherError {
	return &PublisherError{
		Code:    "INVALID_PUBLISHER_NAME",
		Message: fmt.Sprintf("Publisher name is invalid: %s", name),
	}
}

// NewPublisherHasBooks tạo error "publisher has books"
func NewPublisherHasBooks(publisherID string) *PublisherError {
	return &PublisherError{
		Code:    "PUBLISHER_HAS_BOOKS",
		Message: fmt.Sprintf("Cannot delete publisher %s with associated books", publisherID),
	}
}

// NewInvalidPageParams tạo error "invalid page params"
func NewInvalidPageParams(page, pageSize int) *PublisherError {
	return &PublisherError{
		Code:    "INVALID_PAGE_PARAMS",
		Message: fmt.Sprintf("Invalid pagination params: page=%d, pageSize=%d", page, pageSize),
	}
}

// NewCreatePublisherError tạo error "create failed" với underlying error
func NewCreatePublisherError(err error) *PublisherError {
	return &PublisherError{
		Code:    "CREATE_PUBLISHER_ERROR",
		Message: "Failed to create publisher",
		Err:     err,
	}
}

// NewCreatePublisherError tạo error "create failed" với underlying error
func NewListPublisherError(err error) *PublisherError {
	return &PublisherError{
		Code:    "LIST_PUBLISHER_ERROR",
		Message: "Failed to list publisher",
		Err:     err,
	}
}

// NewUpdatePublisherError tạo error "update failed" với underlying error
func NewUpdatePublisherError(err error) *PublisherError {
	return &PublisherError{
		Code:    "UPDATE_PUBLISHER_ERROR",
		Message: "Failed to update publisher",
		Err:     err,
	}
}

// NewDeletePublisherError tạo error "delete failed" với underlying error
func NewDeletePublisherError(err error) *PublisherError {
	return &PublisherError{
		Code:    "DELETE_PUBLISHER_ERROR",
		Message: "Failed to delete publisher",
		Err:     err,
	}
}

// ============================================
// ERROR CHECKING FUNCTIONS
// ============================================

// IsPublisherNotFound kiểm tra có phải "not found" error
func IsPublisherNotFound(err error) bool {
	var pubErr *PublisherError
	return errors.As(err, &pubErr) && pubErr.Code == "PUBLISHER_NOT_FOUND"
}

// IsPublisherSlugAlreadyExists kiểm tra có phải "slug exists" error
func IsPublisherSlugAlreadyExists(err error) bool {
	var pubErr *PublisherError
	return errors.As(err, &pubErr) && pubErr.Code == "PUBLISHER_SLUG_ALREADY_EXISTS"
}

// IsPublisherHasBooks kiểm tra có phải "has books" error
func IsPublisherHasBooks(err error) bool {
	var pubErr *PublisherError
	return errors.As(err, &pubErr) && pubErr.Code == "PUBLISHER_HAS_BOOKS"
}

// IsDomainError kiểm tra có phải PublisherError
func IsDomainError(err error) bool {
	var pubErr *PublisherError
	return errors.As(err, &pubErr)
}

// GetErrorCode lấy error code từ error
func GetErrorCode(err error) string {
	var pubErr *PublisherError
	if errors.As(err, &pubErr) {
		return pubErr.Code
	}
	return "UNKNOWN_ERROR"
}

// GetErrorMessage lấy error message từ error
func GetErrorMessage(err error) string {
	var pubErr *PublisherError
	if errors.As(err, &pubErr) {
		return pubErr.Message
	}
	return err.Error()
}

// ErrorMapping định nghĩa mapping từ domain error → HTTP status code
type ErrorMapping struct {
	StatusCode int    // HTTP status code
	Message    string // User-friendly message
}

// MapErrorToHTTP chuyển PublisherError sang HTTP response
func MapErrorToHTTP(err error) (int, string, interface{}) {
	if err == nil {
		return http.StatusOK, "Success", nil
	}

	// Map domain errors
	switch {
	case IsPublisherNotFound(err):
		return http.StatusNotFound, "Publisher not found", GetErrorCode(err)

	case IsPublisherSlugAlreadyExists(err):
		return http.StatusConflict, GetErrorMessage(err), GetErrorCode(err)

	case IsPublisherHasBooks(err):
		return http.StatusConflict, GetErrorMessage(err), GetErrorCode(err)

	case IsDomainError(err):
		// Validation/business logic errors
		pubErr := err.(*PublisherError)
		switch pubErr.Code {
		case "INVALID_PUBLISHER_ID", "INVALID_SLUG", "INVALID_EMAIL", "INVALID_PUBLISHER_NAME":
			return http.StatusBadRequest, GetErrorMessage(err), GetErrorCode(err)
		case "INVALID_PAGE_PARAMS":
			return http.StatusBadRequest, GetErrorMessage(err), GetErrorCode(err)
		default:
			return http.StatusInternalServerError, GetErrorMessage(err), GetErrorCode(err)
		}

	default:
		// Unknown errors
		return http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR"
	}
}

// GetErrorResponse trả về HTTP response structure cho error
func GetErrorResponse(err error) (statusCode int, message string, errorCode string) {
	statusCode, message, code := MapErrorToHTTP(err)
	if code, ok := code.(string); ok {
		return statusCode, message, code
	}
	return statusCode, message, GetErrorCode(err)
}
