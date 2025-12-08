package repository

import (
	"context"
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

// =================================================================================================
// UTILS & HELPERS (Hàm chung để tái sử dụng)
// =================================================================================================

// scanPromotionRow quét dữ liệu từ một hàng vào struct Promotion
// interface scannable giúp hàm này nhận được cả pgx.Row và pgx.Rows
type scannable interface {
	Scan(dest ...interface{}) error
}

func scanPromotionRow(row scannable) (*model.Promotion, error) {
	var p model.Promotion
	err := row.Scan(
		&p.ID, &p.Code, &p.Name, &p.Description,
		&p.DiscountType, &p.DiscountValue, &p.MaxDiscountAmount,
		&p.MinOrderAmount, &p.ApplicableCategoryIDs, &p.FirstOrderOnly,
		&p.MaxUses, &p.MaxUsesPerUser, &p.CurrentUses,
		&p.StartsAt, &p.ExpiresAt, &p.IsActive, &p.Version,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return &p, err
}

// standardSelectClause trả về danh sách cột chuẩn để tránh viết lại nhiều lần
const standardSelectClause = `
	SELECT
		id, code, name, description,
		discount_type, discount_value, max_discount_amount,
		min_order_amount, applicable_category_ids, first_order_only,
		max_uses, max_uses_per_user, current_uses,
		starts_at, expires_at, is_active, version,
		created_at, updated_at
	FROM promotions
`

// =================================================================================================
// READ OPERATIONS
// =================================================================================================

// FindByID tìm promotion theo ID
func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Promotion, error) {
	query := standardSelectClause + ` WHERE id = $1`

	p, err := scanPromotionRow(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("find promotion by id: %w", err)
	}
	return p, nil
}

// FindByCode tìm promotion theo code (không filter active/time)
func (r *PostgresRepository) FindByCode(ctx context.Context, code string) (*model.Promotion, error) {
	query := standardSelectClause + ` WHERE LOWER(code) = LOWER($1)`

	p, err := scanPromotionRow(r.db.QueryRow(ctx, query, code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("find promotion by code: %w", err)
	}
	return p, nil
}

// FindByCodeActive tìm promotion active theo code
func (r *PostgresRepository) FindByCodeActive(ctx context.Context, code string) (*model.Promotion, error) {
	// Ghi đè logic select một chút để xử lý COALESCE cho max_uses_per_user nếu cần,
	// nhưng để nhất quán với scanPromotionRow, ta nên giữ nguyên thứ tự cột.
	// Ở đây tôi dùng query cụ thể nhưng vẫn dùng hàm scan chung.
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

	p, err := scanPromotionRow(r.db.QueryRow(ctx, query, code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("find promotion by code: %w", err)
	}
	return p, nil
}

// GetUserUsageCount đếm số lần user đã sử dụng promotion
func (r *PostgresRepository) GetUserUsageCount(ctx context.Context, promoID, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM promotion_usage WHERE promotion_id = $1 AND user_id = $2`

	var count int
	if err := r.db.QueryRow(ctx, query, promoID, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get user usage count: %w", err)
	}
	return count, nil
}

// ListActive lấy danh sách promotion active
func (r *PostgresRepository) ListActive(ctx context.Context, categoryID *uuid.UUID, page, limit int) ([]*model.Promotion, int, error) {
	offset := (page - 1) * limit

	baseWhere := ` WHERE is_active = true AND starts_at <= NOW() AND expires_at >= NOW()`
	args := []interface{}{}
	argIndex := 1

	// 1. Build Filter
	if categoryID != nil {
		baseWhere += fmt.Sprintf(" AND $%d = ANY(applicable_category_ids)", argIndex)
		args = append(args, *categoryID)
		argIndex++
	}

	// 2. Build Data Query
	// Lưu ý: Logic SELECT ở đây có xử lý COALESCE cho max_uses_per_user, khớp với struct scan
	query := `
		SELECT
			id, code, name, description,
			discount_type, discount_value, max_discount_amount,
			min_order_amount, applicable_category_ids, first_order_only,
			max_uses, COALESCE(max_uses_per_user, 0), current_uses,
			starts_at, expires_at, is_active, version,
			created_at, updated_at
		FROM promotions` + baseWhere + ` ORDER BY starts_at DESC` + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	queryArgs := append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		logger.Error("list active error", err)
		return nil, 0, fmt.Errorf("list active promotions: %w", err)
	}
	defer rows.Close()

	var promotions []*model.Promotion
	for rows.Next() {
		p, err := scanPromotionRow(rows)
		if err != nil {
			return nil, 0, err
		}
		promotions = append(promotions, p)
	}

	// 3. Count Total
	countQuery := `SELECT COUNT(*) FROM promotions` + baseWhere
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count active promotions: %w", err)
	}

	return promotions, total, nil
}

// ListAdmin lấy danh sách promotion với filter (Admin API)
func (r *PostgresRepository) ListAdmin(ctx context.Context, filter *model.ListPromotionsFilter) ([]*model.PromotionListItem, int, error) {
	offset := (filter.Page - 1) * filter.Limit
	whereClauses := []string{"1=1"} // Dummy condition để dễ append AND
	args := []interface{}{}
	argIndex := 1

	// Build Conditions
	switch filter.Status {
	case "active":
		whereClauses = append(whereClauses, "is_active = true AND NOW() BETWEEN starts_at AND expires_at")
	case "expired":
		whereClauses = append(whereClauses, "NOW() > expires_at")
	case "upcoming":
		whereClauses = append(whereClauses, "NOW() < starts_at")
	}

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(code) LIKE $%d OR LOWER(name) LIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argIndex++
	}

	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	// Build Sort
	orderBySQL := "ORDER BY created_at DESC"
	switch filter.Sort {
	case "expires_at_asc":
		orderBySQL = "ORDER BY expires_at ASC"
	case "usage_desc":
		orderBySQL = "ORDER BY current_uses DESC"
	case "name_asc":
		orderBySQL = "ORDER BY name ASC"
	}

	// Main Query
	query := fmt.Sprintf(`
		SELECT
			id, code, name,
			discount_type, discount_value, max_discount_amount,
			current_uses, max_uses,
			CASE
				WHEN max_uses IS NOT NULL AND max_uses > 0 THEN (current_uses::FLOAT / max_uses * 100)
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
		%s %s LIMIT $%d OFFSET $%d
	`, whereSQL, orderBySQL, argIndex, argIndex+1)

	queryArgs := append(args, filter.Limit, offset)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin promotions: %w", err)
	}
	defer rows.Close()

	var items []*model.PromotionListItem
	for rows.Next() {
		var i model.PromotionListItem
		// Scan trực tiếp vì struct này khác Promotion gốc
		if err := rows.Scan(
			&i.ID, &i.Code, &i.Name,
			&i.DiscountType, &i.DiscountValue, &i.MaxDiscountAmount,
			&i.CurrentUses, &i.MaxUses,
			&i.UsageRate,
			&i.StartsAt, &i.ExpiresAt, &i.IsActive, &i.Status,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, &i)
	}

	// Count Total
	var total int
	if err := r.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM promotions %s", whereSQL), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin promotions: %w", err)
	}

	return items, total, nil
}

// =================================================================================================
// WRITE OPERATIONS
// =================================================================================================

// Create tạo promotion mới
func (r *PostgresRepository) Create(ctx context.Context, promo *model.Promotion) error {
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
			$11, 0, $12, $13, $14, NOW(), NOW()
		)
		RETURNING id, code, name
	`
	// current_uses = 0 ($11 là max_uses_per_user, giá trị cứng 0 ở giữa, $12 là starts_at...) - SAI INDEX
	// Sửa lại index cho chuẩn:
	err := r.db.QueryRow(ctx, query,
		promo.Code, promo.Name, promo.Description,
		promo.DiscountType, promo.DiscountValue, promo.MaxDiscountAmount,
		promo.MinOrderAmount, pq.Array(promo.ApplicableCategoryIDs), promo.FirstOrderOnly,
		promo.MaxUses, promo.MaxUsesPerUser, // $10, $11
		promo.StartsAt, promo.ExpiresAt, promo.IsActive, // $12, $13, $14
	).Scan(&promo.ID, &promo.Code, &promo.Name)

	if err != nil {
		logger.Error("error create::", err)
		if isUniqueViolation(err) {
			return fmt.Errorf("constraint violation") // model.ErrPromotionCodeExists
		}
		return fmt.Errorf("create promotion: %w", err)
	}

	promo.CurrentUses = 0
	promo.Version = 0
	return nil
}

// Update cập nhật promotion với optimistic locking
func (r *PostgresRepository) Update(ctx context.Context, promo *model.Promotion) error {
	promo.Code = strings.ToUpper(promo.Code)

	query := `
		UPDATE promotions
		SET
			code = $2, name = $3, description = $4,
			discount_type = $5, discount_value = $6, max_discount_amount = $7,
			min_order_amount = $8, applicable_category_ids = $9, first_order_only = $10,
			max_uses = $11, max_uses_per_user = $12,
			starts_at = $13, expires_at = $14, is_active = $15,
			version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $16
		RETURNING id, name, code
	`

	err := r.db.QueryRow(ctx, query,
		promo.ID, promo.Code, promo.Name, promo.Description,
		promo.DiscountType, promo.DiscountValue, promo.MaxDiscountAmount,
		promo.MinOrderAmount, pq.Array(promo.ApplicableCategoryIDs), promo.FirstOrderOnly,
		promo.MaxUses, promo.MaxUsesPerUser,
		promo.StartsAt, promo.ExpiresAt, promo.IsActive,
		promo.Version,
	).Scan(&promo.ID, &promo.Name, &promo.Code)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("version mismatch or promotion not exist")
		}
		return fmt.Errorf("update promotion: %w", err)
	}
	return nil
}

// UpdateStatus cập nhật trạng thái active/inactive
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `UPDATE promotions SET is_active = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, isActive)
	if err != nil {
		return fmt.Errorf("update promotion status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrPromotionNotFound
	}
	return nil
}

// SoftDelete xóa promotion
func (r *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	// Gom check và update vào 1 logic nếu có thể, hoặc giữ logic check tường minh
	// Ở đây logic check currentUses > 0 là quan trọng nên giữ nguyên 2 bước
	var currentUses int
	if err := r.db.QueryRow(ctx, "SELECT current_uses FROM promotions WHERE id = $1", id).Scan(&currentUses); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrPromotionNotFound
		}
		return fmt.Errorf("check current uses: %w", err)
	}

	if currentUses > 0 {
		return fmt.Errorf("delete promotion failed: in use")
	}

	_, err := r.db.Exec(ctx, "UPDATE promotions SET is_active = false, updated_at = NOW() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("soft delete promotion: %w", err)
	}
	return nil
}

// =================================================================================================
// USAGE TRACKING
// =================================================================================================

// CreateUsage tạo promotion_usage
func (r *PostgresRepository) CreateUsage(ctx context.Context, tx pgx.Tx, usage *model.PromotionUsage) error {
	if usage.ID == uuid.Nil {
		usage.ID = uuid.New()
	}

	query := `
		INSERT INTO promotion_usage (id, promotion_id, user_id, order_id, discount_amount, used_at, version)
		VALUES ($1, $2, $3, $4, $5, NOW(), 0)
		RETURNING used_at
	`
	err := tx.QueryRow(ctx, query, usage.ID, usage.PromotionID, usage.UserID, usage.OrderID, usage.DiscountAmount).Scan(&usage.UsedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("create promotion usage: duplicate usage")
		}
		return fmt.Errorf("create promotion usage: %w", err)
	}

	usage.Version = 0
	return nil
}

// GetUsageHistory lấy lịch sử sử dụng
func (r *PostgresRepository) GetUsageHistory(ctx context.Context, promoID uuid.UUID, startDate, endDate *time.Time, userID *uuid.UUID, page, limit int) ([]*model.PromotionUsageWithDetails, int, error) {
	offset := (page - 1) * limit
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

	query := fmt.Sprintf(`
		SELECT
			pu.id, pu.promotion_id, pu.user_id, pu.order_id,
			pu.discount_amount, pu.used_at, pu.version,
			u.email, u.full_name, o.order_number, o.total, o.status
		FROM promotion_usage pu
		INNER JOIN users u ON u.id = pu.user_id
		INNER JOIN orders o ON o.id = pu.order_id
		WHERE %s
		ORDER BY pu.used_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIndex, argIndex+1)

	queryArgs := append(args, limit, offset)
	logger.Info("Check data ", map[string]interface{}{
		"query":    query,
		"args":     queryArgs,
		"whereSQL": whereSQL,
	})
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("get usage history: %w", err)
	}
	defer rows.Close()

	var usages []*model.PromotionUsageWithDetails
	for rows.Next() {
		var u model.PromotionUsageWithDetails
		if err := rows.Scan(
			&u.ID, &u.PromotionID, &u.UserID, &u.OrderID,
			&u.DiscountAmount, &u.UsedAt, &u.Version,
			&u.UserEmail, &u.UserFullName, &u.OrderNumber, &u.OrderTotal, &u.OrderStatus,
		); err != nil {
			return nil, 0, err
		}
		usages = append(usages, &u)
	}

	var total int
	if err := r.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM promotion_usage pu WHERE %s", whereSQL), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count usage history: %w", err)
	}

	return usages, total, nil
}

// GetUsageStats tính toán thống kê
func (r *PostgresRepository) GetUsageStats(ctx context.Context, promoID uuid.UUID, startDate, endDate *time.Time) (*model.UsageStats, error) {
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

	query := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(discount_amount), 0),
			COALESCE(AVG(discount_amount), 0),
			COUNT(DISTINCT user_id)
		FROM promotion_usage
		WHERE %s
	`, strings.Join(whereClauses, " AND "))

	var stats model.UsageStats
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&stats.TotalUses,
		&stats.TotalDiscountGiven,
		&stats.AverageDiscountPerOrder,
		&stats.UniqueUsers,
	); err != nil {
		return nil, fmt.Errorf("get usage stats: %w", err)
	}

	stats.RevenueImpact = stats.TotalDiscountGiven.Neg()
	return &stats, nil
}

// CheckCodeExists kiểm tra code tồn tại
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
	return exists, err
}

// isUniqueViolation helper function để check lỗi unique constraints
func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}
