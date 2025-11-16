package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5"        // ← Add
	_ "github.com/jackc/pgx/v5/pgconn" // ← Add
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/pgxpool" // ← Add
	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL driver

	user "bookstore-backend/internal/domains/user"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"
)

// postgresRepository là concrete implementation của user.Repository interface
// Struct này là PRIVATE (lowercase 'p') - chỉ có thể truy cập trong package này
// Pattern này gọi là "Hide implementation, expose interface"
type postgresRepository struct {
	pool  *pgxpool.Pool // PostgreSQL connection pool
	cache cache.Cache   // Redis cache layer (injected dependency)
}

// NewPostgresRepository là constructor function - tạo instance mới của repository
// Function này là PUBLIC (uppercase 'N') - có thể gọi từ bên ngoài package
//
// Tại sao return interface thay vì concrete type?
// - Giúp code phụ thuộc vào abstraction, không phụ thuộc vào implementation
// - Dễ dàng swap implementation (từ Postgres sang MySQL) mà không cần sửa code
// - Dễ dàng mock trong testing
//
// Pattern này gọi là "Dependency Injection via Constructor"
func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) user.Repository {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}

// ========================================
// BASIC CRUD OPERATIONS
// ========================================

// Create tạo user mới trong database
// Context: cho phép cancel operation, set timeout, pass metadata qua request chain
func (r *postgresRepository) Create(ctx context.Context, u *user.User) (uuid.UUID, error) {
	// SQL query - sử dụng $1, $2, ... placeholders để tránh SQL injection
	query := `
		INSERT INTO users (
			 email, password_hash, full_name, phone, role,
			is_active, points, is_verified, 
			verification_token, verification_sent_at,
			verification_token_expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11,
			$12, $13
		)
		RETURNING id
	`

	// ExecContext: thực thi query với context
	// - Context cho phép cancel query khi timeout hoặc user cancel request
	// - Returns: sql.Result (affected rows, last insert id) và error
	var userID uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		u.Email,
		u.PasswordHash,
		u.FullName,
		u.Phone,
		u.Role,
		u.IsActive,
		u.Points,
		u.IsVerified,
		u.VerificationToken,
		u.VerificationSentAt,
		u.VerificationTokenExpiresAt,
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(&userID)

	if err != nil {
		// Type assertion: convert error sang *pq.Error để kiểm tra PostgreSQL error code
		// pq.Error chứa thông tin chi tiết về lỗi từ PostgreSQL
		if pqErr, ok := err.(*pq.Error); ok {
			// Error code 23505 = unique_violation (email đã tồn tại)
			// Mapping PostgreSQL error thành domain error
			if pqErr.Code == "23505" {
				if strings.Contains(pqErr.Message, "email") {
					return uuid.Nil, user.ErrEmailAlreadyExists
				}
			}
		}
		// Wrap error với context để debugging dễ hơn
		// fmt.Errorf với %w verb cho phép unwrap error chain
		return uuid.Nil, err
	}

	return userID, nil
}

// FindByID tìm user theo UUID với Redis caching
// Implement "Cache-Aside Pattern" (aka "Lazy Loading")
func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	// STEP 1: CHECK CACHE FIRST
	// Cache key naming convention: "entity:id"
	cacheKey := fmt.Sprintf("user:%s", id.String())

	// Khai báo biến để nhận data từ cache
	var u user.User

	// cache.Get trả về (found bool, error)
	// found = true: data tồn tại trong cache và đã unmarshal vào &u
	// found = false: cache miss, cần query database
	found, err := r.cache.Get(ctx, cacheKey, &u)
	if err == nil && found {
		// Cache HIT - return ngay, không cần query DB
		return &u, nil
	}

	// STEP 2: CACHE MISS - QUERY DATABASE
	query := `
		SELECT 
			id, email, password_hash, full_name, phone, role,
			is_active, points, is_verified,
			verification_token, verification_sent_at,
			reset_token, reset_token_expires_at,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	// QueryRowContext: query single row với context
	// Returns: *sql.Row - phải gọi .Scan() để lấy data
	err = r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Phone,
		&u.Role,
		&u.IsActive,
		&u.Points,
		&u.IsVerified,
		&u.VerificationToken,
		&u.VerificationSentAt,
		&u.ResetToken,
		&u.ResetTokenExpiresAt,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)

	if err != nil {
		// errors.Is: so sánh error với sentinel error
		// Tốt hơn err == sql.ErrNoRows vì nó unwrap error chain
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	// STEP 3: SET CACHE FOR FUTURE REQUESTS
	// TTL = 15 minutes, balance giữa freshness và performance
	// Ignore cache set error - không nên fail request nếu cache unavailable
	_ = r.cache.Set(ctx, cacheKey, &u, 15*time.Minute)

	return &u, nil
}

// FindByEmail tìm user theo email (dùng cho login)
// Không cache vì email lookup chỉ xảy ra khi login (ít thường xuyên)
func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT 
			id, email, password_hash, full_name, phone, role,
			is_active, points, is_verified, last_login_at,
			created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	var u user.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Phone,
		&u.Role,
		&u.IsActive,
		&u.Points,
		&u.IsVerified,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &u, nil
}

// UpdateProfile updates user profile (name, phone)
func (r *postgresRepository) UpdateProfile(ctx context.Context, id string, fullName, phone *string) error {
	query := `
		UPDATE users
		SET 
			full_name = COALESCE($2, full_name),
			phone = COALESCE($3, phone),
			updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, id, fullName, phone, time.Now())
	return err
}

