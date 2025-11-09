package model

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AddressType enum for address classification
type AddressType string

const (
	AddressTypeHome   AddressType = "home"
	AddressTypeOffice AddressType = "office"
	AddressTypeOther  AddressType = "other"
)

// Address represents a user delivery address
// Used across all layers (repository, service, handler)
type Address struct {
	ID            uuid.UUID   `db:"id"`
	UserID        uuid.UUID   `db:"user_id"`
	RecipientName string      `db:"recipient_name"`
	Phone         string      `db:"phone"`
	Province      string      `db:"province"`
	District      string      `db:"district"`
	Ward          string      `db:"ward"`
	Street        string      `db:"street"`
	AddressType   AddressType `db:"address_type"`
	IsDefault     bool        `db:"is_default"`
	Notes         string      `db:"notes"`
	CreatedAt     time.Time   `db:"created_at"`
	UpdatedAt     time.Time   `db:"updated_at"`
}

// AddressCreateRequest DTO for creating a new address
type AddressCreateRequest struct {
	RecipientName string `json:"recipient_name" binding:"required,min=2,max=255"`
	Phone         string `json:"phone" binding:"required"`
	Province      string `json:"province" binding:"required,min=1,max=100"`
	District      string `json:"district" binding:"required,min=1,max=100"`
	Ward          string `json:"ward" binding:"required,min=1,max=100"`
	Street        string `json:"street" binding:"required,min=1,max=500"`
	AddressType   string `json:"address_type" binding:"required,oneof=home office other"`
	Notes         string `json:"notes" binding:"omitempty,max=500"`
}

// AddressUpdateRequest DTO for updating an address
type AddressUpdateRequest struct {
	RecipientName string `json:"recipient_name" binding:"omitempty,min=2,max=255"`
	Phone         string `json:"phone" binding:"omitempty"`
	Province      string `json:"province" binding:"omitempty,min=1,max=100"`
	District      string `json:"district" binding:"omitempty,min=1,max=100"`
	Ward          string `json:"ward" binding:"omitempty,min=1,max=100"`
	Street        string `json:"street" binding:"omitempty,min=1,max=500"`
	AddressType   string `json:"address_type" binding:"omitempty,oneof=home office other"`
	Notes         string `json:"notes" binding:"omitempty,max=500"`
}

// AddressResponse DTO for API response
type AddressResponse struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	RecipientName string    `json:"recipient_name"`
	Phone         string    `json:"phone"`
	Province      string    `json:"province"`
	District      string    `json:"district"`
	Ward          string    `json:"ward"`
	Street        string    `json:"street"`
	AddressType   string    `json:"address_type"`
	IsDefault     bool      `json:"is_default"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Longitude     *string   `json:"longitude"`
	Latitude      *string   `json:"latitude"`
}

// AddressWithUserResponse DTO - Address with user information
type AddressWithUserResponse struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	UserName      string    `json:"user_name"`  // User's full name
	UserEmail     string    `json:"user_email"` // User's email
	RecipientName string    `json:"recipient_name"`
	Phone         string    `json:"phone"`
	Province      string    `json:"province"`
	District      string    `json:"district"`
	Ward          string    `json:"ward"`
	Street        string    `json:"street"`
	AddressType   string    `json:"address_type"`
	IsDefault     bool      `json:"is_default"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func ValidateAddressCreate(req *AddressCreateRequest) error {
	if req == nil {
		return NewInvalidRecipientName("request is nil")
	}

	// Validate recipient name
	if err := ValidateRecipientName(req.RecipientName); err != nil {
		return err
	}

	// Validate phone
	if err := ValidatePhoneVietnam(req.Phone); err != nil {
		return err
	}

	// Validate province
	if err := ValidateProvince(req.Province); err != nil {
		return err
	}

	// Validate district
	if err := ValidateDistrict(req.District); err != nil {
		return err
	}

	// Validate ward
	if err := ValidateWard(req.Ward); err != nil {
		return err
	}

	// Validate street
	if err := ValidateStreet(req.Street); err != nil {
		return err
	}

	// Validate address type
	if err := ValidateAddressType(req.AddressType); err != nil {
		return err
	}

	return nil
}

