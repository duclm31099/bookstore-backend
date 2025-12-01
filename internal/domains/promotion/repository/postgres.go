package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"bookstore-backend/internal/domains/promotion/model"
	"bookstore-backend/pkg/logger"
)

// PostgresRepository triển khai Repository interface với PostgreSQL
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository tạo instance mới
func NewPostgresRepository(db *pgxpool.Pool) PromotionRepository {
	return &PostgresRepository{db: db}
}

// -------------------------------------------------------------------
// READ OPERATIONS
// -------------------------------------------------------------------

// FindByID tìm promotion theo ID
func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Promotion, error) {
	query := `
		SELECT 
			id, code, name, description,
			discount_type, discount_value, max_discount_amount,
			min_order_amount, applicable_category_ids, first_order_only,
			max_uses, max_uses_per_user, current_uses,
			starts_at, expires_at, is_active, version,
			created_at, updated_at
		FROM promotions
		WHERE id = $1
	`

	var p model.Promotion
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID,                    // id
		&p.Code,                  // code
		&p.Name,                  // name
		&p.Description,           // description (nullable)
		&p.DiscountType,          // discount_type
		&p.DiscountValue,         // discount_value
		&p.MaxDiscountAmount,     // max_discount_amount (nullable)
		&p.MinOrderAmount,        // min_order_amount
		&p.ApplicableCategoryIDs, // applicable_category_ids (array)
		&p.FirstOrderOnly,        // first_order_only
		&p.MaxUses,               // max_uses (nullable)
		&p.MaxUsesPerUser,        // max_uses_per_user
		&p.CurrentUses,           // current_uses
		&p.StartsAt,              // starts_at
		&p.ExpiresAt,             // expires_at
		&p.IsActive,              // is_active
		&p.Version,               // version
		&p.CreatedAt,             // created_at
		&p.UpdatedAt,             // updated_at
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("find promotion by id: %w", err)
	}

	return &p, nil
}

// FindByCode tìm promotion theo code (không filter active/time)
func (r *PostgresRepository) FindByCode(ctx context.Context, code string) (*model.Promotion, error) {
	query := `
		SELECT 
			id, code, name, description,
			discount_type, discount_value, max_discount_amount,
			min_order_amount, applicable_category_ids, first_order_only,
			max_uses, max_uses_per_user, current_uses,
			starts_at, expires_at, is_active, version,
			created_at, updated_at
		FROM promotions
		WHERE LOWER(code) = LOWER($1)
	`

	var p model.Promotion
	err := r.db.QueryRow(ctx, query, code).Scan(
		&p.ID,                    // id
		&p.Code,                  // code
		&p.Name,                  // name
		&p.Description,           // description (nullable)
		&p.DiscountType,          // discount_type
		&p.DiscountValue,         // discount_value
		&p.MaxDiscountAmount,     // max_discount_amount (nullable)
		&p.MinOrderAmount,        // min_order_amount
		&p.ApplicableCategoryIDs, // applicable_category_ids (array)
		&p.FirstOrderOnly,        // first_order_only
		&p.MaxUses,               // max_uses (nullable)
		&p.MaxUsesPerUser,        // max_uses_per_user
		&p.CurrentUses,           // current_uses
		&p.StartsAt,              // starts_at
		&p.ExpiresAt,             // expires_at
		&p.IsActive,              // is_active
		&p.Version,               // version
		&p.CreatedAt,             // created_at
		&p.UpdatedAt,             // updated_at
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("find promotion by code: %w", err)
	}

	return &p, nil
}

