package author

import (
	"time"

	"github.com/google/uuid"
)

// Author represents the core Author entity
// This is the domain model, independent of database/API concerns
type Author struct {
	// Identity - UUID for distributed systems
	ID uuid.UUID `json:"id" db:"id"`

	// Basic Information
	Name string `json:"name" db:"name"`  // Required, max 255 chars
	Slug string `json:"slug"  db:"slug"` // URL-friendly, unique, auto-generated

	// Optional Details
	Bio      *string `json:"bio" db:"bio"`             // Biography, supports Markdown
	PhotoURL *string `json:"photo_url" db:"photo_url"` // Photo storage URL

	// Versioning for Optimistic Locking
	Version int `json:"version" db:"version"` // Incremented on each update

	// Audit timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
