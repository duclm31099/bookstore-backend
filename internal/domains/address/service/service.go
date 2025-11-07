package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	a "bookstore-backend/internal/domains/address"
)

type addressService struct {
	repo a.Repository
}

func NewAddressService(repo a.Repository) a.ServiceInterface {
	return &addressService{
		repo: repo,
	}
}

// CreateAddress creates a new address for user
func (s *addressService) CreateAddress(ctx context.Context, userID uuid.UUID, req *a.AddressCreateRequest) (*a.AddressResponse, error) {
	// Validate request
	if err := a.ValidateAddressCreate(req); err != nil {
		return nil, err
	}

	// Create address model
	addr := &a.Address{
		UserID:        userID,
		RecipientName: strings.TrimSpace(req.RecipientName),
		Phone:         strings.TrimSpace(req.Phone),
		Province:      strings.TrimSpace(req.Province),
		District:      strings.TrimSpace(req.District),
		Ward:          strings.TrimSpace(req.Ward),
		Street:        strings.TrimSpace(req.Street),
		AddressType:   a.AddressType(strings.ToLower(strings.TrimSpace(req.AddressType))),
		IsDefault:     false,
		Notes:         strings.TrimSpace(req.Notes),
	}

	// Create in repository
	createdAddr, err := s.repo.Create(ctx, addr)
	if err != nil {
		return nil, err
	}

	return s.modelToResponse(createdAddr), nil
}

// GetAddress retrieves an address by ID (with authorization check)
func (s *addressService) GetAddressByID(ctx context.Context, userID, addressID uuid.UUID) (*a.AddressResponse, error) {

	addr, err := s.repo.GetByID(ctx, addressID)
	if err != nil {
		return nil, err
	}

	if addr == nil {
		return nil, a.NewAddressNotFound()
	}

	// Check ownership
	if addr.UserID != userID {
		return nil, a.NewAddressNotBelongToUser(addressID.String(), userID.String())
	}

	return s.modelToResponse(addr), nil
}

// ListUserAddresses retrieves all addresses for a user
func (s *addressService) ListUserAddresses(ctx context.Context, userID uuid.UUID) ([]*a.AddressResponse, error) {

	addrs, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return []*a.AddressResponse{}, nil // Return empty array instead of nil
	}

	responses := make([]*a.AddressResponse, len(addrs))
	for i, addr := range addrs {
		responses[i] = s.modelToResponse(addr)
	}

	return responses, nil
}

// GetDefaultAddress retrieves default address for a user
func (s *addressService) GetDefaultAddress(ctx context.Context, userID uuid.UUID) (*a.AddressResponse, error) {

	addr, err := s.repo.GetDefaultByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if addr == nil {
		return nil, a.NewAddressNotFound()
	}

	return s.modelToResponse(addr), nil
}

// UpdateAddress - Optimized version
func (s *addressService) UpdateAddress(ctx context.Context, userID, addressID uuid.UUID, req *a.AddressUpdateRequest) (*a.AddressResponse, error) {
	// Validate request (includes nil check)
	if err := a.ValidateAddressUpdate(req); err != nil {
		return nil, err
	}
	// Get existing address with ownership check
	existing, err := s.getAddressWithOwnershipCheck(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}
	// Merge request with existing values
	updateAddr := s.mergeAddressUpdate(req, existing)
	// Update in repository
	updatedAddr, err := s.repo.Update(ctx, addressID, updateAddr)
	if err != nil {
		return nil, err
	}

	return s.modelToResponse(updatedAddr), nil
}

// Helper: Get address and verify ownership
func (s *addressService) getAddressWithOwnershipCheck(ctx context.Context, userID, addressID uuid.UUID) (*a.Address, error) {
	existing, err := s.repo.GetByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, a.NewAddressNotFound()
	}
	if existing.UserID != userID {
		return nil, a.NewAddressNotBelongToUser(addressID.String(), userID.String())
	}
	return existing, nil
}

// Helper: Merge update request with existing values
func (s *addressService) mergeAddressUpdate(req *a.AddressUpdateRequest, existing *a.Address) *a.Address {
	return &a.Address{
		RecipientName: s.getOrDefault(req.RecipientName, existing.RecipientName),
		Phone:         s.getOrDefault(req.Phone, existing.Phone),
		Province:      s.getOrDefault(req.Province, existing.Province),
		District:      s.getOrDefault(req.District, existing.District),
		Ward:          s.getOrDefault(req.Ward, existing.Ward),
		Street:        s.getOrDefault(req.Street, existing.Street),
		AddressType:   s.getOrDefaultType(req.AddressType, existing.AddressType),
		Notes:         s.getOrDefault(req.Notes, existing.Notes),
	}
}

