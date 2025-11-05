package address

import (
	"errors"
	"fmt"
	"net/http"
)

// AddressError định nghĩa base error cho address domain
type AddressError struct {
	Code    string
	Message string
	Err     error
}

// Error implements error interface
func (e *AddressError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap allows error wrapping compatibility
func (e *AddressError) Unwrap() error {
	return e.Err
}

// ============================================
// ADDRESS-SPECIFIC ERROR DEFINITIONS
// ============================================

// ErrAddressNotFound - Address không tìm thấy
var ErrAddressNotFound = &AddressError{
	Code:    "ADDRESS_NOT_FOUND",
	Message: "Address not found",
}

// ErrAddressNotBelongToUser - Address không thuộc user
var ErrAddressNotBelongToUser = &AddressError{
	Code:    "ADDRESS_NOT_BELONG_TO_USER",
	Message: "Address does not belong to this user",
}

// ErrInvalidAddressID - Address ID không hợp lệ
var ErrInvalidAddressID = &AddressError{
	Code:    "INVALID_ADDRESS_ID",
	Message: "Invalid address ID format",
}

// ErrInvalidUserID - User ID không hợp lệ
var ErrInvalidUserID = &AddressError{
	Code:    "INVALID_USER_ID",
	Message: "Invalid user ID format",
}

// ErrInvalidPhone - Phone không hợp lệ
var ErrInvalidPhone = &AddressError{
	Code:    "INVALID_PHONE",
	Message: "Phone format is invalid for Vietnam",
}

// ErrInvalidRecipientName - Recipient name không hợp lệ
var ErrInvalidRecipientName = &AddressError{
	Code:    "INVALID_RECIPIENT_NAME",
	Message: "Recipient name is invalid or empty",
}

// ErrInvalidProvince - Province không hợp lệ
var ErrInvalidProvince = &AddressError{
	Code:    "INVALID_PROVINCE",
	Message: "Province is invalid or empty",
}

// ErrInvalidDistrict - District không hợp lệ
var ErrInvalidDistrict = &AddressError{
	Code:    "INVALID_DISTRICT",
	Message: "District is invalid or empty",
}

// ErrInvalidWard - Ward không hợp lệ
var ErrInvalidWard = &AddressError{
	Code:    "INVALID_WARD",
	Message: "Ward is invalid or empty",
}

// ErrInvalidStreet - Street không hợp lệ
var ErrInvalidStreet = &AddressError{
	Code:    "INVALID_STREET",
	Message: "Street address is invalid or empty",
}

// ErrInvalidAddressType - Address type không hợp lệ
var ErrInvalidAddressType = &AddressError{
	Code:    "INVALID_ADDRESS_TYPE",
	Message: "Address type must be: home, office, or other",
}

// ErrCannotUnsetOnlyDefault - Không thể unset default khi chỉ có 1 address
var ErrCannotUnsetOnlyDefault = &AddressError{
	Code:    "CANNOT_UNSET_ONLY_DEFAULT",
	Message: "Cannot unset default address when it's the only address",
}

// ErrUserHasNoAddress - User không có address
var ErrUserHasNoAddress = &AddressError{
	Code:    "USER_HAS_NO_ADDRESS",
	Message: "User has no addresses",
}

// ErrCreateAddress - Lỗi khi tạo address
var ErrCreateAddress = &AddressError{
	Code:    "CREATE_ADDRESS_ERROR",
	Message: "Failed to create address",
}

// ErrUpdateAddress - Lỗi khi update address
var ErrUpdateAddress = &AddressError{
	Code:    "UPDATE_ADDRESS_ERROR",
	Message: "Failed to update address",
}

// ErrDeleteAddress - Lỗi khi xóa address
var ErrDeleteAddress = &AddressError{
	Code:    "DELETE_ADDRESS_ERROR",
	Message: "Failed to delete address",
}

// ============================================
// ERROR FACTORY FUNCTIONS
// ============================================

// NewAddressNotFound tạo error "address not found"
func NewAddressNotFound() *AddressError {
	return &AddressError{
		Code:    "ADDRESS_NOT_FOUND",
		Message: "Address not found",
	}
}

// NewAddressNotBelongToUser tạo error "address not belong to user"
func NewAddressNotBelongToUser(addressID, userID string) *AddressError {
	return &AddressError{
		Code:    "ADDRESS_NOT_BELONG_TO_USER",
		Message: fmt.Sprintf("Address %s does not belong to user %s", addressID, userID),
	}
}

// NewInvalidAddressID tạo error "invalid address ID"
func NewInvalidAddressID(id string) *AddressError {
	return &AddressError{
		Code:    "INVALID_ADDRESS_ID",
		Message: fmt.Sprintf("Invalid address ID: %s", id),
	}
}

// NewInvalidUserID tạo error "invalid user ID"
func NewInvalidUserID(id string) *AddressError {
	return &AddressError{
		Code:    "INVALID_USER_ID",
		Message: fmt.Sprintf("Invalid user ID: %s", id),
	}
}

// NewInvalidPhone tạo error "invalid phone"
func NewInvalidPhone(phone string) *AddressError {
	return &AddressError{
		Code:    "INVALID_PHONE",
		Message: fmt.Sprintf("Phone format is invalid: %s (expected: 0xxx-xxx-xxx or +84-xxx-xxx-xxx)", phone),
	}
}

// NewInvalidRecipientName tạo error "invalid recipient name"
func NewInvalidRecipientName(name string) *AddressError {
	return &AddressError{
		Code:    "INVALID_RECIPIENT_NAME",
		Message: fmt.Sprintf("Recipient name is invalid: %s", name),
	}
}

// NewInvalidProvince tạo error "invalid province"
func NewInvalidProvince(province string) *AddressError {
	return &AddressError{
		Code:    "INVALID_PROVINCE",
		Message: fmt.Sprintf("Province is invalid or empty: %s", province),
	}
}

// NewInvalidDistrict tạo error "invalid district"
func NewInvalidDistrict(district string) *AddressError {
	return &AddressError{
		Code:    "INVALID_DISTRICT",
		Message: fmt.Sprintf("District is invalid or empty: %s", district),
	}
}

// NewInvalidWard tạo error "invalid ward"
func NewInvalidWard(ward string) *AddressError {
	return &AddressError{
		Code:    "INVALID_WARD",
		Message: fmt.Sprintf("Ward is invalid or empty: %s", ward),
	}
}

// NewInvalidStreet tạo error "invalid street"
func NewInvalidStreet(street string) *AddressError {
	return &AddressError{
		Code:    "INVALID_STREET",
		Message: fmt.Sprintf("Street address is invalid or empty: %s", street),
	}
}

// NewInvalidAddressType tạo error "invalid address type"
func NewInvalidAddressType(addressType string) *AddressError {
	return &AddressError{
		Code:    "INVALID_ADDRESS_TYPE",
		Message: fmt.Sprintf("Address type is invalid: %s (expected: home, office, other)", addressType),
	}
}

// NewCannotUnsetOnlyDefault tạo error "cannot unset only default"
func NewCannotUnsetOnlyDefault() *AddressError {
	return &AddressError{
		Code:    "CANNOT_UNSET_ONLY_DEFAULT",
		Message: "Cannot unset default address when it's the only address for this user",
	}
}

// NewUserHasNoAddress tạo error "user has no address"
func NewUserHasNoAddress(userID string) *AddressError {
	return &AddressError{
		Code:    "USER_HAS_NO_ADDRESS",
		Message: fmt.Sprintf("User %s has no addresses", userID),
	}
}

// NewCreateAddressError tạo error "create failed"
func NewCreateAddressError(err error) *AddressError {
	return &AddressError{
		Code:    "CREATE_ADDRESS_ERROR",
		Message: "Failed to create address",
		Err:     err,
	}
}

// NewUpdateAddressError tạo error "update failed"
func NewUpdateAddressError(err error) *AddressError {
	return &AddressError{
		Code:    "UPDATE_ADDRESS_ERROR",
		Message: "Failed to update address",
		Err:     err,
	}
}

// NewDeleteAddressError tạo error "delete failed"
func NewDeleteAddressError(err error) *AddressError {
	return &AddressError{
		Code:    "DELETE_ADDRESS_ERROR",
		Message: "Failed to delete address",
		Err:     err,
	}
}

// ============================================
// ERROR CHECKING FUNCTIONS
// ============================================

// IsAddressNotFound kiểm tra có phải "not found" error
func IsAddressNotFound(err error) bool {
	var addrErr *AddressError
	return errors.As(err, &addrErr) && addrErr.Code == "ADDRESS_NOT_FOUND"
}

// IsAddressNotBelongToUser kiểm tra có phải "not belong to user" error
func IsAddressNotBelongToUser(err error) bool {
	var addrErr *AddressError
	return errors.As(err, &addrErr) && addrErr.Code == "ADDRESS_NOT_BELONG_TO_USER"
}

// IsCannotUnsetOnlyDefault kiểm tra có phải "cannot unset only default" error
func IsCannotUnsetOnlyDefault(err error) bool {
	var addrErr *AddressError
	return errors.As(err, &addrErr) && addrErr.Code == "CANNOT_UNSET_ONLY_DEFAULT"
}

// IsDomainError kiểm tra có phải AddressError
func IsDomainError(err error) bool {
	var addrErr *AddressError
	return errors.As(err, &addrErr)
}

// GetErrorCode lấy error code từ error
func GetErrorCode(err error) string {
	var addrErr *AddressError
	if errors.As(err, &addrErr) {
		return addrErr.Code
	}
	return "UNKNOWN_ERROR"
}

// GetErrorMessage lấy error message từ error
func GetErrorMessage(err error) string {
	var addrErr *AddressError
	if errors.As(err, &addrErr) {
		return addrErr.Message
	}
	return err.Error()
}
func MapErrorToHTTP(err error) (int, string, interface{}) {
	if err == nil {
		return http.StatusOK, "Success", nil
	}

	switch {
	case IsAddressNotFound(err):
		return http.StatusNotFound, "Address not found", GetErrorCode(err)

	case IsAddressNotBelongToUser(err):
		return http.StatusForbidden, GetErrorMessage(err), GetErrorCode(err)

	case IsCannotUnsetOnlyDefault(err):
		return http.StatusConflict, GetErrorMessage(err), GetErrorCode(err)

	case IsDomainError(err):
		addrErr := err.(*AddressError)
		switch addrErr.Code {
		case "INVALID_ADDRESS_ID", "INVALID_USER_ID", "INVALID_PHONE", "INVALID_RECIPIENT_NAME",
			"INVALID_PROVINCE", "INVALID_DISTRICT", "INVALID_WARD", "INVALID_STREET", "INVALID_ADDRESS_TYPE":
			return http.StatusBadRequest, GetErrorMessage(err), GetErrorCode(err)
		case "USER_HAS_NO_ADDRESS":
			return http.StatusNotFound, GetErrorMessage(err), GetErrorCode(err)
		default:
			return http.StatusInternalServerError, GetErrorMessage(err), GetErrorCode(err)
		}

	default:
		return http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR"
	}
}

// GetErrorResponse trả về HTTP response structure cho error
func GetErrorResponse(err error) (statusCode int, message string, errorCode string) {
	statusCode, message, code := MapErrorToHTTP(err)
	if codeStr, ok := code.(string); ok {
		return statusCode, message, codeStr
	}
	return statusCode, message, GetErrorCode(err)
}
