package model

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a product review entity
type Review struct {
	ID      uuid.UUID `json:"id"`
	UserID  uuid.UUID `json:"user_id"`
	BookID  uuid.UUID `json:"book_id"`
	OrderID uuid.UUID `json:"order_id"`

	// Content
	Rating  int      `json:"rating"` // 1-5
	Title   *string  `json:"title"`
	Content string   `json:"content"`
	Images  []string `json:"images"`

	// Verification & Moderation
	IsVerifiedPurchase bool    `json:"is_verified_purchase"`
	IsApproved         bool    `json:"is_approved"`
	IsFeatured         bool    `json:"is_featured"`
	AdminNote          *string `json:"admin_note"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CanBeEdited checks if review can be edited by user
func (r *Review) CanBeEdited() bool {
	// Can edit within 7 days of creation
	return time.Since(r.CreatedAt) < 7*24*time.Hour
}

// CanBeDeleted checks if review can be deleted by user
func (r *Review) CanBeDeleted() bool {
	// Can delete within 30 days of creation
	return time.Since(r.CreatedAt) < 30*24*time.Hour
}