// Update cập nhật thông tin user và invalidate cache
func (r *postgresRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET 
			full_name = $2,
			phone = $3,
			updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	// Set updated_at timestamp
	u.UpdatedAt = time.Now()

	// ExecContext returns sql.Result
	result, err := r.pool.Exec(ctx, query,
		u.ID,
		u.FullName,
		u.Phone,
		u.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Nếu = 0, nghĩa là không tìm thấy user (có thể đã bị xóa)
	// CommandTag.RowsAffected(): số rows affected
	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// INVALIDATE CACHE
	// Write-through pattern: xóa cache ngay sau khi update DB
	// Next read sẽ cache miss và load fresh data từ DB
	cacheKey := fmt.Sprintf("user:%s", u.ID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// Delete thực hiện soft delete (set deleted_at)
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", id.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// ========================================
// AUTHENTICATION SPECIFIC
// ========================================

// FindByVerificationToken tìm user theo verification token
// Token chỉ valid trong 24h từ khi gửi (theo migration)
func (r *postgresRepository) FindByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	query := `
		SELECT 
			id, email, full_name, is_verified, 
			verification_sent_at, created_at
		FROM users
		WHERE verification_token = $1 
		  AND deleted_at IS NULL
		  AND verification_sent_at > NOW() - INTERVAL '24 hours'
	`

	var u user.User
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&u.ID,
		&u.Email,
		&u.FullName,
		&u.IsVerified,
		&u.VerificationSentAt,
		&u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrInvalidToken
		}
		return nil, fmt.Errorf("find by verification token: %w", err)
	}

	return &u, nil
}

// FindByResetToken tìm user theo password reset token
// Token có expiry time trong DB (reset_token_expires_at)
func (r *postgresRepository) FindByResetToken(ctx context.Context, token string) (*user.User, error) {
	query := `
		SELECT 
			id, email, reset_token_expires_at
		FROM users
		WHERE reset_token = $1 
		  AND deleted_at IS NULL
		  AND reset_token_expires_at > NOW()
	`

	var u user.User
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&u.ID,
		&u.Email,
		&u.ResetTokenExpiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrInvalidToken
		}
		return nil, fmt.Errorf("find by reset token: %w", err)
	}

	return &u, nil
}

// UpdateVerificationToken sets email verification token
func (r *postgresRepository) UpdateVerificationToken(ctx context.Context, id string, token *string, expiredAt *time.Time) error {
	query := `
		UPDATE users
		SET 
			verification_token = $2,
			verification_sent_at = $3,
			verification_token_expires_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, id, token, time.Now(), expiredAt)
	return err
}

// VerifyEmail marks user email as verified
func (r *postgresRepository) VerifyEmail(ctx context.Context, token string) error {
	query := `
		UPDATE users
		SET 
			is_verified = true,
			verification_token = NULL,
			verification_sent_at = NULL,
			verification_token_expires_at = NULL,
			updated_at = $2
		WHERE verification_token = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, token, time.Now())
	if err != nil {
		return err
	}

	rows := result.RowsAffected()
	if rows == 0 {
		return user.ErrInvalidToken
	}

	return nil
}

