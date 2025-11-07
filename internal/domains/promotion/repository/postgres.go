package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"bookstore-backend/internal/domains/promotion/model"
	"bookstore-backend/internal/shared/utils"
)

type promotionRepo struct {
	db *pgxpool.Pool
}

// NewPromotionRepository creates a new instance of PromotionRepository
func NewPromotionRepository(db *pgxpool.Pool) PromotionRepository {
	return &promotionRepo{
		db: db,
	}
}

// Create implements PromotionRepository
func (r *promotionRepo) Create(ctx context.Context, promotion *model.PromotionEntity) (uuid.UUID, error) {
	query := `
		INSERT INTO promotions (
			code, name, description, discount_type, discount_value,
			max_discount_amount, min_order_amount, applicable_category_ids,
			first_order_only, max_uses, max_uses_per_user, current_uses,
			starts_at, expires_at, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 
			$14, $15
		) RETURNING id`
	var promotionID uuid.UUID
	err := r.db.QueryRow(ctx, query,
		promotion.Code, promotion.Name, promotion.Description,
		promotion.DiscountType, promotion.DiscountValue, promotion.MaxDiscountAmount,
		promotion.MinOrderAmount, promotion.ApplicableCategoryIDs, promotion.FirstOrderOnly,
		promotion.MaxUses, promotion.MaxUsesPerUser, 0,
		promotion.StartsAt, promotion.ExpiresAt, promotion.IsActive,
	).Scan(&promotionID)

	if err != nil {
		// Check for unique code violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return uuid.Nil, fmt.Errorf("promotion code %s already exists", promotion.Code)
		}
		return uuid.Nil, fmt.Errorf("failed to create promotion: %w", err)
	}

	return promotionID, nil
}

// Update implements PromotionRepository
func (r *promotionRepo) Update(ctx context.Context, p *model.PromotionEntity) error {
	p.UpdatedAt = time.Now()

	const sql = `
        UPDATE promotions 
        SET name = $1, description = $2, discount_value = $3, 
            max_discount_amount = $4, min_order_amount = $5,
            applicable_category_ids = $6, max_uses = $7, 
            max_uses_per_user = $8, expires_at = $9, 
            is_active = $10, updated_at = $11
        WHERE id = $12 AND deleted_at IS NULL
        RETURNING id`

	// Đây là dòng THẦN THÁNH – tự map struct → []any
	args := utils.StructArgs(p,
		"Name", "Description", "DiscountValue",
		"MaxDiscountAmount", "MinOrderAmount", "ApplicableCategoryIDs",
		"MaxUses", "MaxUsesPerUser", "ExpiresAt", "IsActive", "UpdatedAt", "ID",
	)

	var id string
	err := r.db.QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.ErrPromotionNotFound
		}
		return fmt.Errorf("update promotion failed: %w", err)
	}
	return nil
}

// GetByID implements PromotionRepository
func (r *promotionRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.PromotionEntity, error) {
	query := `
		SELECT * FROM promotions 
		WHERE id = $1 
		  AND is_active = true
		  AND starts_at <= NOW() 
		  AND expires_at > NOW()`

	var p model.PromotionEntity
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Code, &p.Name, &p.Description, &p.DiscountType, &p.DiscountValue,
		&p.MaxDiscountAmount, &p.MinOrderAmount, &p.ApplicableCategoryIDs,
		&p.FirstOrderOnly, &p.MaxUses, &p.MaxUsesPerUser, &p.CurrentUses,
		&p.StartsAt, &p.ExpiresAt, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("failed to get promotion: %w", err)
	}
	return &p, nil
}

func (r *promotionRepo) GetByCode(ctx context.Context, code string) (*model.PromotionEntity, error) {
	query := `
		SELECT * FROM promotions 
		WHERE code = $1 
		  AND is_active = true
		  AND starts_at <= NOW() 
		  AND expires_at > NOW()`

	var p model.PromotionEntity
	err := r.db.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.Code, &p.Name, &p.Description, &p.DiscountType, &p.DiscountValue,
		&p.MaxDiscountAmount, &p.MinOrderAmount, &p.ApplicableCategoryIDs,
		&p.FirstOrderOnly, &p.MaxUses, &p.MaxUsesPerUser, &p.CurrentUses,
		&p.StartsAt, &p.ExpiresAt, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("failed to get promotion by code: %w", err)
	}
	return &p, nil
}

