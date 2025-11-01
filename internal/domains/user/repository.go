package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository định nghĩa contract cho data access layer
// Interface này cho phép:
// - Swap implementation dễ dàng (Postgres -> MySQL -> MongoDB)
// - Mock trong unit tests
// - Tuân thủ Dependency Inversion Principle (SOLID)
type Repository interface {
	// ========================================
	// BASIC CRUD
	// ========================================

	// Create tạo user mới
	// Returns: ErrEmailAlreadyExists nếu email đã tồn tại
	Create(ctx context.Context, user *User) error

	// FindByID tìm user theo ID
	// Returns: ErrUserNotFound nếu không tìm thấy hoặc đã bị xóa (soft delete)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindByEmail tìm user theo email (dùng cho login)
	// Returns: ErrUserNotFound nếu không tìm thấy
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Update cập nhật thông tin user
	// Returns: ErrUserNotFound nếu user không tồn tại
	Update(ctx context.Context, user *User) error

	// Delete soft delete user (set deleted_at)
	// Returns: ErrUserNotFound nếu user không tồn tại
	Delete(ctx context.Context, id uuid.UUID) error

	// ========================================
	// AUTHENTICATION SPECIFIC
	// ========================================

	// FindByVerificationToken tìm user theo verification token
	// Chỉ trả về user nếu token còn hạn (< 24h từ verification_sent_at)
	FindByVerificationToken(ctx context.Context, token string) (*User, error)

	// FindByResetToken tìm user theo password reset token
	// Chỉ trả về user nếu reset_token_expires_at > NOW()
	FindByResetToken(ctx context.Context, token string) (*User, error)

	// UpdatePassword cập nhật password và clear reset token
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error

	// MarkAsVerified đánh dấu user đã verify email
	MarkAsVerified(ctx context.Context, userID uuid.UUID) error

	// UpdateLastLogin cập nhật last_login_at
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error

	// ========================================
	// ADMIN FUNCTIONS
	// ========================================

	// List trả về danh sách users với filters và pagination
	List(ctx context.Context, req ListUsersRequest) ([]User, int, error)

	// UpdateRole cập nhật role của user (admin only)
	UpdateRole(ctx context.Context, userID uuid.UUID, role Role) error

	// UpdateStatus activate/deactivate user (admin only)
	UpdateStatus(ctx context.Context, userID uuid.UUID, isActive bool) error

	// ========================================
	// LOYALTY POINTS
	// ========================================

	// AddPoints tăng points (transaction-safe)
	AddPoints(ctx context.Context, userID uuid.UUID, points int) error

	// DeductPoints giảm points với validation
	// Returns: ErrInsufficientPoints nếu points không đủ
	DeductPoints(ctx context.Context, userID uuid.UUID, points int) error

	// GetPoints lấy số points hiện tại
	GetPoints(ctx context.Context, userID uuid.UUID) (int, error)

	// ========================================
	// UTILITY
	// ========================================

	// ExistsByEmail kiểm tra email đã tồn tại chưa
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// CountByRole đếm số user theo role (for analytics)
	CountByRole(ctx context.Context, role Role) (int, error)
}