// ValidateAddressUpdate validates AddressUpdateRequest
func ValidateAddressUpdate(req *AddressUpdateRequest) error {
	if req == nil {
		return NewInvalidRecipientName("request is nil")
	}

	// Validate recipient name (optional)
	if req.RecipientName != "" {
		if err := ValidateRecipientName(req.RecipientName); err != nil {
			return err
		}
	}

	// Validate phone (optional)
	if req.Phone != "" {
		if err := ValidatePhoneVietnam(req.Phone); err != nil {
			return err
		}
	}

	// Validate province (optional)
	if req.Province != "" {
		if err := ValidateProvince(req.Province); err != nil {
			return err
		}
	}

	// Validate district (optional)
	if req.District != "" {
		if err := ValidateDistrict(req.District); err != nil {
			return err
		}
	}

	// Validate ward (optional)
	if req.Ward != "" {
		if err := ValidateWard(req.Ward); err != nil {
			return err
		}
	}

	// Validate street (optional)
	if req.Street != "" {
		if err := ValidateStreet(req.Street); err != nil {
			return err
		}
	}

	// Validate address type (optional)
	if req.AddressType != "" {
		if err := ValidateAddressType(req.AddressType); err != nil {
			return err
		}
	}

	return nil
}

// ValidateRecipientName validates recipient name
// - Không để trống
// - Tối thiểu 2 ký tự, tối đa 255 ký tự
func ValidateRecipientName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return NewInvalidRecipientName("")
	}

	if len(name) < 2 {
		return NewInvalidRecipientName("name must be at least 2 characters")
	}

	if len(name) > 255 {
		return NewInvalidRecipientName("name must not exceed 255 characters")
	}

	return nil
}

// ValidatePhoneVietnam validates Vietnam phone number format
// Accepted formats: 0xxx-xxx-xxx or +84-xxx-xxx-xxx
func ValidatePhoneVietnam(phone string) error {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return NewInvalidPhone("")
	}

	// Pattern: 0xxx-xxx-xxx (10 digits with dashes) or +84-xxx-xxx-xxx
	vietnamPhoneRegex := regexp.MustCompile(`^(0\d{3}-\d{3}-\d{4}|\+84-\d{3}-\d{3}-\d{4})$`)

	if !vietnamPhoneRegex.MatchString(phone) {
		return NewInvalidPhone(phone)
	}

	return nil
}

// ValidateProvince validates province
// - Không để trống
// - Tối đa 100 ký tự
// - Không chứa ký tự đặc biệt (except spaces)
func ValidateProvince(province string) error {
	province = strings.TrimSpace(province)

	if province == "" {
		return NewInvalidProvince("")
	}

	if len(province) > 100 {
		return NewInvalidProvince("province must not exceed 100 characters")
	}

	// Allow letters, spaces, and common Vietnamese characters
	provinceRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđ]+$`)

	if !provinceRegex.MatchString(province) {
		return NewInvalidProvince(province)
	}

	return nil
}

// ValidateDistrict validates district
// - Không để trống
// - Tối đa 100 ký tự
func ValidateDistrict(district string) error {
	district = strings.TrimSpace(district)

	if district == "" {
		return NewInvalidDistrict("")
	}

	if len(district) > 100 {
		return NewInvalidDistrict("district must not exceed 100 characters")
	}

	districtRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđ]+$`)

	if !districtRegex.MatchString(district) {
		return NewInvalidDistrict(district)
	}

	return nil
}

// ValidateWard validates ward
// - Không để trống
// - Tối đa 100 ký tự
func ValidateWard(ward string) error {
	ward = strings.TrimSpace(ward)

	if ward == "" {
		return NewInvalidWard("")
	}

	if len(ward) > 100 {
		return NewInvalidWard("ward must not exceed 100 characters")
	}

	wardRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđ]+$`)

	if !wardRegex.MatchString(ward) {
		return NewInvalidWard(ward)
	}

	return nil
}

// ValidateStreet validates street address
// - Không để trống
// - Tối đa 500 ký tự
func ValidateStreet(street string) error {
	street = strings.TrimSpace(street)

	if street == "" {
		return NewInvalidStreet("")
	}

	if len(street) > 500 {
		return NewInvalidStreet("street address must not exceed 500 characters")
	}

	return nil
}

// ValidateAddressType validates address type
// Must be: home, office, or other
func ValidateAddressType(addressType string) error {
	addressType = strings.TrimSpace(strings.ToLower(addressType))

	validTypes := map[string]bool{
		"home":   true,
		"office": true,
		"other":  true,
	}

	if !validTypes[addressType] {
		return NewInvalidAddressType(addressType)
	}

	return nil
}
