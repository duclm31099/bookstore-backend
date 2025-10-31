package model

import (
	"time"

	"github.com/google/uuid"
)

type Publisher struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	Phone     *string   `json:"phone,omitempty" db:"phone"`
	Website   *string   `json:"website,omitempty" db:"website"`
	Email     *string   `json:"email,omitempty" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type PublisherRequest struct {
	Name    string  `json:"name" validate:"required"`
	Slug    string  `json:"slug" validate:"required"`
	Phone   *string `json:"phone,omitempty" validate:"omitempty,e164"`
	Website *string `json:"website,omitempty" validate:"omitempty,url"`
	Email   *string `json:"email,omitempty" validate:"omitempty,email"`
}

type PublisherResponse struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Slug    string    `json:"slug"`
	Phone   *string   `json:"phone,omitempty"`
	Website *string   `json:"website,omitempty"`
	Email   *string   `json:"email,omitempty"`
}

func (p *Publisher) ToResponse() *PublisherResponse {
	return &PublisherResponse{
		ID:      p.ID,
		Name:    p.Name,
		Slug:    p.Slug,
		Website: p.Website,
		Email:   p.Email,
		Phone:   p.Phone,
	}
}
