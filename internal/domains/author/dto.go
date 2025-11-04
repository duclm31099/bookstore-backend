package author

import (
	"time"

	"github.com/google/uuid"
)

// Constants for validation
const (
	MaxNameLength = 255
	MinNameLength = 2
	MaxBioLength  = 5000
)

// IsValid validates the Author entity
func (a *Author) IsValid() error {
	if len(a.Name) < MinNameLength {
		return ErrInvalidName
	}
	if len(a.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if a.Slug == "" {
		return ErrInvalidSlug
	}
	if a.Bio != nil && len(*a.Bio) > MaxBioLength {
		return ErrBioTooLong
	}
	return nil
}

// HasBio checks if author has biography
func (a *Author) HasBio() bool {
	return a.Bio != nil && *a.Bio != ""
}

// HasPhoto checks if author has photo
func (a *Author) HasPhoto() bool {
	return a.PhotoURL != nil && *a.PhotoURL != ""
}

type SearchAuthorResponse struct {
	Authors []AuthorResponse `json:"authors"`
	Total   int32            `json:"total"`
}

// CreateAuthorRequest - POST /v1/authors
type CreateAuthorRequest struct {
	Name     string  `json:"name" validate:"required,min=2,max=255"`
	Bio      *string `json:"bio,omitempty" validate:"omitempty,max=5000"`
	PhotoURL *string `json:"photo_url,omitempty" validate:"omitempty,url"`
}

// UpdateAuthorRequest - PUT /v1/authors/:id
// All fields optional for partial updates (PATCH behavior)
type UpdateAuthorRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Bio      *string `json:"bio,omitempty" validate:"omitempty,max=5000"`
	PhotoURL *string `json:"photo_url,omitempty" validate:"omitempty,url"`
	Version  int     `json:"version" validate:"required"` // Required for conflict detection
}

// UpdateBioRequest - PATCH /v1/authors/:id/bio
type UpdateBioRequest struct {
	Bio     *string `json:"bio" validate:"required,max=5000"`
	Version int     `json:"version" validate:"required"`
}

// UpdatePhotoRequest - PATCH /v1/authors/:id/photo
type UpdatePhotoRequest struct {
	PhotoURL *string `json:"photo_url" validate:"required,url"`
	Version  int     `json:"version" validate:"required"`
}

// AuthorResponse - Basic author information
type AuthorResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Bio       *string   `json:"bio,omitempty"`
	PhotoURL  *string   `json:"photo_url,omitempty"`
	Version   int       `json:"version"` // For client-side conflict detection
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthorDetailResponse - Detailed author with relationships
type AuthorDetailResponse struct {
	AuthorResponse
	BookCount int `json:"book_count"` // Aggregated data
}

// AuthorListResponse - Paginated list response
type AuthorListResponse struct {
	Data       []AuthorResponse `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
}

// PaginationMeta - Reusable pagination metadata
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// AuthorFilter - Query parameters for search/filter
type AuthorFilter struct {
	Search string `json:"search" form:"search"`   // Partial name search
	SortBy string `json:"sort_by" form:"sort_by"` // name, created_at, updated_at
	Order  string `json:"order" form:"order"`     // asc, desc
	Limit  int    `json:"limit" form:"limit" validate:"min=1,max=100"`
	Offset int    `json:"offset" form:"offset" validate:"min=0"`
}

// BulkDeleteRequest - DELETE /v1/authors/bulk
type BulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100"`
}

// BulkDeleteResponse - Response with partial success info
type BulkDeleteResponse struct {
	SuccessCount int         `json:"success_count"`
	FailedCount  int         `json:"failed_count"`
	Errors       []BulkError `json:"errors,omitempty"`
}

type BulkError struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}

// Conversion methods

// ToResponse converts Author entity to AuthorResponse DTO
func (a Author) ToResponse() *AuthorResponse {
	return &AuthorResponse{
		ID:        a.ID,
		Name:      a.Name,
		Slug:      a.Slug,
		Bio:       a.Bio,
		PhotoURL:  a.PhotoURL,
		Version:   a.Version,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

// ToDetailResponse converts Author to detailed response with book count
func (a *Author) ToDetailResponse(bookCount int) *AuthorDetailResponse {
	return &AuthorDetailResponse{
		AuthorResponse: *a.ToResponse(),
		BookCount:      bookCount,
	}
}

// ToEntity converts CreateAuthorRequest to Author entity
func (req *CreateAuthorRequest) ToEntity(slug string) *Author {
	return &Author{
		Name:     req.Name,
		Slug:     slug,
		Bio:      req.Bio,
		PhotoURL: req.PhotoURL,
		Version:  0, // Initial version
	}
}

// ApplyToEntity applies UpdateAuthorRequest to existing Author entity
func (req *UpdateAuthorRequest) ApplyToEntity(author *Author) {
	if req.Name != nil {
		author.Name = *req.Name
	}
	if req.Bio != nil {
		author.Bio = req.Bio
	}
	if req.PhotoURL != nil {
		author.PhotoURL = req.PhotoURL
	}
}
