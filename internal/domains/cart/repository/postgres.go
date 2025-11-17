package repository

import (
	"bookstore-backend/internal/domains/cart/model"
	promo "bookstore-backend/internal/domains/promotion/model"
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
            created_at, updated_at, expires_at,
            promo_code, discount, total, promo_metadata -- ✅ Add promo fields
        FROM carts
        WHERE user_id = $1
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
		&cart.PromoCode,     // ✅ Add
		&cart.Discount,      // ✅ Add
		&cart.Total,         // ✅ Add
		&cart.PromoMetadata, // ✅ Add
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	return &cart, nil
}

func (r *postgresRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.Cart, error) {
	query := `
        SELECT 
            id, user_id, session_id, items_count, subtotal, version,
            created_at, updated_at, expires_at,
            promo_code, discount, total, promo_metadata -- ✅ Add promo fields
        FROM carts
        WHERE user_id = $1
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
		&cart.PromoCode,     // ✅ Add
		&cart.Discount,      // ✅ Add
		&cart.Total,         // ✅ Add
		&cart.PromoMetadata, // ✅ Add
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

// CreateOrGet với DO UPDATE (recommended)
func (r *postgresRepository) CreateOrGet(ctx context.Context, cart *model.Cart) (*model.Cart, error) {
	var conflictColumn string
	if cart.UserID != nil {
		conflictColumn = "user_id"
	} else if cart.SessionID != nil {
		conflictColumn = "session_id"
	} else {
		return nil, fmt.Errorf("either user_id or session_id must be provided")
	}

	query := `
        INSERT INTO carts (
            user_id, session_id, items_count, subtotal, version, 
            created_at, updated_at, expires_at,
            promo_code, discount, total, promo_metadata
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (` + conflictColumn + `) 
        WHERE ` + conflictColumn + ` IS NOT NULL
        DO UPDATE SET
            expires_at = EXCLUDED.expires_at,
            updated_at = EXCLUDED.updated_at
        RETURNING 
            id, user_id, session_id, items_count, subtotal, version, 
            created_at, updated_at, expires_at,
            promo_code, discount, total, promo_metadata
    `

	var result model.Cart
	err := r.pool.QueryRow(ctx, query,
		cart.UserID,
		cart.SessionID,
		cart.ItemsCount,
		cart.Subtotal,
		cart.Version,
		cart.CreatedAt,
		cart.UpdatedAt,
		cart.ExpiresAt,
		cart.PromoCode,     // ✅ Add
		cart.Discount,      // ✅ Add
		cart.Total,         // ✅ Add
		cart.PromoMetadata, // ✅ Add
	).Scan(
		&result.ID,
		&result.UserID,
		&result.SessionID,
		&result.ItemsCount,
		&result.Subtotal,
		&result.Version,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.ExpiresAt,
		&result.PromoCode,     // ✅ Add
		&result.Discount,      // ✅ Add
		&result.Total,         // ✅ Add
		&result.PromoMetadata, // ✅ Add
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create or get cart: %w", err)
	}

	return &result, nil
}

// AddItem implements RepositoryInterface.AddItem
// INSERT or UPDATE if item already exists
func (r *postgresRepository) AddItem(ctx context.Context, item *model.CartItem) (*model.CartItem, error) {
	query := `
        INSERT INTO cart_items (cart_id, book_id, quantity, price, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (cart_id, book_id) DO UPDATE SET
            quantity = EXCLUDED.quantity,
            price = EXCLUDED.price,
            updated_at = EXCLUDED.updated_at
        RETURNING id, cart_id, book_id, quantity, price, created_at, updated_at
    `

	var result model.CartItem
	err := r.pool.QueryRow(ctx, query,
		item.CartID,
		item.BookID,
		item.Quantity,
		item.Price,
		item.CreatedAt,
		item.UpdatedAt,
	).Scan(
		&result.ID,
		&result.CartID,
		&result.BookID,
		&result.Quantity,
		&result.Price,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add item: %w", err) // ✅ Fix: thêm err
	}

	return &result, nil
}

func (r *postgresRepository) GetItemsWithBooks(ctx context.Context, cartID uuid.UUID, page int, limit int) ([]model.CartItemWithBook, int, error) {
	// Handle fetch all case
	var limitClause string

	var args []interface{}

	if limit > 0 && page > 0 {
		offset := (page - 1) * limit
		limitClause = fmt.Sprintf("LIMIT $%d OFFSET $%d", len(args)+2, len(args)+3)
		args = append(args, cartID, limit, offset)
	} else {
		// Fetch all - no limit/offset
		limitClause = ""

		args = []interface{}{cartID}
	}

	// Single query with window function for count + optimized join
	query := `
        SELECT 
            ci.id, ci.cart_id, ci.book_id, ci.quantity, ci.price, 
            ci.created_at, ci.updated_at,
            b.title, b.slug, b.cover_url,
            a.name as book_author,
            b.price as current_price,
            b.is_active,
            COALESCE(bts.available, 0) as total_stock,
            COUNT(*) OVER() as total_count
        FROM cart_items ci
        LEFT JOIN books b ON ci.book_id = b.id
        LEFT JOIN authors a ON b.author_id = a.id
        LEFT JOIN books_total_stock bts ON b.id = bts.book_id
        WHERE ci.cart_id = $1
        ORDER BY ci.created_at DESC
        ` + limitClause

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []model.CartItemWithBook
	var totalCount int

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
			&totalCount, // ✅ Scan total count from window function
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
	result, err := r.pool.Exec(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("item not found")
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
// UpdateCartPromo updates cart with promo code and metadata
func (r *postgresRepository) UpdateCartPromo(ctx context.Context, cartID uuid.UUID, version int, promoCode *string, discountAmount decimal.Decimal, metadata map[string]interface{}) error {
	// Convert metadata to JSONB
	var metadataJSON interface{}
	if metadata != nil {
		metadataJSON = metadata
	}

	query := `
        UPDATE carts
        SET 
            promo_code = $2,
            discount = $3,
            promo_metadata = $4,
            version = version + 1,
            updated_at = NOW()
        WHERE id = $1 AND version = $5
    `

	result, err := r.pool.Exec(ctx, query, cartID, promoCode, discountAmount, metadataJSON, version)
	if err != nil {
		return fmt.Errorf("failed to update cart promo: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("cart version mismatch or not found")
	}

	return nil
}

// RemoveCartPromo implements RepositoryInterface.RemoveCartPromo
func (r *postgresRepository) RemoveCartPromo(ctx context.Context, cartID uuid.UUID) error {
	query := `
		UPDATE carts
		SET 
			promo_code = NULL,
			discount = 0,
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

func (r *postgresRepository) GetByID(ctx context.Context, cartID uuid.UUID) (*model.Cart, error) {
	query := `
        SELECT 
            id, user_id, session_id, items_count, subtotal, version,
            created_at, updated_at, expires_at,
            promo_code, discount, total, promo_metadata
        FROM carts
        WHERE id = $1
    `

	var cart model.Cart
	err := r.pool.QueryRow(ctx, query, cartID).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.SessionID,
		&cart.ItemsCount,
		&cart.Subtotal,
		&cart.Version,
		&cart.CreatedAt,
		&cart.UpdatedAt,
		&cart.ExpiresAt,
		&cart.PromoCode,
		&cart.Discount, // ✅ Not pointer, scan directly
		&cart.Total,
		&cart.PromoMetadata, // ✅ Scan JSONB into map
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	return &cart, nil
}

// ==================== TRANSACTION MANAGEMENT ====================

// BeginTx starts a new database transaction
func (r *postgresRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// CommitTx commits the transaction
func (r *postgresRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// RollbackTx rolls back the transaction
func (r *postgresRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil {
		// Ignore "transaction already committed/rolled back" error
		if !errors.Is(err, pgx.ErrTxClosed) {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}
	}
	return nil
}

// ==================== TRANSACTION-AWARE CART OPERATIONS ====================

// GetByUserIDWithTx retrieves cart by user ID within a transaction
func (r *postgresRepository) GetByUserIDWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Cart, error) {
	query := `
        SELECT 
            id, user_id, session_id, items_count, subtotal, version,
            created_at, updated_at, expires_at,
						promo_code, discount, total, promo_metadata
        FROM carts
        WHERE user_id = $1
        FOR UPDATE -- Lock row for transaction
    `

	var cart model.Cart
	err := tx.QueryRow(ctx, query, userID).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.SessionID,
		&cart.ItemsCount,
		&cart.Subtotal,
		&cart.Version,
		&cart.CreatedAt,
		&cart.UpdatedAt,
		&cart.ExpiresAt,
		&cart.PromoCode,
		&cart.Discount, // ✅ Not pointer, scan directly
		&cart.Total,
		&cart.PromoMetadata, // ✅ Scan JSONB into map
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	return &cart, nil
}

// CreateOrGetWithTx creates cart or returns existing one (atomic, within transaction)
func (r *postgresRepository) CreateOrGetWithTx(ctx context.Context, tx pgx.Tx, cart *model.Cart) (*model.Cart, error) {
	var conflictColumn string
	if cart.UserID != nil {
		conflictColumn = "user_id"
	} else if cart.SessionID != nil {
		conflictColumn = "session_id"
	} else {
		return nil, fmt.Errorf("either user_id or session_id must be provided")
	}

	query := `
        INSERT INTO carts (id, user_id, session_id, items_count, subtotal, version, created_at, updated_at, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (` + conflictColumn + `) 
        WHERE ` + conflictColumn + ` IS NOT NULL
        DO UPDATE SET
            expires_at = EXCLUDED.expires_at,
            updated_at = EXCLUDED.updated_at
        RETURNING id, user_id, session_id, items_count, subtotal, version, created_at, updated_at, expires_at
    `

	var result model.Cart
	err := tx.QueryRow(ctx, query,
		cart.ID,
		cart.UserID,
		cart.SessionID,
		cart.ItemsCount,
		cart.Subtotal,
		cart.Version,
		cart.CreatedAt,
		cart.UpdatedAt,
		cart.ExpiresAt,
	).Scan(
		&result.ID,
		&result.UserID,
		&result.SessionID,
		&result.ItemsCount,
		&result.Subtotal,
		&result.Version,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create or get cart: %w", err)
	}

	return &result, nil
}

// ==================== TRANSACTION-AWARE CART ITEM OPERATIONS ====================

// GetItemsByCartIDWithTx retrieves all items in a cart within transaction
func (r *postgresRepository) GetItemsByCartIDWithTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) ([]model.CartItem, error) {
	query := `
        SELECT 
            id, cart_id, book_id, quantity, price, created_at, updated_at
        FROM cart_items
        WHERE cart_id = $1
        FOR UPDATE -- Lock rows for transaction
    `

	rows, err := tx.Query(ctx, query, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart items: %w", err)
	}
	defer rows.Close()

	var items []model.CartItem
	for rows.Next() {
		var item model.CartItem
		err := rows.Scan(
			&item.ID,
			&item.CartID,
			&item.BookID,
			&item.Quantity,
			&item.Price,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cart item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cart items: %w", err)
	}

	return items, nil
}

// UpdateItemWithTx updates a cart item within transaction
func (r *postgresRepository) UpdateItemWithTx(ctx context.Context, tx pgx.Tx, item *model.CartItem) error {
	query := `
        UPDATE cart_items
        SET quantity = $1, price = $2, updated_at = $3
        WHERE id = $4
    `

	result, err := tx.Exec(ctx, query,
		item.Quantity,
		item.Price,
		item.UpdatedAt,
		item.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update cart item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("cart item not found: %s", item.ID)
	}

	return nil
}

// AddItemWithTx adds or updates cart item within transaction
func (r *postgresRepository) AddItemWithTx(ctx context.Context, tx pgx.Tx, item *model.CartItem) error {
	query := `
        INSERT INTO cart_items (cart_id, book_id, quantity, price, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (cart_id, book_id) DO UPDATE SET
            quantity = EXCLUDED.quantity,
            price = EXCLUDED.price,
            updated_at = EXCLUDED.updated_at
    `

	_, err := tx.Exec(ctx, query,
		item.CartID,
		item.BookID,
		item.Quantity,
		item.Price,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add cart item: %w", err)
	}

	return nil
}

// DeleteCartWithTx deletes a cart and its items (CASCADE) within transaction
func (r *postgresRepository) DeleteCartWithTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) error {
	query := `DELETE FROM carts WHERE id = $1`

	result, err := tx.Exec(ctx, query, cartID)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("cart not found: %s", cartID)
	}

	return nil
}

// ==================== HELPER: GET ITEM BY BOOK WITH LOCK ====================

// GetItemByBookInCartWithTx retrieves cart item by book ID within transaction (with lock)
func (r *postgresRepository) GetItemByBookInCartWithTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID, bookID uuid.UUID) (*model.CartItem, error) {
	query := `
        SELECT id, cart_id, book_id, quantity, price, created_at, updated_at
        FROM cart_items
        WHERE cart_id = $1 AND book_id = $2
        FOR UPDATE -- Lock for transaction
    `

	var item model.CartItem
	err := tx.QueryRow(ctx, query, cartID, bookID).Scan(
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
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get cart item: %w", err)
	}

	return &item, nil
}

// GetCartAndItem retrieves cart and item in single query (optimized)
func (r *postgresRepository) GetCartAndItem(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID) (*model.Cart, *model.CartItem, error) {
	query := `
        SELECT 
            c.id, c.user_id, c.session_id, c.items_count, c.subtotal, c.version,
            c.created_at, c.updated_at, c.expires_at,
            ci.id, ci.cart_id, ci.book_id, ci.quantity, ci.price, ci.created_at, ci.updated_at
        FROM carts c
        INNER JOIN cart_items ci ON c.id = ci.cart_id
        WHERE c.id = $1 AND ci.id = $2
    `

	var cart model.Cart
	var item model.CartItem

	err := r.pool.QueryRow(ctx, query, cartID, itemID).Scan(
		&cart.ID, &cart.UserID, &cart.SessionID, &cart.ItemsCount, &cart.Subtotal, &cart.Version,
		&cart.CreatedAt, &cart.UpdatedAt, &cart.ExpiresAt,
		&item.ID, &item.CartID, &item.BookID, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get cart and item: %w", err)
	}

	return &cart, &item, nil
}

// GetItemWithBookByID retrieves cart item with book details by item ID
func (r *postgresRepository) GetItemWithBookByID(ctx context.Context, itemID uuid.UUID) (*model.CartItemWithBook, error) {
	query := `
        SELECT 
            ci.id, ci.cart_id, ci.book_id, ci.quantity, ci.price, ci.created_at, ci.updated_at,
            b.title, b.slug, b.cover_url, a.name as author_name, b.price as current_price, b.is_active,
            COALESCE(bts.available, 0) as total_stock
        FROM cart_items ci
        LEFT JOIN books b ON ci.book_id = b.id
        LEFT JOIN authors a ON b.author_id = a.id
        LEFT JOIN books_total_stock bts ON b.id = bts.book_id
        WHERE ci.id = $1
    `

	var item model.CartItemWithBook
	err := r.pool.QueryRow(ctx, query, itemID).Scan(
		&item.ID, &item.CartID, &item.BookID, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt,
		&item.BookTitle, &item.BookSlug, &item.BookCoverURL, &item.BookAuthor, &item.CurrentPrice, &item.IsActive, &item.TotalStock,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("item not found")
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	return &item, nil
}

// GetByCode retrieves promotion by code (case-insensitive)
func (r *postgresRepository) GetPromoByCode(ctx context.Context, code string) (*promo.Promotion, error) {
	query := `
        SELECT 
            id, code, name, description,
            discount_type, discount_value, max_discount_amount,
            min_order_amount, applicable_category_ids, first_order_only,
            max_uses, max_uses_per_user, current_uses,
            starts_at, expires_at, is_active,
            created_at, updated_at
        FROM promotions
        WHERE LOWER(code) = LOWER($1)
    `

	var promo promo.Promotion
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&promo.ID,
		&promo.Code,
		&promo.Name,
		&promo.Description,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MaxDiscountAmount,
		&promo.MinOrderAmount,
		&promo.ApplicableCategoryIDs,
		&promo.FirstOrderOnly,
		&promo.MaxUses,
		&promo.MaxUsesPerUser,
		&promo.CurrentUses,
		&promo.StartsAt,
		&promo.ExpiresAt,
		&promo.IsActive,
		&promo.CreatedAt,
		&promo.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get promotion: %w", err)
	}

	return &promo, nil
}

// CountUserUsage counts how many times user has used a promotion
func (r *postgresRepository) CountUserUsage(ctx context.Context, promotionID uuid.UUID, userID uuid.UUID) (int, error) {
	query := `
        SELECT COUNT(*)
        FROM promotion_usage
        WHERE promotion_id = $1 AND user_id = $2
    `

	var count int
	err := r.pool.QueryRow(ctx, query, promotionID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user usage: %w", err)
	}

	return count, nil
}

// UserHasCompletedOrders checks if user has any completed orders
func (r *postgresRepository) UserHasCompletedOrders(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 FROM orders
            WHERE user_id = $1 AND status IN ('completed', 'delivered')
        )
    `

	var hasOrders bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&hasOrders)
	if err != nil {
		return false, fmt.Errorf("failed to check user orders: %w", err)
	}

	return hasOrders, nil
}
func (r *postgresRepository) ClearCartPromo(ctx context.Context, cartID uuid.UUID) error {
	query := `
        UPDATE carts
        SET 
            promo_code = NULL,
            discount = 0,
            promo_metadata = NULL,
            updated_at = NOW()
        WHERE id = $1
    `

	result, err := r.pool.Exec(ctx, query, cartID)
	if err != nil {
		return fmt.Errorf("failed to clear cart promo: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("cart not found")
	}

	return nil
}

// GetUserEmail retrieves user email by user ID
func (r *postgresRepository) GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	query := `SELECT email FROM users WHERE id = $1`

	var email string
	err := r.pool.QueryRow(ctx, query, userID).Scan(&email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("user not found")
		}
		return "", fmt.Errorf("get user email: %w", err)
	}

	return email, nil
}