// FindByCodeActive tìm promotion active theo code
//
// Business Logic:
// - is_active = true
// - starts_at <= NOW <= expires_at
// - Nếu có max_uses: current_uses < max_uses
//
// Note: Query này được optimize với index idx_promotions_active
func (r *PostgresRepository) FindByCodeActive(ctx context.Context, code string) (*model.Promotion, error) {
	query := `
		SELECT 
			id, code, name, description,
			discount_type, discount_value, max_discount_amount,
			min_order_amount, applicable_category_ids, first_order_only,
			max_uses, COALESCE(max_uses_per_user, 0) AS max_uses_per_user, current_uses,
			starts_at, expires_at, is_active, version,
			created_at, updated_at
		FROM promotions
		WHERE LOWER(code) = LOWER($1)
		  AND is_active = true
		  AND starts_at <= NOW()
		  AND expires_at >= NOW()
		  AND (max_uses IS NULL OR current_uses < max_uses)
	`

	var p model.Promotion
	err := r.db.QueryRow(ctx, query, code).Scan(
		&p.ID,                    // id
		&p.Code,                  // code
		&p.Name,                  // name
		&p.Description,           // description (nullable)
		&p.DiscountType,          // discount_type
		&p.DiscountValue,         // discount_value
		&p.MaxDiscountAmount,     // max_discount_amount (nullable)
		&p.MinOrderAmount,        // min_order_amount
		&p.ApplicableCategoryIDs, // applicable_category_ids (array)
		&p.FirstOrderOnly,        // first_order_only
		&p.MaxUses,               // max_uses (nullable)
		&p.MaxUsesPerUser,        // max_uses_per_user
		&p.CurrentUses,           // current_uses
		&p.StartsAt,              // starts_at
		&p.ExpiresAt,             // expires_at
		&p.IsActive,              // is_active
		&p.Version,               // version
		&p.CreatedAt,             // created_at
		&p.UpdatedAt,             // updated_at
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("find active promotion by code: %w", err)
	}

	return &p, nil
}

// GetUserUsageCount đếm số lần user đã sử dụng promotion
//
// Note: Query này sử dụng index idx_promotion_usage_user
func (r *PostgresRepository) GetUserUsageCount(ctx context.Context, promoID, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM promotion_usage
		WHERE promotion_id = $1 AND user_id = $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, promoID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get user usage count: %w", err)
	}

	return count, nil
}

