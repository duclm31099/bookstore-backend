package model

import (
	"time"

	"github.com/google/uuid"
)

type AddressTypeEnum string

const (
	AddressTypeHome   AddressTypeEnum = "home"
	AddressTypeOffice AddressTypeEnum = "office"
	AddressTypeOther  AddressTypeEnum = "other"
)

func (a AddressTypeEnum) IsValid() bool {
	switch a {
	case AddressTypeHome, AddressTypeOffice, AddressTypeOther:
		return true
	}
	return false
}
func (a AddressTypeEnum) String() string {
	return string(a)
}

type Address struct {
	ID     uuid.UUID `json:"id" db:"id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	RecipientName string `json:"recipient_name" db:"recipient_name"`
	Phone         string `json:"phone" db:"phone"`

	Province string `json:"province" db:"province"`
	District string `json:"district" db:"district"`
	Ward     string `json:"ward" db:"ward"`
	Street   string `json:"street" db:"street"`

	AddressType *string   `json:"address_type" db:"address_type"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
	Notes       *string   `json:"notes" db:"notes"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type AddressRequest struct {
	UserID uuid.UUID `json:"user_id"`

	RecipientName string `json:"recipient_name" validate:"required,min=2,max=100"`
	Phone         string `json:"phone" validate:"required,e164"`

	Province string `json:"province" validate:"required"`
	District string `json:"district" validate:"required"`
	Ward     string `json:"ward" validate:"required"`
	Street   string `json:"street" validate:"required"`

	AddressType *string `json:"address_type" validate:"omitempty,oneof=home office other"`
	IsDefault   bool    `json:"is_default"`
	Notes       *string `json:"notes" validate:"omitempty,max=500"`
}

type AddressResponse struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`

	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`

	Province string `json:"province"`
	District string `json:"district"`
	Ward     string `json:"ward"`
	Street   string `json:"street"`

	AddressType *string `json:"address_type,omitempty"`
	IsDefault   bool    `json:"is_default"`
	Notes       *string `json:"notes,omitempty"`
}

func (a *Address) ToResponse() *AddressResponse {
	return &AddressResponse{
		ID:            a.ID,
		UserID:        a.UserID,
		RecipientName: a.RecipientName,
		Phone:         a.Phone,
		Province:      a.Province,
		Ward:          a.Ward,
		District:      a.District,
		Street:        a.Street,
		AddressType:   a.AddressType,
		IsDefault:     a.IsDefault,
		Notes:         a.Notes,
	}
}
