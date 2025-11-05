package address

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines all data access operations for Address domain
type Repository interface {
	// Create inserts a new address record
	Create(ctx context.Context, address *Address) (*Address, error)

	// GetByID retrieves an address by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Address, error)

	// GetByUserID retrieves all addresses for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Address, error)

	// GetDefaultByUserID retrieves default address for a user
	GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*Address, error)

	// List retrieves all addresses (for admin use)
	List(ctx context.Context, offset, limit int) ([]*Address, error)

	// Count returns total number of addresses
	Count(ctx context.Context) (int, error)

	// CountByUserID returns total addresses for a user
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// Update updates address information
	Update(ctx context.Context, id uuid.UUID, address *Address) (*Address, error)

	// Delete removes an address record
	Delete(ctx context.Context, id uuid.UUID) error

	// SetDefault sets an address as default for user
	SetDefault(ctx context.Context, addressID, userID uuid.UUID) error

	// UnsetDefault unsets default flag for an address
	UnsetDefault(ctx context.Context, id uuid.UUID) error

	// GetAddressWithUser retrieves address with user information
	GetAddressWithUser(ctx context.Context, id uuid.UUID) (*AddressWithUserResponse, error)

	// ListAddressesWithUser retrieves addresses with user info (for admin)
	ListAddressesWithUser(ctx context.Context, offset, limit int) ([]*AddressWithUserResponse, error)
}