// Helper: Return new value if not empty, else return existing
func (s *addressService) getOrDefault(newVal, existingVal string) string {
	newVal = strings.TrimSpace(newVal)
	if newVal == "" {
		return existingVal
	}
	return newVal
}

// Helper: Same for AddressType
func (s *addressService) getOrDefaultType(newVal string, existingVal a.AddressType) a.AddressType {
	newVal = strings.TrimSpace(strings.ToLower(newVal))
	if newVal == "" {
		return existingVal
	}
	return a.AddressType(newVal)
}

// DeleteAddress removes an address (with authorization check)
func (s *addressService) DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error {

	_, err := s.getAddressWithOwnershipCheck(ctx, userID, addressID)
	if err != nil {
		return err
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if count == 1 {
		return errors.New("Can not delete the last address")
	}

	// Delete address
	err = s.repo.Delete(ctx, addressID)
	if err != nil {
		return err
	}

	return nil
}

// SetDefaultAddress sets an address as default for user
func (s *addressService) SetDefaultAddress(ctx context.Context, userID, addressID uuid.UUID) (*a.AddressResponse, error) {
	if userID == uuid.Nil {
		return nil, a.NewInvalidUserID("user_id cannot be nil")
	}

	if addressID == uuid.Nil {
		return nil, a.NewInvalidAddressID("address_id cannot be nil")
	}

	// Verify address exists and belongs to user
	existing, err := s.getAddressWithOwnershipCheck(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}

	// Set as default
	err = s.repo.SetDefault(ctx, addressID, userID)
	if err != nil {
		return nil, err
	}

	existing.IsDefault = true

	return s.modelToResponse(existing), nil
}

// UnsetDefaultAddress unsets default flag (only if user has multiple addresses)
func (s *addressService) UnsetDefaultAddress(ctx context.Context, userID, addressID uuid.UUID) error {
	if userID == uuid.Nil {
		return a.NewInvalidUserID("user_id cannot be nil")
	}

	if addressID == uuid.Nil {
		return a.NewInvalidAddressID("address_id cannot be nil")
	}

	// Verify address exists and belongs to user
	_, err := s.getAddressWithOwnershipCheck(ctx, userID, addressID)

	// Check if this is the only address for user
	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if count == 1 {
		return a.NewCannotUnsetOnlyDefault()
	}

	// Unset default
	err = s.repo.UnsetDefault(ctx, addressID)
	if err != nil {
		return err
	}

	return nil
}

// GetAddressWithUser retrieves address with user information (for admin)
func (s *addressService) GetAddressWithUser(ctx context.Context, addressID uuid.UUID) (*a.AddressWithUserResponse, error) {
	if addressID == uuid.Nil {
		return nil, a.NewInvalidAddressID("address_id cannot be nil")
	}

	resp, err := s.repo.GetAddressWithUser(ctx, addressID)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, a.NewAddressNotFound()
	}

	return resp, nil
}

// ListAllAddresses retrieves all addresses (for admin)
func (s *addressService) ListAllAddresses(ctx context.Context, page, pageSize int) ([]*a.AddressResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	addrs, err := s.repo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*a.AddressResponse, len(addrs))
	for i, addr := range addrs {
		responses[i] = s.modelToResponse(addr)
	}

	return responses, total, nil
}

// Helper: Convert Address model to AddressResponse DTO
func (s *addressService) modelToResponse(addr *a.Address) *a.AddressResponse {
	return &a.AddressResponse{
		ID:            addr.ID,
		UserID:        addr.UserID,
		RecipientName: addr.RecipientName,
		Phone:         addr.Phone,
		Province:      addr.Province,
		District:      addr.District,
		Ward:          addr.Ward,
		Street:        addr.Street,
		AddressType:   string(addr.AddressType),
		IsDefault:     addr.IsDefault,
		Notes:         addr.Notes,
		CreatedAt:     addr.CreatedAt,
		UpdatedAt:     addr.UpdatedAt,
	}
}
