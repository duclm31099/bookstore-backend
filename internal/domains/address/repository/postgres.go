package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	a "bookstore-backend/internal/domains/address"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) a.Repository {
	return &postgresRepository{
		pool: pool,
	}
}

// Create inserts a new address record
func (r *postgresRepository) Create(ctx context.Context, addr *a.Address) (*a.Address, error) {
	query := `
    INSERT INTO addresses 
    (user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
    RETURNING id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at
  `

	row := r.pool.QueryRow(
		ctx, query,
		addr.UserID, addr.RecipientName, addr.Phone, addr.Province, addr.District,
		addr.Ward, addr.Street, addr.AddressType, addr.IsDefault, addr.Notes,
	)

	var address a.Address
	err := row.Scan(
		&address.ID, &address.UserID, &address.RecipientName, &address.Phone,
		&address.Province, &address.District, &address.Ward, &address.Street,
		&address.AddressType, &address.IsDefault, &address.Notes,
		&address.CreatedAt, &address.UpdatedAt,
	)

	if err != nil {
		return nil, a.NewCreateAddressError(err)
	}

	return &address, nil
}

// GetByID retrieves an address by ID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*a.Address, error) {
	query := `
    SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at
    FROM addresses
    WHERE id = $1
  `

	row := r.pool.QueryRow(ctx, query, id)

	var addr a.Address
	err := row.Scan(
		&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
		&addr.Province, &addr.District, &addr.Ward, &addr.Street,
		&addr.AddressType, &addr.IsDefault, &addr.Notes,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, a.NewCreateAddressError(err)
	}

	return &addr, nil
}

// GetByUserID retrieves all addresses for a user
func (r *postgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*a.Address, error) {
	query := `
    SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at
    FROM addresses
    WHERE user_id = $1
    ORDER BY is_default DESC, created_at DESC
  `

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, a.NewCreateAddressError(err)
	}
	defer rows.Close()

	var addresses []*a.Address

	for rows.Next() {
		var addr a.Address
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
			&addr.Province, &addr.District, &addr.Ward, &addr.Street,
			&addr.AddressType, &addr.IsDefault, &addr.Notes,
			&addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			return nil, a.NewCreateAddressError(err)
		}
		addresses = append(addresses, &addr)
	}

	if err = rows.Err(); err != nil {
		return nil, a.NewCreateAddressError(err)
	}

	return addresses, nil
}

// GetDefaultByUserID retrieves default address for a user
func (r *postgresRepository) GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*a.Address, error) {
	query := `
    SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at
    FROM addresses
    WHERE user_id = $1 AND is_default = true
  `

	row := r.pool.QueryRow(ctx, query, userID)

	var addr a.Address
	err := row.Scan(
		&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
		&addr.Province, &addr.District, &addr.Ward, &addr.Street,
		&addr.AddressType, &addr.IsDefault, &addr.Notes,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, a.NewCreateAddressError(err)
	}

	return &addr, nil
}

// List retrieves all addresses (for admin use)
func (r *postgresRepository) List(ctx context.Context, offset, limit int) ([]*a.Address, error) {
	query := `
    SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at
    FROM addresses
    ORDER BY created_at DESC
    LIMIT $1 OFFSET $2
  `

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, a.NewCreateAddressError(err)
	}
	defer rows.Close()

	var addresses []*a.Address

	for rows.Next() {
		var addr a.Address
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
			&addr.Province, &addr.District, &addr.Ward, &addr.Street,
			&addr.AddressType, &addr.IsDefault, &addr.Notes,
			&addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			return nil, a.NewCreateAddressError(err)
		}
		addresses = append(addresses, &addr)
	}

	if err = rows.Err(); err != nil {
		return nil, a.NewCreateAddressError(err)
	}

	return addresses, nil
}

// Count returns total number of addresses
func (r *postgresRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM addresses`

	row := r.pool.QueryRow(ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, a.NewCreateAddressError(err)
	}

	return count, nil
}

// CountByUserID returns total addresses for a user
func (r *postgresRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM addresses WHERE user_id = $1`

	row := r.pool.QueryRow(ctx, query, userID)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, a.NewCreateAddressError(err)
	}

	return count, nil
}

