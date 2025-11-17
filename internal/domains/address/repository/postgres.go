package repository

import (
	"bookstore-backend/internal/domains/address/model"
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) RepositoryInterface {
	return &postgresRepository{
		pool: pool,
	}
}

// Create inserts a new address record (bao gồm latitude/longitude)
func (r *postgresRepository) Create(ctx context.Context, addr *model.Address) (*model.Address, error) {
	query := `
        INSERT INTO addresses 
        (user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
        RETURNING id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at
    `

	row := r.pool.QueryRow(
		ctx, query,
		addr.UserID, addr.RecipientName, addr.Phone, addr.Province, addr.District,
		addr.Ward, addr.Street, addr.AddressType, addr.IsDefault, addr.Notes,
		addr.Latitude, addr.Longitude,
	)

	var address model.Address
	err := row.Scan(
		&address.ID, &address.UserID, &address.RecipientName, &address.Phone,
		&address.Province, &address.District, &address.Ward, &address.Street,
		&address.AddressType, &address.IsDefault, &address.Notes,
		&address.Latitude, &address.Longitude,
		&address.CreatedAt, &address.UpdatedAt,
	)

	if err != nil {
		return nil, model.NewCreateAddressError(err)
	}

	return &address, nil
}

// GetByID retrieves an address by ID (bao gồm latitude/longitude)
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Address, error) {
	query := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at
        FROM addresses
        WHERE id = $1
    `

	row := r.pool.QueryRow(ctx, query, id)

	var addr model.Address
	err := row.Scan(
		&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
		&addr.Province, &addr.District, &addr.Ward, &addr.Street,
		&addr.AddressType, &addr.IsDefault, &addr.Notes,
		&addr.Latitude, &addr.Longitude,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, model.NewCreateAddressError(err)
	}

	return &addr, nil
}

// GetByUserID retrieves all addresses for a user (bao gồm latitude/longitude)
func (r *postgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Address, error) {
	query := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at
        FROM addresses
        WHERE user_id = $1
        ORDER BY is_default DESC, created_at DESC
    `

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, model.NewCreateAddressError(err)
	}
	defer rows.Close()

	var addresses []*model.Address

	for rows.Next() {
		var addr model.Address
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
			&addr.Province, &addr.District, &addr.Ward, &addr.Street,
			&addr.AddressType, &addr.IsDefault, &addr.Notes,
			&addr.Latitude, &addr.Longitude,
			&addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			return nil, model.NewCreateAddressError(err)
		}
		addresses = append(addresses, &addr)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewCreateAddressError(err)
	}

	return addresses, nil
}

// GetDefaultByUserID retrieves default address for a user (bao gồm latitude/longitude)
func (r *postgresRepository) GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*model.Address, error) {
	query := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at
        FROM addresses
        WHERE user_id = $1 AND is_default = true
    `

	row := r.pool.QueryRow(ctx, query, userID)

	var addr model.Address
	err := row.Scan(
		&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
		&addr.Province, &addr.District, &addr.Ward, &addr.Street,
		&addr.AddressType, &addr.IsDefault, &addr.Notes,
		&addr.Latitude, &addr.Longitude,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, model.NewCreateAddressError(err)
	}

	return &addr, nil
}

// List retrieves all addresses (for admin use) (bao gồm latitude/longitude)
func (r *postgresRepository) List(ctx context.Context, offset, limit int) ([]*model.Address, error) {
	query := `
        SELECT id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at
        FROM addresses
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, model.NewCreateAddressError(err)
	}
	defer rows.Close()

	var addresses []*model.Address

	for rows.Next() {
		var addr model.Address
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.RecipientName, &addr.Phone,
			&addr.Province, &addr.District, &addr.Ward, &addr.Street,
			&addr.AddressType, &addr.IsDefault, &addr.Notes,
			&addr.Latitude, &addr.Longitude,
			&addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			return nil, model.NewCreateAddressError(err)
		}
		addresses = append(addresses, &addr)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewCreateAddressError(err)
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
		return 0, model.NewCreateAddressError(err)
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
		return 0, model.NewCreateAddressError(err)
	}

	return count, nil
}

// Update updates address information (bao gồm latitude/longitude)
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, addr *model.Address) (*model.Address, error) {
	query := `
        UPDATE addresses
        SET recipient_name = $1, phone = $2, province = $3, district = $4, ward = $5, 
            street = $6, address_type = $7, notes = $8, latitude = $9, longitude = $10, updated_at = NOW()
        WHERE id = $11
        RETURNING id, user_id, recipient_name, phone, province, district, ward, street, address_type, is_default, notes, latitude, longitude, created_at, updated_at
    `

	row := r.pool.QueryRow(
		ctx, query,
		addr.RecipientName, addr.Phone, addr.Province, addr.District, addr.Ward,
		addr.Street, addr.AddressType, addr.Notes, addr.Latitude, addr.Longitude, id,
	)

	var updatedAddr model.Address
	err := row.Scan(
		&updatedAddr.ID, &updatedAddr.UserID, &updatedAddr.RecipientName, &updatedAddr.Phone,
		&updatedAddr.Province, &updatedAddr.District, &updatedAddr.Ward, &updatedAddr.Street,
		&updatedAddr.AddressType, &updatedAddr.IsDefault, &updatedAddr.Notes,
		&updatedAddr.Latitude, &updatedAddr.Longitude,
		&updatedAddr.CreatedAt, &updatedAddr.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewAddressNotFound()
		}
		return nil, model.NewUpdateAddressError(err)
	}

	return &updatedAddr, nil
}