// ListActive lấy danh sách promotion active (cho public API)
//
// Params:
// - categoryID: Filter theo category (nil = tất cả)
// - page, limit: Pagination
//
// Returns: promotions, total count, error
func (r *PostgresRepository) ListActive(ctx context.Context, categoryID *uuid.UUID, page, limit int) ([]*model.Promotion, int, error) {
	offset := (page - 1) * limit

	// Build query động với category filter
	query := `
		SELECT 
			id, code, name, description,
			discount_type, discount_value, max_discount_amount,
			min_order_amount, applicable_category_ids, first_order_only,
			max_uses, COALESCE(max_uses_per_user, 0) AS max_uses_per_user, current_uses,
			starts_at, expires_at, is_active, version,
			created_at, updated_at
		FROM promotions
		WHERE is_active = true
		  AND starts_at <= NOW()
		  AND expires_at >= NOW()
	`

	args := []interface{}{}
	argIndex := 1

	// Filter theo category nếu có
	if categoryID != nil {
		query += fmt.Sprintf(" AND $%d = ANY(applicable_category_ids)", argIndex)
		args = append(args, *categoryID)
		argIndex++
	}

	query += " ORDER BY starts_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)
	logger.Info("in it", map[string]interface{}{
		"offset":     offset,
		"categoryID": categoryID,
		"args":       args,
	})
	// Execute query
	var promotions []*model.Promotion
	rows, err := r.db.Query(ctx, query, args...)
	logger.Info("after query", map[string]interface{}{
		"rows": rows,
		"eror": err,
	})
	if err != nil {
		logger.Error("get error", err)
		return nil, 0, fmt.Errorf("list active promotions: %w", err)
	}

	for rows.Next() {
		var p model.Promotion
		err := rows.Scan(
			&p.ID,                    // id
			&p.Code,                  // code
			&p.Name,                  // name
			&p.Description,           // description (nullable)
			&p.DiscountType,          // discount_type
			&p.DiscountValue,         // discount_value
			&p.MaxDiscountAmount,     // max_discount_amount (nullable)
			&p.MinOrderAmount,        // min_order_amount
			&p.ApplicableCategoryIDs, // applicable_category_ids (array)
			&p.FirstOrderOnly,        // first_order_only
			&p.MaxUses,               // max_uses (nullable)
			&p.MaxUsesPerUser,        // max_uses_per_user
			&p.CurrentUses,           // current_uses
			&p.StartsAt,              // starts_at
			&p.ExpiresAt,             // expires_at
			&p.IsActive,              // is_active
			&p.Version,               // version
			&p.CreatedAt,             // created_at
			&p.UpdatedAt,             // updated_at
		)
		if err != nil {
			return nil, 0, err
		}
		promotions = append(promotions, &p)
	}
	logger.Info("Get Query ", map[string]interface{}{
		"promotions": promotions,
	})

	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM promotions
		WHERE is_active = true
		  AND starts_at <= NOW()
		  AND expires_at >= NOW()
	`

	countArgs := []interface{}{}
	if categoryID != nil {
		countQuery += " AND $1 = ANY(applicable_category_ids)"
		countArgs = append(countArgs, *categoryID)
	}

	var total int
	err = r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		logger.Error("count error", err)
		return nil, 0, fmt.Errorf("count active promotions: %w", err)
	}

	return promotions, total, nil
}

// ListAdmin lấy danh sách promotion với filter (Admin API)
//
// Business Logic:
// - Status filter: active, expired, upcoming, all
// - Search: Tìm kiếm theo code hoặc name (case-insensitive)
// - Sort: Nhiều cách sắp xếp khác nhau
// - Calculate: usage_rate, status
func (r *PostgresRepository) ListAdmin(ctx context.Context, filter *model.ListPromotionsFilter) ([]*model.PromotionListItem, int, error) {
	offset := (filter.Page - 1) * filter.Limit

	// Build WHERE clause động
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	// Status filter
	switch filter.Status {
	case "active":
		whereClauses = append(whereClauses, "is_active = true AND NOW() BETWEEN starts_at AND expires_at")
	case "expired":
		whereClauses = append(whereClauses, "NOW() > expires_at")
	case "upcoming":
		whereClauses = append(whereClauses, "NOW() < starts_at")
	case "all":
		// Không filter
	}

	// Search filter
	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(LOWER(code) LIKE $%d OR LOWER(name) LIKE $%d)",
			argIndex, argIndex,
		))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argIndex++
	}

	// IsActive filter
	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	// Combine WHERE clauses
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build ORDER BY clause
	orderBySQL := "ORDER BY created_at DESC" // Default
	switch filter.Sort {
	case "created_at_desc":
		orderBySQL = "ORDER BY created_at DESC"
	case "expires_at_asc":
		orderBySQL = "ORDER BY expires_at ASC"
	case "usage_desc":
		orderBySQL = "ORDER BY current_uses DESC"
	case "name_asc":
		orderBySQL = "ORDER BY name ASC"
	}

	// Main query với calculated fields
	query := fmt.Sprintf(`
		SELECT 
			id, code, name,
			discount_type, discount_value, max_discount_amount,
			current_uses, max_uses,
			CASE 
				WHEN max_uses IS NOT NULL THEN (current_uses::FLOAT / max_uses * 100)
				ELSE NULL
			END as usage_rate,
			starts_at, expires_at, is_active,
			CASE 
				WHEN NOT is_active THEN 'inactive'
				WHEN NOW() < starts_at THEN 'upcoming'
				WHEN NOW() > expires_at THEN 'expired'
				WHEN max_uses IS NOT NULL AND current_uses >= max_uses THEN 'exhausted'
				ELSE 'active'
			END as status
		FROM promotions
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBySQL, argIndex, argIndex+1)

	args = append(args, filter.Limit, offset)

	// Execute query
	var items []*model.PromotionListItem
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin promotions: %w", err)
	}

	for rows.Next() {
		var i model.PromotionListItem
		err := rows.Scan(&i)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, &i)
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM promotions %s", whereSQL)
	countArgs := args[:len(args)-2] // Loại bỏ LIMIT và OFFSET

	var total int
	err = r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin promotions: %w", err)
	}

	return items, total, nil
}

// -------------------------------------------------------------------
// WRITE OPERATIONS
// -------------------------------------------------------------------

// Create tạo promotion mới
//
// Note:
// - Generate UUID nếu chưa có
// - Normalize code về uppercase
// - Set default values
func (r *PostgresRepository) Create(ctx context.Context, promo *model.Promotion) error {

	// Normalize code
	promo.Code = strings.ToUpper(promo.Code)

	query := `
		INSERT INTO promotions (
			code, name, description,
			discount_type, discount_value, max_discount_amount,
			min_order_amount, applicable_category_ids, first_order_only,
			max_uses, max_uses_per_user, current_uses,
			starts_at, expires_at, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, NOW(), NOW()
		)
		RETURNING id, code, name
	`

	err := r.db.QueryRow(ctx, query,
		promo.Code,
		promo.Name,
		promo.Description,
		promo.DiscountType,
		promo.DiscountValue,
		*promo.MaxDiscountAmount,
		promo.MinOrderAmount,
		pq.Array(promo.ApplicableCategoryIDs), // Convert []uuid.UUID to pq.Array
		promo.FirstOrderOnly,
		promo.MaxUses,
		promo.MaxUsesPerUser,
		0, // current_uses = 0
		promo.StartsAt,
		promo.ExpiresAt,
		promo.IsActive,
	).Scan(&promo.ID, &promo.Code, &promo.Name)

	if err != nil {
		logger.Error("error create::", err)
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf(" constraint violation") // model.ErrPromotionCodeExists
			}
		}
		return fmt.Errorf("create promotion: %w", err)
	}

	promo.CurrentUses = 0
	promo.Version = 0

	return nil
}

