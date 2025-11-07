package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Cart represents shopping cart for authenticated and anonymous users
type Cart struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	UserID     *uuid.UUID      `json:"user_id" db:"user_id"`
	SessionID  *string         `json:"session_id" db:"session_id"`
	ItemsCount int             `json:"items_count" db:"items_count"`
	Subtotal   decimal.Decimal `json:"subtotal" db:"subtotal"`
	Version    int             `json:"version" db:"version"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
	ExpiresAt  time.Time       `json:"expires_at" db:"expires_at"`
}

// CartItem represents items in shopping cart
type CartItem struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	CartID    uuid.UUID       `json:"cart_id" db:"cart_id"`
	BookID    uuid.UUID       `json:"book_id" db:"book_id"`
	Quantity  int             `json:"quantity" db:"quantity"`
	Price     decimal.Decimal `json:"price" db:"price"` // Snapshot price at time of adding
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}