// List implements PromotionRepository
func (r *promotionRepo) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*model.PromotionEntity, int64, error) {
	var where []string
	var args []interface{}
	argID := 1

	// Base: chỉ lấy promotion đang hoạt động
	baseWhere := "is_active = true AND starts_at <= NOW() AND expires_at > NOW()"
	query := "SELECT * FROM promotions WHERE " + baseWhere
	countQuery := "SELECT COUNT(*) FROM promotions WHERE " + baseWhere

	if filters != nil {
		if name, ok := filters["name"].(string); ok && name != "" {
			where = append(where, fmt.Sprintf("name ILIKE $%d", argID))
			args = append(args, "%"+name+"%")
			argID++
		}
		if code, ok := filters["code"].(string); ok && code != "" {
			where = append(where, fmt.Sprintf("code ILIKE $%d", argID))
			args = append(args, "%"+code+"%")
			argID++
		}
	}

	if len(where) > 0 {
		clause := strings.Join(where, " AND ")
		query += " AND " + clause
		countQuery += " AND " + clause
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argID, argID+1)
	args = append(args, pageSize, (page-1)*pageSize)

	// Count
	var total int64
	countArgs := append([]interface{}{}, args[:len(args)-2]...)
	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count promotions: %w", err)
	}

	// List
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list promotions: %w", err)
	}
	defer rows.Close()

	var promotions []*model.PromotionEntity
	for rows.Next() {
		var p model.PromotionEntity
		err := rows.Scan(
			&p.ID, &p.Code, &p.Name, &p.Description, &p.DiscountType, &p.DiscountValue,
			&p.MaxDiscountAmount, &p.MinOrderAmount, &p.ApplicableCategoryIDs,
			&p.FirstOrderOnly, &p.MaxUses, &p.MaxUsesPerUser, &p.CurrentUses,
			&p.StartsAt, &p.ExpiresAt, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan promotion failed: %w", err)
		}
		promotions = append(promotions, &p)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return promotions, total, nil
}

func (r *promotionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE promotions 
		SET is_active = false, updated_at = $1 
		WHERE id = $2 AND is_active = true`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to deactivate promotion: %w", err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrPromotionNotFound
	}
	return nil
}

// === GET ACTIVE PROMOTIONS ===
func (r *promotionRepo) GetActivePromotions(ctx context.Context) ([]*model.PromotionEntity, error) {
	query := `
		SELECT * FROM promotions
		WHERE is_active = true
		  AND starts_at <= NOW()
		  AND expires_at > NOW()
		  AND (max_uses IS NULL OR current_uses < max_uses)`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active promotions: %w", err)
	}
	defer rows.Close()

	var promotions []*model.PromotionEntity
	for rows.Next() {
		var p model.PromotionEntity
		err := rows.Scan(
			&p.ID, &p.Code, &p.Name, &p.Description, &p.DiscountType, &p.DiscountValue,
			&p.MaxDiscountAmount, &p.MinOrderAmount, &p.ApplicableCategoryIDs,
			&p.FirstOrderOnly, &p.MaxUses, &p.MaxUsesPerUser, &p.CurrentUses,
			&p.StartsAt, &p.ExpiresAt, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		promotions = append(promotions, &p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return promotions, nil
}

// === INCREMENT USAGE (vẫn giữ transaction + RETURNING) ===
func (r *promotionRepo) IncrementUsage(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE promotions 
		SET current_uses = current_uses + 1, updated_at = $1
		WHERE id = $2 
		  AND is_active = true
		  AND starts_at <= NOW()
		  AND expires_at > NOW()
		  AND (max_uses IS NULL OR current_uses < max_uses)
		RETURNING current_uses`

	var currentUses int
	err = tx.QueryRow(ctx, query, time.Now(), id).Scan(&currentUses)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.ErrPromotionNotFound // hoặc ErrPromotionExhausted nếu muốn phân biệt
		}
		return fmt.Errorf("increment usage failed: %w", err)
	}

	return tx.Commit(ctx)
}

// GetUserPromotionUsage implements PromotionRepository
func (r *promotionRepo) GetUserPromotionUsage(ctx context.Context, promotionID, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM promotion_usages
		WHERE promotion_id = $1 AND user_id = $2`

	var count int
	err := r.db.QueryRow(ctx, query, promotionID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user promotion usage: %w", err)
	}

	return count, nil
}

// RecordPromotionUsage implements PromotionRepository
// === RECORD USAGE ===
func (r *promotionRepo) RecordPromotionUsage(ctx context.Context, usage *model.PromotionUsageEntity) error {
	query := `
		INSERT INTO promotion_usages (id, promotion_id, user_id, order_id, discount_amount, used_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	usage.ID = uuid.New()
	usage.UsedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		usage.ID, usage.PromotionID, usage.UserID, usage.OrderID, usage.DiscountAmount, usage.UsedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to record promotion usage: %w", err)
	}
	return nil
}

// GetPromotionUsageHistory implements PromotionRepository
func (r *promotionRepo) GetPromotionUsageHistory(ctx context.Context, promotionID uuid.UUID, page, pageSize int) ([]*model.PromotionUsageEntity, int64, error) {
	query := `
		SELECT * FROM promotion_usages
		WHERE promotion_id = $1
		ORDER BY used_at DESC
		LIMIT $2 OFFSET $3`

	usages := []*model.PromotionUsageEntity{}
	rows, err := r.db.Query(ctx, query, promotionID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get promotion usage history: %w", err)
	}

	for rows.Next() {
		var u model.PromotionUsageEntity
		err := rows.Scan(&u)
		if err != nil {
			return nil, 0, err
		}
		usages = append(usages, &u)
	}

	// Get total count
	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM promotion_usages
		WHERE promotion_id = $1`

	err = r.db.QueryRow(ctx, countQuery, promotionID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count promotion usages: %w", err)
	}

	return usages, total, nil
}

// exists checks if a promotion exists
func (r *promotionRepo) exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM promotions
			WHERE id = $1 AND deleted_at IS NULL
		)`

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check promotion existence: %w", err)
	}

	return exists, nil
}