// Update cập nhật promotion với optimistic locking
//
// Business Logic:
// - Sử dụng version field để tránh race condition
// - Nếu version không khớp → promotion đã bị modify → return conflict error
// - Tự động increment version
func (r *PostgresRepository) Update(ctx context.Context, promo *model.Promotion) error {
	// Normalize code
	promo.Code = strings.ToUpper(promo.Code)

	query := `
		UPDATE promotions
		SET 
			code = $2,
			name = $3,
			description = $4,
			discount_type = $5,
			discount_value = $6,
			max_discount_amount = $7,
			min_order_amount = $8,
			applicable_category_ids = $9,
			first_order_only = $10,
			max_uses = $11,
			max_uses_per_user = $12,
			starts_at = $13,
			expires_at = $14,
			is_active = $15,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND version = $16
		RETURNING id, name, code
	`

	oldVersion := promo.Version
	logger.Info("promo info", map[string]interface{}{
		"promo":      promo,
		"oldVersion": oldVersion,
	})
	err := r.db.QueryRow(ctx, query,
		promo.ID,
		promo.Code,
		promo.Name,
		promo.Description,
		promo.DiscountType,
		promo.DiscountValue,
		promo.MaxDiscountAmount,
		promo.MinOrderAmount,
		pq.Array(promo.ApplicableCategoryIDs),
		promo.FirstOrderOnly,
		promo.MaxUses,
		promo.MaxUsesPerUser,
		promo.StartsAt,
		promo.ExpiresAt,
		promo.IsActive,
		oldVersion, // Version check (optimistic locking)
	).Scan(&promo.ID, &promo.Name, &promo.Code)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Version mismatch hoặc promotion không tồn tại
			return fmt.Errorf("Version missmatch or promotion not exist") // model.ErrPromotionVersionConflict
		}
		return fmt.Errorf("update promotion: %w", err)
	}

	return nil
}

// UpdateStatus cập nhật trạng thái active/inactive
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `
		UPDATE promotions
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id, isActive)
	if err != nil {
		return fmt.Errorf("update promotion status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrPromotionNotFound
	}

	return nil
}

// SoftDelete xóa promotion (soft delete)
//
// Note: Chỉ cho phép xóa nếu current_uses = 0
func (r *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	// Check current_uses
	var currentUses int
	err := r.db.QueryRow(ctx, "SELECT current_uses FROM promotions WHERE id = $1", id).Scan(&currentUses)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrPromotionNotFound
		}
		return fmt.Errorf("check current uses: %w", err)
	}

	if currentUses > 0 {
		return fmt.Errorf("Delete promotion failed") //  model.ErrPromotionCannotDelete
	}

	// Soft delete (set is_active = false, có thể thêm deleted_at column)
	query := `
		UPDATE promotions
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	_, err = r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft delete promotion: %w", err)
	}

	return nil
}

// -------------------------------------------------------------------
// USAGE TRACKING
// -------------------------------------------------------------------

// CreateUsage tạo promotion_usage record trong transaction
//
// Important Notes:
// - Phải gọi trong transaction (tx parameter)
// - Trigger `trigger_increment_promotion_usage` sẽ tự động increment current_uses
// - Unique constraint đảm bảo một order chỉ dùng một promotion
func (r *PostgresRepository) CreateUsage(ctx context.Context, tx pgx.Tx, usage *model.PromotionUsage) error {
	// Generate ID nếu chưa có
	if usage.ID == uuid.Nil {
		usage.ID = uuid.New()
	}

	query := `
		INSERT INTO promotion_usage (
			id, promotion_id, user_id, order_id,
			discount_amount, used_at, version
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), 0
		)
		RETURNING used_at
	`

	err := tx.QueryRow(ctx, query,
		usage.ID,
		usage.PromotionID,
		usage.UserID,
		usage.OrderID,
		usage.DiscountAmount,
	).Scan(&usage.UsedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate usage)
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("create promotion usage: %w", err) //model.ErrPromotionDuplicateUsage
			}
		}
		return fmt.Errorf("create promotion usage: %w", err)
	}

	usage.Version = 0

	return nil
}

