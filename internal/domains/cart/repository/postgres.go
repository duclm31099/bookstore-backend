package repository

import (
	"bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/pkg/cache"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type postgresRepository struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) RepositoryInterface {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}

// GetByUserID implements RepositoryInterface.GetByUserID
func (r *postgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Cart, error) {
	query := `
		SELECT 
			id, user_id, session_id, items_count, subtotal, version,
			created_at, updated_at, expires_at
		FROM carts
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var cart model.Cart
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.SessionID,
		&cart.ItemsCount,
		&cart.Subtotal,
		&cart.Version,
		&cart.CreatedAt,
		&cart.UpdatedAt,
		&cart.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found - return nil, not error
		}
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	return &cart, nil
}

// GetBySessionID implements RepositoryInterface.GetBySessionID
func (r *postgresRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.Cart, error) {
	query := `
		SELECT 
			id, user_id, session_id, items_count, subtotal, version,
			created_at, updated_at, expires_at
		FROM carts
		WHERE session_id = $1 AND expires_at > NOW()
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var cart model.Cart
	err := r.pool.QueryRow(ctx, query, sessionID).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.SessionID,
		&cart.ItemsCount,
		&cart.Subtotal,
		&cart.Version,
		&cart.CreatedAt,
		&cart.UpdatedAt,
		&cart.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session cart: %w", err)
	}

	return &cart, nil
}

// Create implements RepositoryInterface.Create
func (r *postgresRepository) Create(ctx context.Context, cart *model.Cart) error {
	query := `
		INSERT INTO carts (id, user_id, session_id, items_count, subtotal, version, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		cart.ID,
		cart.UserID,
		cart.SessionID,
		cart.ItemsCount,
		cart.Subtotal,
		cart.Version,
		cart.CreatedAt,
		cart.UpdatedAt,
		cart.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create cart: %w", err)
	}

	return nil
}

// UpdateExpiration implements RepositoryInterface.UpdateExpiration
func (r *postgresRepository) UpdateExpiration(ctx context.Context, cartID uuid.UUID) error {
	query := `
		UPDATE carts
		SET expires_at = NOW() + INTERVAL '30 days', updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, cartID)
	if err != nil {
		return fmt.Errorf("failed to update expiration: %w", err)
	}

	return nil
}

// AddItem implements RepositoryInterface.AddItem
// INSERT or UPDATE if item already exists
func (r *postgresRepository) AddItem(ctx context.Context, item *model.CartItem) error {
	query := `
		INSERT INTO cart_items ( 
		cart_id, book_id, quantity, price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cart_id, book_id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			price = EXCLUDED.price,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(ctx, query,
		item.CartID,
		item.BookID,
		item.Quantity,
		item.Price,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add item: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetItemsWithBooks(ctx context.Context, cartID uuid.UUID, page int, limit int) ([]model.CartItemWithBook, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM cart_items WHERE cart_id = $1`
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, cartID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count items: %w", err)
	}

	// Get paginated items with book details
	// Uses new warehouse_inventory schema with aggregated stock across all warehouses
	query := `
		SELECT 
			ci.id, 
			ci.cart_id, 
			ci.book_id, 
			ci.quantity, 
			ci.price, 
			ci.created_at, 
			ci.updated_at,
			b.title as book_title,
			b.slug as book_slug,
			b.cover_url as book_cover_url,
			a.name as book_author,
			b.price as current_price,
			b.is_active,
			COALESCE(inv.total_available, 0) as total_stock
		FROM cart_items ci
		LEFT JOIN books b ON ci.book_id = b.id
		LEFT JOIN authors a ON b.author_id = a.id
		LEFT JOIN (
			SELECT 
				book_id, 
				SUM(quantity - reserved) as total_available
			FROM warehouse_inventory
			GROUP BY book_id
		) inv ON b.id = inv.book_id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at DESC
		LIMIT $2 OFFSET $3
	`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, cartID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []model.CartItemWithBook
	for rows.Next() {
		var item model.CartItemWithBook
		err := rows.Scan(
			&item.ID,
			&item.CartID,
			&item.BookID,
			&item.Quantity,
			&item.Price,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.BookTitle,
			&item.BookSlug,
			&item.BookCoverURL,
			&item.BookAuthor,
			&item.CurrentPrice,
			&item.IsActive,
			&item.TotalStock,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating items: %w", err)
	}

	return items, totalCount, nil
}

