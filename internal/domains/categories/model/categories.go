package model

import (
	"time"

	"github.com/google/uuid"
)

// Category represents the categories table
type Category struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Slug        string     `json:"slug" db:"slug"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"` // ✅ Fixed: UUID pointer (nullable)
	SortOrder   int        `json:"sort_order" db:"sort_order"`
	Description *string    `json:"description,omitempty" db:"description"` // ✅ Fixed: Nullable
	IconURL     *string    `json:"icon_url,omitempty" db:"icon_url"`       // ✅ Fixed: Nullable
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CategoryTree represents the category_tree materialized view
// Used for hierarchical queries with pre-computed paths
type CategoryTree struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Slug      string     `json:"slug" db:"slug"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	SortOrder int        `json:"sort_order" db:"sort_order"`
	Level     int        `json:"level" db:"level"`         // Tree depth (1 = root)
	Path      []int      `json:"path" db:"path"`           // Sort order path
	FullPath  string     `json:"full_path" db:"full_path"` // "Parent > Child > Grandchild"
}

// ================================================
// REQUEST DTOs
// ================================================

// CategoryRequest for creating/updating categories
type CategoryRequest struct {
	Name        string     `json:"name" validate:"required,min=2,max=100"`
	Slug        string     `json:"slug" validate:"required,slug,min=2,max=100"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" validate:"omitempty,uuid"` // ✅ UUID validation
	SortOrder   int        `json:"sort_order" validate:"min=0"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=500"`
	IconURL     *string    `json:"icon_url,omitempty" validate:"omitempty,url"`
	IsActive    bool       `json:"is_active"`
}

// UpdateCategoryRequest for partial updates (all fields optional except ID)
type UpdateCategoryRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Slug        *string    `json:"slug,omitempty" validate:"omitempty,slug,min=2,max=100"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" validate:"omitempty,uuid"`
	SortOrder   *int       `json:"sort_order,omitempty" validate:"omitempty,min=0"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=500"`
	IconURL     *string    `json:"icon_url,omitempty" validate:"omitempty,url"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// ================================================
// RESPONSE DTOs
// ================================================

// CategoryResponse for API responses
type CategoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	SortOrder   int        `json:"sort_order"`
	Description *string    `json:"description,omitempty"`
	IconURL     *string    `json:"icon_url,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CategoryTreeResponse for hierarchical responses
type CategoryTreeResponse struct {
	ID          uuid.UUID               `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	ParentID    *uuid.UUID              `json:"parent_id,omitempty"`
	SortOrder   int                     `json:"sort_order"`
	Description *string                 `json:"description,omitempty"`
	IconURL     *string                 `json:"icon_url,omitempty"`
	Level       int                     `json:"level"`
	FullPath    string                  `json:"full_path"`
	Children    []*CategoryTreeResponse `json:"children,omitempty"` // Nested children
}

// CategoryBreadcrumb for navigation
type CategoryBreadcrumb struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// ================================================
// CONVERSION METHODS
// ================================================

// ToResponse converts Category to CategoryResponse
func (c *Category) ToResponse() *CategoryResponse {
	return &CategoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		ParentID:    c.ParentID,
		SortOrder:   c.SortOrder,
		Description: c.Description,
		IconURL:     c.IconURL,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// FromRequest converts CategoryRequest to Category
func (r *CategoryRequest) ToModel() *Category {
	return &Category{
		Name:        r.Name,
		Slug:        r.Slug,
		ParentID:    r.ParentID,
		SortOrder:   r.SortOrder,
		Description: r.Description,
		IconURL:     r.IconURL,
		IsActive:    r.IsActive,
	}
}

// ToTreeResponse converts CategoryTree to CategoryTreeResponse
func (ct *CategoryTree) ToTreeResponse() *CategoryTreeResponse {
	return &CategoryTreeResponse{
		ID:        ct.ID,
		Name:      ct.Name,
		Slug:      ct.Slug,
		ParentID:  ct.ParentID,
		SortOrder: ct.SortOrder,
		Level:     ct.Level,
		FullPath:  ct.FullPath,
	}
}

// ================================================
// HELPER METHODS
// ================================================

// IsRoot checks if category is a root category (no parent)
func (c *Category) IsRoot() bool {
	return c.ParentID == nil
}

// HasChildren checks if category has children (requires separate query)
func (c *Category) HasChildren(childCount int) bool {
	return childCount > 0
}

// GetBreadcrumbs returns breadcrumb trail (must be populated from DB)
func (c *Category) GetBreadcrumbs() []CategoryBreadcrumb {
	// This would be populated from a recursive query
	return []CategoryBreadcrumb{}
}
