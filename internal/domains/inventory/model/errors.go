package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ===================================
// DOMAIN ERRORS
// ===================================

var (
	// ErrInventoryNotFound is returned when inventory record is not found
	ErrInventoryNotFound = errors.New("inventory not found")

	// ErrInventoryAlreadyExists is returned when trying to create duplicate inventory
	ErrInventoryAlreadyExists = errors.New("inventory already exists for this book and warehouse")

	// ErrInvalidWarehouseLocation is returned when warehouse location is invalid
	ErrInvalidWarehouseLocation = errors.New("invalid warehouse location, must be one of: HN, HCM, DN, CT")

	// ErrInvalidQuantity is returned when quantity is invalid
	ErrInvalidQuantity = errors.New("quantity cannot be negative")

	// ErrReservedExceedsQuantity is returned when reserved quantity exceeds total quantity
	ErrReservedExceedsQuantity = errors.New("reserved quantity cannot exceed total quantity")

	// ErrCannotDeleteNonEmptyInventory is returned when trying to delete inventory with stock
	ErrCannotDeleteNonEmptyInventory = errors.New("cannot delete inventory with quantity > 0 or reserved_quantity > 0")

	// ErrOptimisticLockFailed is returned when version mismatch occurs (concurrent update)
	ErrOptimisticLockFailed = errors.New("optimistic lock failed: inventory was modified by another transaction")

	// ErrBookNotFound is returned when referenced book does not exist
	ErrBookNotFound = errors.New("book not found")

	// ErrInsufficientStock is returned when not enough available stock to reserve
	ErrInsufficientStock = errors.New("insufficient stock available for reservation")

	// ErrInvalidReleaseQuantity is returned when trying to release more than reserved
	ErrInvalidReleaseQuantity = errors.New("cannot release more than reserved quantity")

	// ErrNoReservationFound is returned when trying to release stock with no reservation
	ErrNoReservationFound = errors.New("no reservation found for this reference")

	// ErrItemNotFulfillable is returned when item cannot be fulfilled from any warehouse
	ErrItemNotFulfillable = errors.New("requested item quantity not fulfillable from any warehouse")

	// ErrInvalidMovementType is returned for invalid movement type
	ErrInvalidMovementType = errors.New("invalid movement type")

	// ErrInvalidAdjustmentQuantity is returned when adjustment quantity is invalid
	ErrInvalidAdjustmentQuantity = errors.New("adjustment quantity cannot be zero")

	// ErrNoInventoryData is returned when no inventory data exists
	ErrNoInventoryData = errors.New("no inventory data available for aggregation")
)

// ===================================
// ERROR HELPERS
// ===================================

// NewInventoryNotFoundError creates a detailed not found error
func NewInventoryNotFoundError(id uuid.UUID) error {
	return fmt.Errorf("%w: id=%s", ErrInventoryNotFound, id)
}

// NewInventoryNotFoundByBookError creates error for book+warehouse not found
func NewInventoryNotFoundByBookError(bookID uuid.UUID, warehouse string) error {
	return fmt.Errorf("%w: book_id=%s, warehouse=%s", ErrInventoryNotFound, bookID, warehouse)
}

// NewOptimisticLockError creates error with version details
func NewOptimisticLockError(expectedVersion, actualVersion int) error {
	return fmt.Errorf("%w: expected version %d, got %d", ErrOptimisticLockFailed, expectedVersion, actualVersion)
}

// IsNotFoundError checks if error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrInventoryNotFound)
}

// IsOptimisticLockError checks if error is an optimistic lock error
func IsOptimisticLockError(err error) bool {
	return errors.Is(err, ErrOptimisticLockFailed)
}

// IsValidationError checks if error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidWarehouseLocation) ||
		errors.Is(err, ErrInvalidQuantity) ||
		errors.Is(err, ErrReservedExceedsQuantity) ||
		errors.Is(err, ErrCannotDeleteNonEmptyInventory)
}

// NewInsufficientStockError creates error with stock details
func NewInsufficientStockError(requested, available int) error {
	return fmt.Errorf("%w: requested=%d, available=%d", ErrInsufficientStock, requested, available)
}

// IsInsufficientStockError checks if error is insufficient stock error
func IsInsufficientStockError(err error) bool {
	return errors.Is(err, ErrInsufficientStock)
}

// IsItemNotFulfillableError checks if error is about unfulfillable item
func IsItemNotFulfillableError(err error) bool {
	return errors.Is(err, ErrItemNotFulfillable)
}

// IsMovementError checks if error is movement-related
func IsMovementError(err error) bool {
	return errors.Is(err, ErrInvalidMovementType) ||
		errors.Is(err, ErrInvalidAdjustmentQuantity)
}
