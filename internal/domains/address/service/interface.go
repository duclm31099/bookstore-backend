package service

import (
	"bookstore-backend/internal/domains/address/model"
	"context"

	"github.com/google/uuid"
)

// Service defines all business logic operations for Address domain
type ServiceInterface interface {
	// CreateAddress creates a new address for user
	CreateAddress(ctx context.Context, userID uuid.UUID, req *model.AddressCreateRequest) (*model.AddressResponse, error)

	// GetAddress retrieves an address by ID
	GetAddressByID(ctx context.Context, userID, addressID uuid.UUID) (*model.AddressResponse, error)

	// ListUserAddresses retrieves all addresses for a user
	ListUserAddresses(ctx context.Context, userID uuid.UUID) ([]*model.AddressResponse, error)

	// GetDefaultAddress retrieves default address for a user
	GetDefaultAddress(ctx context.Context, userID uuid.UUID) (*model.AddressResponse, error)

	// UpdateAddress updates an address
	UpdateAddress(ctx context.Context, userID, addressID uuid.UUID, req *model.AddressUpdateRequest) (*model.AddressResponse, error)

	// DeleteAddress removes an address
	DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error

	// SetDefaultAddress sets an address as default for user
	SetDefaultAddress(ctx context.Context, userID, addressID uuid.UUID) (*model.AddressResponse, error)

	// UnsetDefaultAddress unsets default flag (only if user has multiple addresses)
	UnsetDefaultAddress(ctx context.Context, userID, addressID uuid.UUID) error

	// GetAddressWithUser retrieves address with user information (for admin)
	GetAddressWithUser(ctx context.Context, addressID uuid.UUID) (*model.AddressWithUserResponse, error)

	// ListAllAddresses retrieves all addresses (for admin)
	ListAllAddresses(ctx context.Context, page, pageSize int) ([]*model.AddressResponse, int, error)
}