// Delete removes an address record
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM addresses WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return model.NewDeleteAddressError(err)
	}

	if result.RowsAffected() == 0 {
		return model.NewAddressNotFound()
	}

	return nil
}

// SetDefault sets an address as default for user
func (r *postgresRepository) SetDefault(ctx context.Context, addressID, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.NewUpdateAddressError(err)
	}
	defer tx.Rollback(ctx)

	// Unset all other addresses as default
	_, err = tx.Exec(ctx, `UPDATE addresses SET is_default = false WHERE user_id = $1 AND id != $2`, userID, addressID)
	if err != nil {
		return model.NewUpdateAddressError(err)
	}

	// Set this address as default
	result, err := tx.Exec(ctx, `UPDATE addresses SET is_default = true WHERE id = $1 AND user_id = $2`, addressID, userID)
	if err != nil {
		return model.NewUpdateAddressError(err)
	}

	if result.RowsAffected() == 0 {
		return model.NewAddressNotFound()
	}

	if err = tx.Commit(ctx); err != nil {
		return model.NewUpdateAddressError(err)
	}

	return nil
}

// UnsetDefault unsets default flag for an address
func (r *postgresRepository) UnsetDefault(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE addresses SET is_default = false WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return model.NewUpdateAddressError(err)
	}

	if result.RowsAffected() == 0 {
		return model.NewAddressNotFound()
	}

	return nil
}

// GetAddressWithUser retrieves address with user information (bao gồm latitude/longitude)
func (r *postgresRepository) GetAddressWithUser(ctx context.Context, id uuid.UUID) (*model.AddressWithUserResponse, error) {
	query := `
        SELECT 
            a.id, a.user_id, u.full_name, u.email, 
            a.recipient_name, a.phone, a.province, a.district, a.ward, a.street, 
            a.address_type, a.is_default, a.notes, a.latitude, a.longitude, a.created_at, a.updated_at
        FROM addresses a
        LEFT JOIN users u ON a.user_id = u.id
        WHERE a.id = $1
    `

	row := r.pool.QueryRow(ctx, query, id)

	var resp model.AddressWithUserResponse
	var lat, lon *float64
	err := row.Scan(
		&resp.ID, &resp.UserID, &resp.UserName, &resp.UserEmail,
		&resp.RecipientName, &resp.Phone, &resp.Province, &resp.District, &resp.Ward, &resp.Street,
		&resp.AddressType, &resp.IsDefault, &resp.Notes, &lat, &lon, &resp.CreatedAt, &resp.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, model.NewCreateAddressError(err)
	}

	// Convert float64 to *string for response
	if lat != nil {
		latStr := strconv.FormatFloat(*lat, 'f', 6, 64)
		resp.Latitude = &latStr
	}
	if lon != nil {
		lonStr := strconv.FormatFloat(*lon, 'f', 6, 64)
		resp.Longitude = &lonStr
	}

	return &resp, nil
}

// ListAddressesWithUser retrieves addresses with user info (for admin) (bao gồm latitude/longitude)
func (r *postgresRepository) ListAddressesWithUser(ctx context.Context, offset, limit int) ([]*model.AddressWithUserResponse, error) {
	query := `
        SELECT 
            a.id, a.user_id, u.full_name, u.email, 
            a.recipient_name, a.phone, a.province, a.district, a.ward, a.street, 
            a.address_type, a.is_default, a.notes, a.latitude, a.longitude, a.created_at, a.updated_at
        FROM addresses a
        LEFT JOIN users u ON a.user_id = u.id
        ORDER BY a.created_at DESC
        LIMIT $1 OFFSET $2
    `

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, model.NewCreateAddressError(err)
	}
	defer rows.Close()

	var addresses []*model.AddressWithUserResponse

	for rows.Next() {
		var resp model.AddressWithUserResponse
		var lat, lon *float64
		err := rows.Scan(
			&resp.ID, &resp.UserID, &resp.UserName, &resp.UserEmail,
			&resp.RecipientName, &resp.Phone, &resp.Province, &resp.District, &resp.Ward, &resp.Street,
			&resp.AddressType, &resp.IsDefault, &resp.Notes, &lat, &lon, &resp.CreatedAt, &resp.UpdatedAt,
		)
		if err != nil {
			return nil, model.NewCreateAddressError(err)
		}

		// Convert float64 to *string for response
		if lat != nil {
			latStr := strconv.FormatFloat(*lat, 'f', 6, 64)
			resp.Latitude = &latStr
		}
		if lon != nil {
			lonStr := strconv.FormatFloat(*lon, 'f', 6, 64)
			resp.Longitude = &lonStr
		}

		addresses = append(addresses, &resp)
	}

	if err = rows.Err(); err != nil {
		return nil, model.NewCreateAddressError(err)
	}

	return addresses, nil
}
