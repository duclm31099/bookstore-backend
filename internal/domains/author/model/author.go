package model

import (
	"time"

	"github.com/google/uuid"
)

type Author struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	Bio       *string   `json:"bio" db:"bio"`
	PhotoUrl  *string   `json:"photo_url" db:"photo_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type AuthorRequest struct {
	Name     string  `json:"name" validate:"required"`
	Slug     string  `json:"slug" validate:"required"`
	Bio      *string `json:"bio,omitempty"`
	PhotoUrl *string `json:"photo_url,omitempty" validate:"omitempty,url"`
}

type AuthorResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Bio      *string   `json:"bio,omitempty"`
	PhotoUrl *string   `json:"photo_url,omitempty"`
}

// ToResponse converts Author to AuthorResponse
func (a *Author) ToResponse() *AuthorResponse {
	return &AuthorResponse{
		ID:       a.ID,
		Name:     a.Name,
		Slug:     a.Slug,
		Bio:      a.Bio,
		PhotoUrl: a.PhotoUrl,
	}
}
