package repository

import (
	"bookstore-backend/internal/domains/user/model"
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(tx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(tx context.Context, email string) (*model.User, error)
	Update(tx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (email, password_hash, full_name, phone, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		user.Email,
		user.Password,
		user.FullName,
		user.Phone,
		user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, phone, role, is_active, points,
		       is_verified, last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	user := &model.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.FullName, &user.Phone,
		&user.Role, &user.IsActive, &user.Points, &user.IsVerified,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}
func (r *userRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return nil, nil
}

func (r *userRepo) Update(ctx context.Context, user *model.User) error {
	return nil
}