// Update updates address information
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, addr *a.Address) (*a.Address, error) {
	query := `
    UPDATE addresses
    SET recipient_name = $1, phone = $2, province = $3, district = $4, ward = $5, 
        street = $6, address_type = $7, notes = $8, updated_at = NOW()
    WHERE id = $9
    RETURNING id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, created_at, updated_at
  `

	row := r.pool.QueryRow(
		ctx, query,
		addr.RecipientName, addr.Phone, addr.Province, addr.District, addr.Ward,
		addr.Street, addr.AddressType, addr.Notes, id,
	)

	var updatedAddr a.Address
	err := row.Scan(
		&updatedAddr.ID, &updatedAddr.UserID, &updatedAddr.RecipientName, &updatedAddr.Phone,
		&updatedAddr.Province, &updatedAddr.District, &updatedAddr.Ward, &updatedAddr.Street,
		&updatedAddr.AddressType, &updatedAddr.IsDefault, &updatedAddr.Notes,
		&updatedAddr.CreatedAt, &updatedAddr.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, a.NewAddressNotFound()
		}
		return nil, a.NewUpdateAddressError(err)
	}

	return &updatedAddr, nil
}

// Delete removes an address record
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM addresses WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return a.NewDeleteAddressError(err)
	}

	if result.RowsAffected() == 0 {
		return a.NewAddressNotFound()
	}

	return nil
}

// SetDefault sets an address as default for user
func (r *postgresRepository) SetDefault(ctx context.Context, addressID, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return a.NewUpdateAddressError(err)
	}
	defer tx.Rollback(ctx)

	// Unset all other addresses as default
	_, err = tx.Exec(ctx, `UPDATE addresses SET is_default = false WHERE user_id = $1 AND id != $2`, userID, addressID)
	if err != nil {
		return a.NewUpdateAddressError(err)
	}

	// Set this address as default
	result, err := tx.Exec(ctx, `UPDATE addresses SET is_default = true WHERE id = $1 AND user_id = $2`, addressID, userID)
	if err != nil {
		return a.NewUpdateAddressError(err)
	}

	if result.RowsAffected() == 0 {
		return a.NewAddressNotFound()
	}

	if err = tx.Commit(ctx); err != nil {
		return a.NewUpdateAddressError(err)
	}

	return nil
}

// UnsetDefault unsets default flag for an address
func (r *postgresRepository) UnsetDefault(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE addresses SET is_default = false WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return a.NewUpdateAddressError(err)
	}

	if result.RowsAffected() == 0 {
		return a.NewAddressNotFound()
	}

	return nil
}

// GetAddressWithUser retrieves address with user information
func (r *postgresRepository) GetAddressWithUser(ctx context.Context, id uuid.UUID) (*a.AddressWithUserResponse, error) {
	query := `
    SELECT 
      a.id, a.user_id, u.full_name, u.email, 
      a.recipient_name, a.phone, a.province, a.district, a.ward, a.street, 
      a.address_type, a.is_default, a.notes, a.created_at, a.updated_at
    FROM addresses a
    LEFT JOIN users u ON a.user_id = u.id
    WHERE a.id = $1
  `

	row := r.pool.QueryRow(ctx, query, id)

	var resp a.AddressWithUserResponse
	err := row.Scan(
		&resp.ID, &resp.UserID, &resp.UserName, &resp.UserEmail,
		&resp.RecipientName, &resp.Phone, &resp.Province, &resp.District, &resp.Ward, &resp.Street,
		&resp.AddressType, &resp.IsDefault, &resp.Notes, &resp.CreatedAt, &resp.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, a.NewCreateAddressError(err)
	}

	return &resp, nil
}

// ListAddressesWithUser retrieves addresses with user info (for admin)
func (r *postgresRepository) ListAddressesWithUser(ctx context.Context, offset, limit int) ([]*a.AddressWithUserResponse, error) {
	query := `
    SELECT 
      a.id, a.user_id, u.full_name, u.email, 
      a.recipient_name, a.phone, a.province, a.district, a.ward, a.street, 
      a.address_type, a.is_default, a.notes, a.created_at, a.updated_at
    FROM addresses a
    LEFT JOIN users u ON a.user_id = u.id
    ORDER BY a.created_at DESC
    LIMIT $1 OFFSET $2
  `

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, a.NewCreateAddressError(err)
	}
	defer rows.Close()

	var addresses []*a.AddressWithUserResponse

	for rows.Next() {
		var resp a.AddressWithUserResponse
		err := rows.Scan(
			&resp.ID, &resp.UserID, &resp.UserName, &resp.UserEmail,
			&resp.RecipientName, &resp.Phone, &resp.Province, &resp.District, &resp.Ward, &resp.Street,
			&resp.AddressType, &resp.IsDefault, &resp.Notes, &resp.CreatedAt, &resp.UpdatedAt,
		)
		if err != nil {
			return nil, a.NewCreateAddressError(err)
		}
		addresses = append(addresses, &resp)
	}

	if err = rows.Err(); err != nil {
		return nil, a.NewCreateAddressError(err)
	}

	return addresses, nil
}
