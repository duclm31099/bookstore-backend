package address

import (
	"context"

	"github.com/google/uuid"
)

// Service defines all business logic operations for Address domain
type ServiceInterface interface {
	// CreateAddress creates a new address for user
	CreateAddress(ctx context.Context, userID uuid.UUID, req *AddressCreateRequest) (*AddressResponse, error)

	// GetAddress retrieves an address by ID
	GetAddressByID(ctx context.Context, userID, addressID uuid.UUID) (*AddressResponse, error)

	// ListUserAddresses retrieves all addresses for a user
	ListUserAddresses(ctx context.Context, userID uuid.UUID) ([]*AddressResponse, error)

	// GetDefaultAddress retrieves default address for a user
	GetDefaultAddress(ctx context.Context, userID uuid.UUID) (*AddressResponse, error)

	// UpdateAddress updates an address
	UpdateAddress(ctx context.Context, userID, addressID uuid.UUID, req *AddressUpdateRequest) (*AddressResponse, error)

	// DeleteAddress removes an address
	DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error

	// SetDefaultAddress sets an address as default for user
	SetDefaultAddress(ctx context.Context, userID, addressID uuid.UUID) (*AddressResponse, error)

	// UnsetDefaultAddress unsets default flag (only if user has multiple addresses)
	UnsetDefaultAddress(ctx context.Context, userID, addressID uuid.UUID) error

	// GetAddressWithUser retrieves address with user information (for admin)
	GetAddressWithUser(ctx context.Context, addressID uuid.UUID) (*AddressWithUserResponse, error)

	// ListAllAddresses retrieves all addresses (for admin)
	ListAllAddresses(ctx context.Context, page, pageSize int) ([]*AddressResponse, int, error)
}
