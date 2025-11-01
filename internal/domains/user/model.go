package user

import (
	"time"

	"github.com/google/uuid"
)

// User là domain entity - ánh xạ 1:1 với bảng users trong DB
// Match 100% với migration 000001_create_users_table.up.sql
type User struct {
	// Identity
	ID    uuid.UUID `db:"id" json:"id"`
	Email string    `db:"email" json:"email"`

	// Authentication
	PasswordHash string `db:"password_hash" json:"-"` // Never expose in JSON

	// Profile
	FullName string  `db:"full_name" json:"full_name"` // Lưu ý: DB dùng full_name không phải fullname
	Phone    *string `db:"phone" json:"phone,omitempty"`

	// Authorization - ĐÚNG 4 ROLES từ migration
	Role     Role `db:"role" json:"role"`
	IsActive bool `db:"is_active" json:"is_active"`

	// Loyalty Program
	Points int `db:"points" json:"points"`

	// Email Verification
	IsVerified                 bool       `db:"is_verified" json:"is_verified"`
	VerificationToken          *string    `db:"verification_token" json:"-"`
	VerificationSentAt         *time.Time `db:"verification_sent_at" json:"-"`
	VerificationTokenExpiresAt *time.Time `db:"verification_token_expires_at"` // ← ADD
	// Password Reset
	ResetToken          *string    `db:"reset_token" json:"-"`
	ResetTokenExpiresAt *time.Time `db:"reset_token_expires_at" json:"-"`

	// Activity
	LastLoginAt *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`

	// Timestamps
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"` // Soft delete
}

// Role enum - CHÍNH XÁC theo migration (4 roles)
type Role string

const (
	RoleUser      Role = "user"      // Regular customer
	RoleAdmin     Role = "admin"     // Full system access
	RoleWarehouse Role = "warehouse" // Inventory management
	RoleCSKH      Role = "cskh"      // Customer service
)

// AllRoles returns all valid roles
func AllRoles() []Role {
	return []Role{RoleUser, RoleAdmin, RoleWarehouse, RoleCSKH}
}

// IsValid kiểm tra role hợp lệ
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin, RoleWarehouse, RoleCSKH:
		return true
	}
	return false
}

// String implements Stringer interface
func (r Role) String() string {
	return string(r)
}

// HasPermission kiểm tra quyền theo hierarchy
// Hierarchy: admin > cskh > warehouse > user
func (r Role) HasPermission(requiredRole Role) bool {
	hierarchy := map[Role]int{
		RoleUser:      1,
		RoleWarehouse: 2,
		RoleCSKH:      3,
		RoleAdmin:     4,
	}
	return hierarchy[r] >= hierarchy[requiredRole]
}

// CanManageInventory kiểm tra quyền quản lý kho
func (r Role) CanManageInventory() bool {
	return r == RoleAdmin || r == RoleWarehouse
}

// CanManageOrders kiểm tra quyền quản lý đơn hàng
func (r Role) CanManageOrders() bool {
	return r == RoleAdmin || r == RoleCSKH
}

// IsDeleted kiểm tra user đã bị xóa (soft delete)
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// IsPasswordResetValid kiểm tra token reset password còn hạn
func (u *User) IsPasswordResetValid() bool {
	if u.ResetToken == nil || u.ResetTokenExpiresAt == nil {
		return false
	}
	return time.Now().Before(*u.ResetTokenExpiresAt)
}

// IsVerificationValid kiểm tra token verification còn hạn (24h)
func (u *User) IsVerificationValid() bool {
	if u.VerificationToken == nil || u.VerificationSentAt == nil {
		return false
	}
	// Token valid trong 24h theo migration
	expiresAt := u.VerificationSentAt.Add(24 * time.Hour)
	return time.Now().Before(expiresAt)
}

// Sanitize removes sensitive data before sending to client
func (u *User) Sanitize() {
	u.PasswordHash = ""
	u.VerificationToken = nil
	u.ResetToken = nil
}
