package publisher

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Publisher represents a book publisher in the system
// Used across all layers (repository, service, handler)
type Publisher struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	Website   string    `db:"website"`
	Email     string    `db:"email"`
	Phone     string    `db:"phone"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// PublisherCreateRequest DTO for creating a new publisher
// Validation tags có thể thêm (validation layer tùy bạn)
type PublisherCreateRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=255"`
	Slug    string `json:"slug" binding:"required,min=1,max=255"`
	Website string `json:"website" binding:"omitempty,url"`
	Email   string `json:"email" binding:"omitempty,email"`
	Phone   string `json:"phone" binding:"omitempty"`
}

// PublisherUpdateRequest DTO for updating a publisher
type PublisherUpdateRequest struct {
	Name    string `json:"name" binding:"omitempty,min=1,max=255"`
	Website string `json:"website" binding:"omitempty,url"`
	Email   string `json:"email" binding:"omitempty,email"`
	Phone   string `json:"phone" binding:"omitempty"`
}

// PublisherResponse DTO for API response
type PublisherResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Website   string    `json:"website"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PublisherWithBooksResponse DTO - Publisher with associated books
type PublisherWithBooksResponse struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Slug      string      `json:"slug"`
	Website   string      `json:"website"`
	Email     string      `json:"email"`
	Phone     string      `json:"phone"`
	Books     []BookBasic `json:"books"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// BookBasic DTO - Basic book info for publisher response
type BookBasic struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Slug  string    `json:"slug"`
}

func ValidatePublisherCreate(req *PublisherCreateRequest) error {
	if req == nil {
		return NewInvalidPublisherName("request is nil")
	}

	// Validate name
	if err := ValidateName(req.Name); err != nil {
		return err
	}

	// Validate slug
	if err := ValidateSlug(req.Slug); err != nil {
		return err
	}

	// Validate website (optional)
	if req.Website != "" {
		if err := ValidateWebsite(req.Website); err != nil {
			return err
		}
	}

	// Validate email (optional)
	if req.Email != "" {
		if err := ValidateEmail(req.Email); err != nil {
			return err
		}
	}

	// Validate phone (optional)
	if req.Phone != "" {
		if err := ValidatePhone(req.Phone); err != nil {
			return err
		}
	}

	return nil
}

// ValidatePublisherUpdate validates PublisherUpdateRequest
func ValidatePublisherUpdate(req *PublisherUpdateRequest) error {
	if req == nil {
		return NewInvalidPublisherName("request is nil")
	}

	// Validate name (optional)
	if req.Name != "" {
		if err := ValidateName(req.Name); err != nil {
			return err
		}
	}

	// Validate website (optional)
	if req.Website != "" {
		if err := ValidateWebsite(req.Website); err != nil {
			return err
		}
	}

	// Validate email (optional)
	if req.Email != "" {
		if err := ValidateEmail(req.Email); err != nil {
			return err
		}
	}

	// Validate phone (optional)
	if req.Phone != "" {
		if err := ValidatePhone(req.Phone); err != nil {
			return err
		}
	}

	return nil
}

// ValidateName validates publisher name
// - Không để trống
// - Tối thiểu 2 ký tự, tối đa 255 ký tự
// - Không chứa ký tự đặc biệt
func ValidateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return NewInvalidPublisherName("name cannot be empty")
	}

	if len(name) < 2 {
		return NewInvalidPublisherName("name must be at least 2 characters")
	}

	if len(name) > 255 {
		return NewInvalidPublisherName("name must not exceed 255 characters")
	}

	return nil
}

// ValidateSlug validates publisher slug
// - Không để trống
// - Chỉ chứa: a-z, 0-9, dấu gạch ngang (-)
// - Không được bắt đầu/kết thúc bằng dấu gạch ngang
// - Tối đa 255 ký tự
func ValidateSlug(slug string) error {
	slug = strings.TrimSpace(slug)

	if slug == "" {
		return NewInvalidSlug("slug cannot be empty")
	}

	if len(slug) > 255 {
		return NewInvalidSlug("slug must not exceed 255 characters")
	}

	// Chỉ allow a-z, 0-9, dấu gạch ngang
	slugRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)
	if !slugRegex.MatchString(slug) {
		return NewInvalidSlug("slug must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return nil // email optional
	}

	// Simple email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return NewInvalidEmail(email)
	}

	return nil
}

// ValidateWebsite validates website URL format
func ValidateWebsite(website string) error {
	website = strings.TrimSpace(website)

	if website == "" {
		return nil // website optional
	}

	// Simple URL validation - must start with http:// or https://
	if !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
		website = "https://" + website
	}

	return nil
}

// ValidatePhone validates phone format
// - 10-20 ký tự
// - Chỉ chứa số, +, -, khoảng trắng, ()
func ValidatePhone(phone string) error {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return nil // phone optional
	}

	if len(phone) < 10 || len(phone) > 20 {
		return &PublisherError{
			Code:    "INVALID_PHONE",
			Message: "Phone must be between 10 and 20 characters",
		}
	}

	// Allow: digits, +, -, spaces, ()
	phoneRegex := regexp.MustCompile(`^[\d\+\-\s\(\)]+$`)
	if !phoneRegex.MatchString(phone) {
		return &PublisherError{
			Code:    "INVALID_PHONE",
			Message: "Phone contains invalid characters",
		}
	}

	return nil
}

// ContainsOnlyAlphanumeric kiểm tra chuỗi chỉ chứa chữ và số
func ContainsOnlyAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