// GetItemByID implements RepositoryInterface.GetItemByID
func (r *postgresRepository) GetItemByID(ctx context.Context, itemID uuid.UUID) (*model.CartItem, error) {
	query := `
		SELECT id, cart_id, book_id, quantity, price, created_at, updated_at
		FROM cart_items
		WHERE id = $1
	`

	var item model.CartItem
	err := r.pool.QueryRow(ctx, query, itemID).Scan(
		&item.ID,
		&item.CartID,
		&item.BookID,
		&item.Quantity,
		&item.Price,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", model.ErrCartItemNotFound, itemID)
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	return &item, nil
}

// GetItemByBookInCart implements RepositoryInterface.GetItemByBookInCart
func (r *postgresRepository) GetItemByBookInCart(ctx context.Context, cartID uuid.UUID, bookID uuid.UUID) (*model.CartItem, error) {
	query := `
		SELECT id, cart_id, book_id, quantity, price, created_at, updated_at
		FROM cart_items
		WHERE cart_id = $1 AND book_id = $2
	`

	var item model.CartItem
	err := r.pool.QueryRow(ctx, query, cartID, bookID).Scan(
		&item.ID,
		&item.CartID,
		&item.BookID,
		&item.Quantity,
		&item.Price,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not an error - just doesn't exist
		}
		return nil, fmt.Errorf("failed to check item: %w", err)
	}

	return &item, nil
}

// DeleteExpiredCarts implements RepositoryInterface.DeleteExpiredCarts
func (r *postgresRepository) DeleteExpiredCarts(ctx context.Context) (int, error) {
	query := `DELETE FROM carts WHERE expires_at < NOW()`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired carts: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// domains/cart/repository_impl.go

// UpdateItem implements RepositoryInterface.UpdateItem
func (r *postgresRepository) UpdateItem(ctx context.Context, item *model.CartItem) error {
	query := `
		UPDATE cart_items
		SET quantity = $2, price = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		item.ID,
		item.Quantity,
		item.Price,
		item.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	return nil
}

// DeleteItem implements RepositoryInterface.DeleteItem
func (r *postgresRepository) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	query := `DELETE FROM cart_items WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

// DeleteCart implements RepositoryInterface.DeleteCart
func (r *postgresRepository) DeleteCart(ctx context.Context, cartID uuid.UUID) error {
	query := `DELETE FROM carts WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, cartID)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	return nil
}

// TransferItem implements RepositoryInterface.TransferItem
func (r *postgresRepository) TransferItem(ctx context.Context, item *model.CartItem, targetCartID uuid.UUID) error {
	query := `
		UPDATE cart_items
		SET cart_id = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		item.ID,
		targetCartID,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to transfer item: %w", err)
	}

	return nil
}
func (r *postgresRepository) ClearCartItems(ctx context.Context, cartID uuid.UUID) (int, error) {
	query := `DELETE FROM cart_items WHERE cart_id = $1`

	result, err := r.pool.Exec(ctx, query, cartID)
	if err != nil {
		return 0, fmt.Errorf("failed to clear cart: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// domains/cart/repository_impl.go

// UpdateCartPromo implements RepositoryInterface.UpdateCartPromo
func (r *postgresRepository) UpdateCartPromo(ctx context.Context, cartID uuid.UUID, promoCode *string, discountAmount decimal.Decimal) error {
	query := `
		UPDATE carts
		SET 
			promo_code = $2,
			discount = $3,
			total = subtotal - $3,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, cartID, promoCode, discountAmount)
	if err != nil {
		return fmt.Errorf("failed to update cart promo: %w", err)
	}

	return nil
}

// RemoveCartPromo implements RepositoryInterface.RemoveCartPromo
func (r *postgresRepository) RemoveCartPromo(ctx context.Context, cartID uuid.UUID) error {
	query := `
		UPDATE carts
		SET 
			promo_code = NULL,
			discount = NULL,
			total = NULL,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, cartID)
	if err != nil {
		return fmt.Errorf("failed to remove cart promo: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetByID(c context.Context, cartID uuid.UUID) (*model.Cart, error) {
	query := `
		SELECT 
			id, user_id, session_id, items_count, subtotal, version,
			created_at, updated_at, expires_at
		FROM carts
		WHERE id = $1 AND expires_at > NOW()
	`

	var cart model.Cart
	err := r.pool.QueryRow(c, query, cartID).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.SessionID,
		&cart.ItemsCount,
		&cart.Subtotal,
		&cart.Version,
		&cart.CreatedAt,
		&cart.UpdatedAt,
		&cart.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session cart: %w", err)
	}

	return &cart, nil
}
