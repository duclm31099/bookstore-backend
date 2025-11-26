package repository

import (
	"context"

	"bookstore-backend/internal/domains/cart/model"
	promo "bookstore-backend/internal/domains/promotion/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// RepositoryInterface defines data access methods for cart
type RepositoryInterface interface {
	// GetByUserID retrieves cart for authenticated user
	// Returns: nil if not exists (don't treat as error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Cart, error)

	GetByID(ctx context.Context, cartID uuid.UUID) (*model.Cart, error)
	GetItemsByCartID(ctx context.Context, cartID uuid.UUID) ([]model.CartItem, error)
	// GetBySessionID retrieves cart for anonymous user
	// Returns: nil if not exists (don't treat as error)
	GetBySessionID(ctx context.Context, sessionID string) (*model.Cart, error)

	// Create creates new cart
	Create(ctx context.Context, cart *model.Cart) error
	CreateOrGet(ctx context.Context, cart *model.Cart) (*model.Cart, error)
	// UpdateExpiration extends cart expiration by 30 days
	UpdateExpiration(ctx context.Context, cartID uuid.UUID) error

	// AddItem adds or updates item in cart
	AddItem(ctx context.Context, item *model.CartItem) (*model.CartItem, error)

	// GetItemsWithBooks retrieves cart items with book details (via JOIN)
	// Returns: items, total_count, error
	GetItemsWithBooks(ctx context.Context, cartID uuid.UUID, page int, limit int) ([]*model.CartItemWithBook, int, error)

	// GetItemByID retrieves single cart item by ID
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*model.CartItem, error)

	// GetItemByBookInCart checks if book already in cart
	// Returns: item if exists, nil if not
	GetItemByBookInCart(ctx context.Context, cartID uuid.UUID, bookID uuid.UUID) (*model.CartItem, error)
	UpdateCartPromo(ctx context.Context, cartID uuid.UUID, version int, promoCode *string, discountAmount decimal.Decimal, metadata map[string]interface{}) error
	// DeleteExpiredCarts deletes expired carts (background job)
	// Returns: number of deleted carts
	DeleteExpiredCarts(ctx context.Context) (int, error)

	// UpdateItem updates cart item quantity
	UpdateItem(ctx context.Context, item *model.CartItem) error

	// DeleteItem removes item from cart
	DeleteItem(ctx context.Context, itemID uuid.UUID) error
	ClearCartPromo(ctx context.Context, cartID uuid.UUID) error
	// DeleteCart removes cart and all its items (CASCADE)
	DeleteCart(ctx context.Context, cartID uuid.UUID) error
	GetPromoByCode(ctx context.Context, code string) (*promo.Promotion, error)
	CountUserUsage(ctx context.Context, promotionID uuid.UUID, userID uuid.UUID) (int, error)
	UserHasCompletedOrders(ctx context.Context, userID uuid.UUID) (bool, error)
	// TransferItem moves item from one cart to another
	// Used during cart merge
	TransferItem(ctx context.Context, item *model.CartItem, targetCartID uuid.UUID) error
	ClearCartItems(ctx context.Context, cartID uuid.UUID) (int, error) // Returns count deleted

	// UpdateCartPromo updates cart with promo code and discount
	GetCartAndItem(ctx context.Context, cartID uuid.UUID, itemID uuid.UUID) (*model.Cart, *model.CartItem, error)
	// RemoveCartPromo removes promo from cart
	GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error)
	RemoveCartPromo(ctx context.Context, cartID uuid.UUID) error
	GetItemWithBookByID(ctx context.Context, itemID uuid.UUID) (*model.CartItemWithBook, error)
	// Transaction-aware methods
	BeginTx(ctx context.Context) (pgx.Tx, error)
	CommitTx(ctx context.Context, tx pgx.Tx) error
	RollbackTx(ctx context.Context, tx pgx.Tx) error
	GetByUserIDWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Cart, error)
	CreateOrGetWithTx(ctx context.Context, tx pgx.Tx, cart *model.Cart) (*model.Cart, error)
	GetItemsByCartIDWithTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) ([]model.CartItem, error)
	UpdateItemWithTx(ctx context.Context, tx pgx.Tx, item *model.CartItem) error
	AddItemWithTx(ctx context.Context, tx pgx.Tx, item *model.CartItem) error
	DeleteCartWithTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) error
}