// GetUsageHistory lấy lịch sử sử dụng promotion với chi tiết user & order
//
// Params:
// - promoID: ID của promotion
// - startDate, endDate: Filter theo thời gian (nil = không filter)
// - userID: Filter theo user (nil = tất cả users)
// - page, limit: Pagination
func (r *PostgresRepository) GetUsageHistory(
	ctx context.Context,
	promoID uuid.UUID,
	startDate, endDate *time.Time,
	userID *uuid.UUID,
	page, limit int,
) ([]*model.PromotionUsageWithDetails, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause
	whereClauses := []string{"pu.promotion_id = $1"}
	args := []interface{}{promoID}
	argIndex := 2

	if startDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("pu.used_at >= $%d", argIndex))
		args = append(args, *startDate)
		argIndex++
	}

	if endDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("pu.used_at <= $%d", argIndex))
		args = append(args, *endDate)
		argIndex++
	}

	if userID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("pu.user_id = $%d", argIndex))
		args = append(args, *userID)
		argIndex++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Query với JOIN để lấy thông tin user và order
	query := fmt.Sprintf(`
		SELECT 
			pu.id, pu.promotion_id, pu.user_id, pu.order_id,
			pu.discount_amount, pu.used_at, pu.version,
			u.email as user_email,
			u.full_name as user_full_name,
			o.order_number,
			o.total as order_total,
			o.status as order_status
		FROM promotion_usage pu
		INNER JOIN users u ON u.id = pu.user_id
		INNER JOIN orders o ON o.id = pu.order_id
		WHERE %s
		ORDER BY pu.used_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIndex, argIndex+1)

	args = append(args, limit, offset)

	// Execute query
	var usages []*model.PromotionUsageWithDetails
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("get usage history: %w", err)
	}

	for rows.Next() {
		var u model.PromotionUsageWithDetails
		err := rows.Scan(&u)
		if err != nil {
			return nil, 0, err
		}
		usages = append(usages, &u)
	}
	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM promotion_usage pu
		WHERE %s
	`, whereSQL)
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET

	var total int
	err = r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count usage history: %w", err)
	}

	return usages, total, nil
}

// GetUsageStats tính toán thống kê sử dụng promotion
//
// Returns:
// - total_uses: Tổng số lần sử dụng
// - total_discount_given: Tổng tiền đã giảm
// - average_discount_per_order: Trung bình giảm giá/đơn
// - unique_users: Số user đã sử dụng
func (r *PostgresRepository) GetUsageStats(
	ctx context.Context,
	promoID uuid.UUID,
	startDate, endDate *time.Time,
) (*model.UsageStats, error) {
	// Build WHERE clause
	whereClauses := []string{"promotion_id = $1"}
	args := []interface{}{promoID}
	argIndex := 2

	if startDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("used_at >= $%d", argIndex))
		args = append(args, *startDate)
		argIndex++
	}

	if endDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("used_at <= $%d", argIndex))
		args = append(args, *endDate)
		argIndex++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_uses,
			COALESCE(SUM(discount_amount), 0) as total_discount_given,
			COALESCE(AVG(discount_amount), 0) as average_discount_per_order,
			COUNT(DISTINCT user_id) as unique_users
		FROM promotion_usage
		WHERE %s
	`, whereSQL)

	var stats model.UsageStats
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalUses,
		&stats.TotalDiscountGiven,
		&stats.AverageDiscountPerOrder,
		&stats.UniqueUsers,
	)
	if err != nil {
		return nil, fmt.Errorf("get usage stats: %w", err)
	}

	// Revenue impact = negative (đã giảm)
	stats.RevenueImpact = stats.TotalDiscountGiven.Neg()

	return &stats, nil
}

// -------------------------------------------------------------------
// UTILITY
// -------------------------------------------------------------------

// CheckCodeExists kiểm tra code đã tồn tại chưa
//
// Params:
// - excludeID: Loại trừ ID này (dùng cho update)
func (r *PostgresRepository) CheckCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM promotions WHERE LOWER(code) = LOWER($1)"
	args := []interface{}{code}

	if excludeID != nil {
		query += " AND id != $2"
		args = append(args, *excludeID)
	}

	query += ")"

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check code exists: %w", err)
	}

	return exists, nil
}