// UpdateResetToken sets password reset token
func (r *postgresRepository) UpdateResetToken(ctx context.Context, id string, token *string, expiresAt *time.Time) error {
	query := `
		UPDATE users
		SET 
			reset_token = $2,
			reset_token_expires_at = $3,
			updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, id, token, expiresAt, time.Now())
	return err
}

// UpdatePassword cập nhật password và clear reset token
func (r *postgresRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET 
			password_hash = $2,
			reset_token = NULL,
			reset_token_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// MarkAsVerified đánh dấu user đã verify email
func (r *postgresRepository) MarkAsVerified(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET 
			is_verified = true,
			verification_token = NULL,
			verification_sent_at = NULL,
			verification_token_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("mark as verified: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// UpdateLastLogin cập nhật last_login_at timestamp
func (r *postgresRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET last_login_at = NOW()
		WHERE id = $1
	`

	// Ignore result - không cần check RowsAffected
	// Vì update last_login không critical
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// ========================================
// ADMIN FUNCTIONS
// ========================================

// List trả về danh sách users với filters và pagination
// Sử dụng "Dynamic Query Building" pattern
func (r *postgresRepository) List(ctx context.Context, req user.ListUsersRequest) ([]user.User, int, error) {
	// DYNAMIC QUERY BUILDING
	// strings.Builder: efficient string concatenation (tốt hơn += string)
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT 
			id, email, full_name, phone, role, 
			is_active, points, is_verified, 
			last_login_at, created_at
		FROM users
		WHERE deleted_at IS NULL
	`)

	// args: slice chứa query parameters
	// []interface{}: slice of "any type" - có thể chứa string, int, bool, ...
	args := []interface{}{}
	argPos := 1 // PostgreSQL placeholders start at $1

	// APPLY FILTERS DYNAMICALLY
	if req.Role != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND role = $%d", argPos))
		args = append(args, *req.Role)
		argPos++
	}

	if req.IsVerified != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND is_verified = $%d", argPos))
		args = append(args, *req.IsVerified)
		argPos++
	}

	if req.IsActive != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	// SEARCH by email or name using ILIKE (case-insensitive)
	if req.Search != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND (email ILIKE $%d OR full_name ILIKE $%d)", argPos, argPos))
		// %pattern%: search anywhere in string
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern)
		argPos++
	}

	// COUNT TOTAL (before pagination)
	// Subquery: wrap main query để đếm total rows
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS t", queryBuilder.String())
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// SORTING
	// Validate sort field để tránh SQL injection (vì không thể dùng placeholder cho column names)
	validSortFields := map[string]bool{
		"created_at":    true,
		"points":        true,
		"last_login_at": true,
		"email":         true,
	}
	if !validSortFields[req.SortBy] {
		req.SortBy = "created_at" // Fallback to default
	}

	// Validate sort order
	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", req.SortBy, req.SortOrder))

	// PAGINATION
	// LIMIT: số rows trả về
	// OFFSET: bỏ qua bao nhiêu rows
	// Formula: offset = (page - 1) * limit
	limit := req.Limit
	if limit == 0 {
		limit = 20 // Default
	}
	offset := (req.Page - 1) * limit

	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, limit, offset)

	// EXECUTE QUERY
	query := queryBuilder.String()
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	// defer: đảm bảo rows.Close() được gọi khi function return (cleanup resources)
	// Quan trọng: phải close rows để tránh memory leak
	defer rows.Close()

	// SCAN ROWS
	// make(): khởi tạo slice với capacity để tránh reallocation
	users := make([]user.User, 0, limit)

	// rows.Next(): iterate qua từng row, return false khi hết
	for rows.Next() {
		var u user.User
		err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.FullName,
			&u.Phone,
			&u.Role,
			&u.IsActive,
			&u.Points,
			&u.IsVerified,
			&u.LastLoginAt,
			&u.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return users, total, nil
}

// UpdateRole cập nhật role của user (admin only)
func (r *postgresRepository) UpdateRole(ctx context.Context, userID uuid.UUID, role user.Role) error {
	query := `
		UPDATE users
		SET role = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, userID, role)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// UpdateStatus activate/deactivate user
func (r *postgresRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	query := `
		UPDATE users
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, userID, isActive)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// ========================================
// LOYALTY POINTS (TRANSACTION-SAFE)
// ========================================

// AddPoints tăng points - transaction safe với atomic increment
func (r *postgresRepository) AddPoints(ctx context.Context, userID uuid.UUID, points int) error {
	// points = points + $2: atomic operation trong database
	// Không cần lock vì DB xử lý concurrency
	query := `
		UPDATE users
		SET points = points + $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, userID, points)
	if err != nil {
		return fmt.Errorf("add points: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// DeductPoints giảm points với validation (đảm bảo points >= 0)
func (r *postgresRepository) DeductPoints(ctx context.Context, userID uuid.UUID, points int) error {
	// WHERE clause có condition: points >= $2
	// Nếu points không đủ, RowsAffected = 0
	query := `
		UPDATE users
		SET points = points - $2, updated_at = NOW()
		WHERE id = $1 
		  AND deleted_at IS NULL
		  AND points >= $2
	`

	result, err := r.pool.Exec(ctx, query, userID, points)
	if err != nil {
		return fmt.Errorf("deduct points: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Không rõ lỗi là user not found hay insufficient points
		// Cần query lại để xác định
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)"
		_ = r.pool.QueryRow(ctx, checkQuery, userID).Scan(&exists)

		if !exists {
			return user.ErrUserNotFound
		}
		return user.ErrInsufficientPoints
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// GetPoints lấy số points hiện tại
func (r *postgresRepository) GetPoints(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT points
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var points int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&points)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, user.ErrUserNotFound
		}
		return 0, fmt.Errorf("get points: %w", err)
	}

	return points, nil
}

// ========================================
// UTILITY FUNCTIONS
// ========================================

// ExistsByEmail kiểm tra email đã tồn tại chưa
// Sử dụng EXISTS - performance tốt hơn COUNT(*)
func (r *postgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	// EXISTS: return true ngay khi tìm thấy 1 row (không scan toàn bộ table)
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM users 
			WHERE email = $1 AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists by email: %w", err)
	}

	return exists, nil
}

// CountByRole đếm số user theo role (for analytics)
func (r *postgresRepository) CountByRole(ctx context.Context, role user.Role) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM users
		WHERE role = $1 AND deleted_at IS NULL
	`

	var count int
	err := r.pool.QueryRow(ctx, query, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by role: %w", err)
	}

	return count, nil
}

// InvalidateUserCache - Remove user from cache
// This method clears the cached user data when it's updated
// func (r *postgresRepository) InvalidateUserCache(
// 	ctx context.Context,
// 	userID uuid.UUID,
// ) error {
// 	if r.cache == nil {
// 		// Cache not configured, skip
// 		return nil
// 	}

// 	cacheKey := fmt.Sprintf("user:%s", userID.String())

// 	err := r.cache.Delete(ctx, cacheKey)
// 	if err != nil && err != redis.Nil {
// 		// Log but don't fail - cache miss is not critical
// 		logger.Info("failed to invalidate user cache",
// 			map[string]interface{}{
// 				"user_id": userID.String(),
// 				"error":   err.Error(),
// 			})
// 		return nil // Don't fail the operation
// 	}
// 	return nil
// }

func (r *postgresRepository) DeleteExpiredVerifyTokens(ctx context.Context, cutoffTime time.Time) (int, error) {
	query := `
		UPDATE users
		SET 
			verification_token = NULL,
			verification_sent_at = NULL,
			verification_token_expires_at = NULL
			updated_at = NOW()
		WHERE
			verification_token IS NOT NULL
			AND verification_sent_at IS NOT NULL
			AND verification_token_expires_at < $1
			AND is_verified = false
	`

	result, err := r.pool.Exec(ctx, query, cutoffTime)
	if err != nil {
		logger.Error("Query delete expired verify token failed due to ", err)
		return 0, err
	}

	if result.RowsAffected() == 0 {
		logger.Error("Query delete expired verify token failed -> ", err)
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func (r *postgresRepository) DeleteExpiredResetTokens(ctx context.Context, cutoffTime time.Time) (int, error) {
	query := `
		UPDATE users
		SET 
			reset_token = NULL
			reset_token_expires_at = NULL
			updated_at = NOW()
		WHERE
			reset_token IS NOT NULL
			AND reset_token_expires_at < $1
			AND reset_token_expires_at IS NOT NULL
	`

	result, err := r.pool.Exec(ctx, query, cutoffTime)
	if err != nil {
		logger.Error("Query delete expired reset token failed due to ", err)
		return 0, err
	}

	if result.RowsAffected() == 0 {
		logger.Error("Query delete expired reset token failed -> ", err)
		return 0, err
	}

	return int(result.RowsAffected()), nil
}
