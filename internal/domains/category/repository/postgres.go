package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bookstore-backend/internal/domains/category"
	"bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewpostgresRepository tạo repository instance
func NewPostgresRepository(pool *pgxpool.Pool, cache cache.Cache) category.CategoryRepository {
	return &postgresRepository{
		pool:  pool,
		cache: cache,
	}
}
func (r *postgresRepository) Create(
	ctx context.Context,
	entity *category.Category,
) (*category.Category, error) {
	const query = `
		INSERT INTO categories (
			id, name, slug, parent_id, sort_order, 
			description, icon_url, is_active, 
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING 
			id, name, slug, parent_id, sort_order, 
			description, icon_url, is_active, 
			created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		entity.ID,
		entity.Name,
		entity.Slug,
		entity.ParentID,
		entity.SortOrder,
		entity.Description,
		entity.IconURL,
		entity.IsActive,
		entity.CreatedAt,
		entity.UpdatedAt,
	)

	created := &category.Category{}
	err := row.Scan(
		&created.ID,
		&created.Name,
		&created.Slug,
		&created.ParentID,
		&created.SortOrder,
		&created.Description,
		&created.IconURL,
		&created.IsActive,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "idx_categories_slug" {
				logger.Error("Create: duplicate slug", err)
				return nil, category.ErrDuplicateSlug
			}
			if pgErr.ConstraintName == "categories_parent_id_fkey" {
				logger.Error("Create: parent not found", err)
				return nil, category.ErrParentNotFound
			}
		}
		logger.Error("Create: database error", err)
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	// ========== CRITICAL FIX: PRESERVE LEVEL FROM SERVICE ==========
	// Service đã tính level (check max depth)
	// Repository PHẢI preserve nó vào created entity
	// KHÔNG để nó bị mất
	if entity.Level != nil {
		created.Level = entity.Level
	} else {
		// Fallback: calculate (không nên xảy ra)
		level := 1
		if created.ParentID != nil {
			calcLevel, err := r.calculateLevelFromParent(ctx, *created.ParentID)
			if err == nil {
				level = calcLevel
			}
		}
		created.Level = &level
	}

	return created, nil
}

// ========== Helper: Calculate Level from Parent ==========
// Tính level từ parent_id bằng recursive query
func (r *postgresRepository) calculateLevelFromParent(ctx context.Context, parentID uuid.UUID) (int, error) {
	const query = `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 1 as level
			FROM categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.level + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
		)
		SELECT MAX(level) FROM parent_chain
	`

	var maxLevel int
	err := r.pool.QueryRow(ctx, query, parentID).Scan(&maxLevel)
	if err != nil {
		return 0, err
	}

	// Parent's level + 1
	return maxLevel + 1, nil
}

// ============================================================
// READ: GetByID --------------- GetByID fetches category by ID
// ============================================================
func (r *postgresRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*category.Category, error) {
	const query = `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 0 as depth
			FROM categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.depth + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
			WHERE pc.parent_id IS NOT NULL
		)
		SELECT 
			c.id, c.name, c.slug, c.parent_id, c.sort_order, 
			c.description, c.icon_url, c.is_active, 
			c.created_at, c.updated_at,
			(SELECT MAX(depth) FROM parent_chain) + 1 as level,
			COALESCE((
				SELECT COUNT(*) 
				FROM categories 
				WHERE parent_id = c.id AND is_active = true
			), 0) as children_count
		FROM categories c
		WHERE c.id = $1
	`

	entity := &category.Category{}
	var level int
	var childrenCount int

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Slug,
		&entity.ParentID,
		&entity.SortOrder,
		&entity.Description,
		&entity.IconURL,
		&entity.IsActive,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&level,
		&childrenCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, category.ErrCategoryNotFound
		}
		logger.Error("GetByID: database error", err)
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	entity.Level = &level
	entity.ChildCount = &childrenCount

	return entity, nil
}

// ============================================================
// READ: GetBySlug --------- GetBySlug fetches category by slug
// ============================================================
func (r *postgresRepository) GetBySlug(
	ctx context.Context,
	slug string,
) (*category.Category, error) {
	const query = `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 0 as depth
			FROM categories
			WHERE slug = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.depth + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
			WHERE pc.parent_id IS NOT NULL
		)
		SELECT 
			c.id, c.name, c.slug, c.parent_id, c.sort_order, 
			c.description, c.icon_url, c.is_active, 
			c.created_at, c.updated_at,
			(SELECT MAX(depth) FROM parent_chain) + 1 as level,
			COALESCE((
				SELECT COUNT(*) 
				FROM categories 
				WHERE parent_id = c.id AND is_active = true
			), 0) as children_count
		FROM categories c
		WHERE c.slug = $1
	`

	entity := &category.Category{}
	var level int
	var childrenCount int

	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Slug,
		&entity.ParentID,
		&entity.SortOrder,
		&entity.Description,
		&entity.IconURL,
		&entity.IsActive,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&level,
		&childrenCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, category.ErrCategoryNotFound
		}
		logger.Error("GetBySlug: database error", err)
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	entity.Level = &level
	entity.ChildCount = &childrenCount

	return entity, nil
}

// ============================================================
// GetAll - fetches categories with filters and pagination
// ============================================================
// FLOW:
// 1. Build WHERE clause dynamically based on filter
// 2. Count total (without LIMIT/OFFSET)
// 3. Query with LIMIT/OFFSET
// 4. Return both list + total
func (r *postgresRepository) GetAll(
	ctx context.Context,
	filter *category.CategoryFilter,
) ([]category.Category, int64, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("c.is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	} else if !filter.IncludeInactive {
		whereClauses = append(whereClauses, "c.is_active = true")
	}

	if filter.ParentID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("c.parent_id = $%d", argIndex))
		args = append(args, filter.ParentID)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM categories c
		%s
	`, whereClause)

	var total int64
	countArgs := args
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		logger.Error("GetAll: count query failed", err)
		return nil, 0, fmt.Errorf("failed to count categories: %w", err)
	}

	// ========== LIST QUERY WITH CHILDREN_COUNT ==========
	listQuery := fmt.Sprintf(`
		WITH category_levels AS (
			SELECT 
				c.id,
				c.name,
				c.slug,
				c.parent_id,
				c.sort_order,
				c.description,
				c.icon_url,
				c.is_active,
				c.created_at,
				c.updated_at,
				(
					WITH RECURSIVE parent_chain AS (
						SELECT id, parent_id, 1 as level
						FROM categories
						WHERE id = c.id

						UNION ALL

						SELECT cat.id, cat.parent_id, pc.level + 1
						FROM categories cat
						INNER JOIN parent_chain pc ON cat.id = pc.parent_id
					)
					SELECT MAX(level) FROM parent_chain
				) as level,
				COALESCE((
					SELECT COUNT(*)
					FROM categories
					WHERE parent_id = c.id AND is_active = true
				), 0) as children_count
			FROM categories c
			%s
		)
		SELECT * FROM category_levels
		ORDER BY sort_order ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	listArgs := append(args, filter.Limit, filter.Offset)
	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		logger.Error("GetAll: query failed", err)
		return nil, 0, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	entities := make([]category.Category, 0, filter.Limit)
	for rows.Next() {
		entity := category.Category{}
		var level int
		var childrenCount int

		err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Slug,
			&entity.ParentID,
			&entity.SortOrder,
			&entity.Description,
			&entity.IconURL,
			&entity.IsActive,
			&entity.CreatedAt,
			&entity.UpdatedAt,
			&level,
			&childrenCount,
		)

		if err != nil {
			logger.Error("GetAll: scan error", err)
			return nil, 0, fmt.Errorf("failed to scan category: %w", err)
		}

		entity.Level = &level
		entity.ChildCount = &childrenCount
		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		logger.Error("GetAll: rows error", err)
		return nil, 0, fmt.Errorf("failed to get categories: %w", err)
	}

	return entities, total, nil
}

// ============================================================
// READ: GetTree (FIXED - CRITICAL)
// ============================================================
// GetTree lấy toàn bộ category tree với level calculation
// KHÔNG DÙNG Materialized View (vì không auto-refresh)
// Dùng Recursive CTE trực tiếp
// File: internal/domains/category/repository/category_repository.go

func (r *postgresRepository) GetTree(ctx context.Context) ([]category.Category, error) {
	// ========== RECURSIVE CTE WITH CHILDREN_COUNT ==========
	const query = `
		WITH RECURSIVE tree AS (
			-- Base case: Root categories
			SELECT 
				id,
				name,
				slug,
				parent_id,
				sort_order,
				description,
				icon_url,
				is_active,
				created_at,
				updated_at,
				1 as level,
				ARRAY[sort_order] as path,
				name::TEXT as full_path
			FROM categories
			WHERE parent_id IS NULL AND is_active = true
			
			UNION ALL
			
			-- Recursive case: Child categories
			SELECT 
				c.id,
				c.name,
				c.slug,
				c.parent_id,
				c.sort_order,
				c.description,
				c.icon_url,
				c.is_active,
				c.created_at,
				c.updated_at,
				t.level + 1,
				t.path || c.sort_order,
				t.full_path || ' > ' || c.name
			FROM categories c
			INNER JOIN tree t ON c.parent_id = t.id
			WHERE c.is_active = true
		)
		-- ========== COUNT CHILDREN FOR EACH CATEGORY ==========
		-- Join với subquery để count direct children
		SELECT 
			t.id, 
			t.name, 
			t.slug, 
			t.parent_id, 
			t.sort_order, 
			t.description, 
			t.icon_url, 
			t.is_active, 
			t.created_at, 
			t.updated_at, 
			t.level, 
			t.full_path,
			COALESCE(child_count.count, 0) as children_count
		FROM tree t
		LEFT JOIN (
			-- Subquery: Count direct children for each category
			SELECT parent_id, COUNT(*) as count
			FROM categories
			WHERE is_active = true AND parent_id IS NOT NULL
			GROUP BY parent_id
		) child_count ON t.id = child_count.parent_id
		ORDER BY t.path ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		logger.Error("GetTree: query failed", err)
		return nil, fmt.Errorf("failed to get category tree: %w", err)
	}
	defer rows.Close()

	entities := make([]category.Category, 0)
	for rows.Next() {
		entity := category.Category{}
		var level int
		var fullPath string
		var childrenCount int

		err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Slug,
			&entity.ParentID,
			&entity.SortOrder,
			&entity.Description,
			&entity.IconURL,
			&entity.IsActive,
			&entity.CreatedAt,
			&entity.UpdatedAt,
			&level,
			&fullPath,
			&childrenCount,
		)

		if err != nil {
			logger.Error("GetTree: scan error", err)
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		entity.Level = &level
		entity.FullPath = &fullPath
		entity.ChildCount = &childrenCount

		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		logger.Error("GetTree: rows error", err)
		return nil, fmt.Errorf("failed to get category tree: %w", err)
	}

	return entities, nil
}

// ============================================================
// READ: GetChildren
// ============================================================
// GetChildren lấy direct children của category
//
// DATABASE:
// SELECT * FROM categories
// WHERE parent_id = $1 AND is_active = true
// ORDER BY sort_order ASC
//
// INDEX: (parent_id, sort_order) composite index
// => Efficient query + sort
func (r *postgresRepository) GetChildren(
	ctx context.Context,
	parentID uuid.UUID,
) ([]category.Category, error) {
	const query = `
		SELECT id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
		FROM categories
		WHERE parent_id = $1 AND is_active = true
		ORDER BY sort_order ASC
	`

	rows, err := r.pool.Query(ctx, query, parentID)
	if err != nil {
		fmt.Println("GetChildren: query failed", err)
		return nil, fmt.Errorf("failed to get children: %w", err)
	}
	defer rows.Close()

	entities := make([]category.Category, 0)
	for rows.Next() {
		entity := category.Category{}
		err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Slug,
			&entity.ParentID,
			&entity.SortOrder,
			&entity.Description,
			&entity.IconURL,
			&entity.IsActive,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		)

		if err != nil {
			fmt.Println("GetChildren: scan error", err)
			return nil, fmt.Errorf("failed to scan child: %w", err)
		}

		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("GetChildren: rows error", err)
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	return entities, nil
}

// ============================================================
// READ: GetDescendants
// ============================================================
// GetDescendants lấy tất cả descendants (recursive)
//
// DATABASE (Recursive CTE):
// WITH RECURSIVE descendants AS (
//
//	SELECT * FROM categories WHERE id = $1
//	UNION ALL
//	SELECT c.* FROM categories c
//	INNER JOIN descendants d ON c.parent_id = d.id
//
// )
// SELECT * FROM descendants
//
// RECURSIVE CTE EXPLANATION:
// 1. Base case: SELECT category with id = $1
// 2. Recursive case: SELECT con của base case
// 3. UNION ALL: Combine kết quả
// 4. Repeat until no more rows
//
// EXAMPLE:
// Tree: A > B > C
// GetDescendants(A):
//
//	Iteration 1: [A] (base)
//	Iteration 2: [B] (child of A)
//	Iteration 3: [C] (child of B)
//	Iteration 4: [] (no more)
//	RESULT: [A, B, C]
//
// PERFORMANCE:
// - Recursive CTE: ~100ms
// - But limited by depth (max 3 levels)
// - So at most 3 iterations
func (r *postgresRepository) GetDescendants(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]category.Category, error) {
	const query = `
		WITH RECURSIVE descendants AS (
			SELECT id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
			FROM categories
			WHERE id = $1
			
			UNION ALL
			
			SELECT c.id, c.name, c.slug, c.parent_id, c.sort_order, c.description, c.icon_url, c.is_active, c.created_at, c.updated_at
			FROM categories c
			INNER JOIN descendants d ON c.parent_id = d.id
		)
		SELECT * FROM descendants
	`

	rows, err := r.pool.Query(ctx, query, categoryID)
	if err != nil {
		fmt.Println("GetDescendants: query failed", err)
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}
	defer rows.Close()

	entities := make([]category.Category, 0)
	for rows.Next() {
		entity := category.Category{}
		err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Slug,
			&entity.ParentID,
			&entity.SortOrder,
			&entity.Description,
			&entity.IconURL,
			&entity.IsActive,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		)

		if err != nil {
			fmt.Println("GetDescendants: scan error", err)
			return nil, fmt.Errorf("failed to scan descendant: %w", err)
		}

		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("GetDescendants: rows error", err)
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}

	return entities, nil
}

// ============================================================
// READ: GetAncestors
// ============================================================
// GetAncestors lấy tất cả ancestors (từ dưới lên trên)
//
// DATABASE (Recursive CTE từ dưới lên):
// WITH RECURSIVE ancestors AS (
//
//	SELECT * FROM categories WHERE id = $1
//	UNION ALL
//	SELECT c.* FROM categories c
//	INNER JOIN ancestors a ON c.id = a.parent_id
//
// )
// SELECT * FROM ancestors ORDER BY level ASC
//
// DIRECTION:
// GetDescendants: Từ trên xuống (parent => children)
// GetAncestors: Từ dưới lên (child => parents)
//
// EXAMPLE:
// Tree: A > B > C
// GetAncestors(C):
//
//	Iteration 1: [C] (base)
//	Iteration 2: [B] (parent of C)
//	Iteration 3: [A] (parent of B)
//	Iteration 4: [] (no more parents)
//	RESULT: [C, B, A] => ORDER BY level => [A, B, C]
//
// USE CASE:
// - Build breadcrumb
// - Validate circular reference
// - Check if newParent is descendant
func (r *postgresRepository) GetAncestors(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]category.Category, error) {
	const query = `
		WITH RECURSIVE ancestors AS (
			SELECT id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
			FROM categories
			WHERE id = $1
			
			UNION ALL
			
			SELECT c.id, c.name, c.slug, c.parent_id, c.sort_order, c.description, c.icon_url, c.is_active, c.created_at, c.updated_at
			FROM categories c
			INNER JOIN ancestors a ON c.id = a.parent_id
		)
		SELECT * FROM ancestors
	`

	rows, err := r.pool.Query(ctx, query, categoryID)
	if err != nil {
		fmt.Println("GetAncestors: query failed", err)
		return nil, fmt.Errorf("failed to get ancestors: %w", err)
	}
	defer rows.Close()

	entities := make([]category.Category, 0)
	for rows.Next() {
		entity := category.Category{}
		err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Slug,
			&entity.ParentID,
			&entity.SortOrder,
			&entity.Description,
			&entity.IconURL,
			&entity.IsActive,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		)

		if err != nil {
			fmt.Println("GetAncestors: scan error", err)
			return nil, fmt.Errorf("failed to scan ancestor: %w", err)
		}

		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("GetAncestors: rows error", err)
		return nil, fmt.Errorf("failed to get ancestors: %w", err)
	}

	return entities, nil
}

// ============================================================
// READ: GetCategoryBreadcrumb
// ============================================================
// GetCategoryBreadcrumb là wrapper của GetAncestors
func (r *postgresRepository) GetCategoryBreadcrumb(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]category.Category, error) {
	return r.GetAncestors(ctx, categoryID)
}

// ============================================================
// UPDATE: Update
// ============================================================
// Update cập nhật category
//
// DATABASE:
// UPDATE categories
// SET name = $1, slug = $2, ...
// WHERE id = $n
// RETURNING *
func (r *postgresRepository) Update(
	ctx context.Context,
	entity *category.Category,
) (*category.Category, error) {
	const query = `
		UPDATE categories
		SET name = $1, slug = $2, description = $3, icon_url = $4, sort_order = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		entity.Name,
		entity.Slug,
		entity.Description,
		entity.IconURL,
		entity.SortOrder,
		entity.UpdatedAt,
		entity.ID,
	)

	updated := &category.Category{}
	err := row.Scan(
		&updated.ID,
		&updated.Name,
		&updated.Slug,
		&updated.ParentID,
		&updated.SortOrder,
		&updated.Description,
		&updated.IconURL,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, category.ErrCategoryNotFound
		}

		// Check constraint violations
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "idx_categories_slug" {
				fmt.Println("Update: duplicate slug", pgErr)
				return nil, category.ErrDuplicateSlug
			}
		}

		fmt.Println("Update: database error", err)
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	levelQuery := `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 0 as depth
			FROM categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.depth + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
			WHERE pc.parent_id IS NOT NULL
		)
		SELECT 
			(SELECT MAX(depth) FROM parent_chain) + 1 as level,
			COALESCE((
				SELECT COUNT(*)
				FROM categories
				WHERE parent_id = $1 AND is_active = true
			), 0) as children_count
	`

	var level int
	var childrenCount int
	err = r.pool.QueryRow(ctx, levelQuery, updated.ID).Scan(&level, &childrenCount)
	if err != nil {
		logger.Error("Update: failed to calculate level", err)
		// Set defaults (not ideal, but don't fail the whole update)
		level = 1
		childrenCount = 0
	}

	updated.Level = &level
	updated.ChildCount = &childrenCount

	return updated, nil
}

// ============================================================
// UPDATE: MoveToParent
// ============================================================
func (r *postgresRepository) MoveToParent(
	ctx context.Context,
	categoryID uuid.UUID,
	newParentID *uuid.UUID,
) (*category.Category, error) {
	const query = `
		UPDATE categories
		SET parent_id = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, newParentID, categoryID)

	updated := &category.Category{}
	err := row.Scan(
		&updated.ID,
		&updated.Name,
		&updated.Slug,
		&updated.ParentID,
		&updated.SortOrder,
		&updated.Description,
		&updated.IconURL,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error("MoveToParent: category not found", err)
			return nil, category.ErrCategoryNotFound
		}
		logger.Error("MoveToParent: database error", err)
		return nil, fmt.Errorf("failed to move category: %w", err)
	}

	// ========== Calculate level + children_count after move ==========
	levelQuery := `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 0 as depth
			FROM categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.depth + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
			WHERE pc.parent_id IS NOT NULL
		)
		SELECT 
			(SELECT MAX(depth) FROM parent_chain) + 1 as level,
			COALESCE((
				SELECT COUNT(*)
				FROM categories
				WHERE parent_id = $1 AND is_active = true
			), 0) as children_count
	`

	var level int
	var childrenCount int
	err = r.pool.QueryRow(ctx, levelQuery, updated.ID).Scan(&level, &childrenCount)
	if err != nil {
		logger.Error("MoveToParent: failed to calculate level", err)
		level = 1
		childrenCount = 0
	}

	updated.Level = &level
	updated.ChildCount = &childrenCount

	return updated, nil
}

// ============================================================
// UPDATE: Activate
// ============================================================
func (r *postgresRepository) Activate(
	ctx context.Context,
	categoryID uuid.UUID,
) (*category.Category, error) {
	const query = `
		UPDATE categories
		SET is_active = true, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, categoryID)

	updated := &category.Category{}
	err := row.Scan(
		&updated.ID,
		&updated.Name,
		&updated.Slug,
		&updated.ParentID,
		&updated.SortOrder,
		&updated.Description,
		&updated.IconURL,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error("Activate: category not found", err)
			return nil, category.ErrCategoryNotFound
		}
		logger.Error("Activate: database error", err)
		return nil, fmt.Errorf("failed to activate category: %w", err)
	}

	// ========== Calculate level + children_count ==========
	levelQuery := `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 0 as depth
			FROM categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.depth + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
			WHERE pc.parent_id IS NOT NULL
		)
		SELECT 
			(SELECT MAX(depth) FROM parent_chain) + 1 as level,
			COALESCE((
				SELECT COUNT(*)
				FROM categories
				WHERE parent_id = $1 AND is_active = true
			), 0) as children_count
	`

	var level int
	var childrenCount int
	err = r.pool.QueryRow(ctx, levelQuery, updated.ID).Scan(&level, &childrenCount)
	if err != nil {
		level = 1
		childrenCount = 0
	}

	updated.Level = &level
	updated.ChildCount = &childrenCount

	return updated, nil
}

func (r *postgresRepository) Deactivate(
	ctx context.Context,
	categoryID uuid.UUID,
) (*category.Category, error) {
	// Get descendants
	descendants, err := r.GetDescendants(ctx, categoryID)
	if err != nil {
		logger.Error("Deactivate: failed to get descendants", err)
		return nil, fmt.Errorf("failed to deactivate category: %w", err)
	}

	// Build ID list
	ids := make([]uuid.UUID, 0, len(descendants)+1)
	ids = append(ids, categoryID)
	for _, d := range descendants {
		ids = append(ids, d.ID)
	}

	// Update all
	const query = `
		UPDATE categories
		SET is_active = false, updated_at = NOW()
		WHERE id = ANY($1::uuid[])
		RETURNING id, name, slug, parent_id, sort_order, description, icon_url, is_active, created_at, updated_at
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		logger.Error("Deactivate: database error", err)
		return nil, fmt.Errorf("failed to deactivate category: %w", err)
	}
	defer rows.Close()

	var updated *category.Category
	for rows.Next() {
		entity := &category.Category{}
		err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Slug,
			&entity.ParentID,
			&entity.SortOrder,
			&entity.Description,
			&entity.IconURL,
			&entity.IsActive,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		)

		if err != nil {
			logger.Error("Deactivate: scan error", err)
			return nil, fmt.Errorf("failed to deactivate category: %w", err)
		}

		if entity.ID == categoryID {
			updated = entity
		}
	}

	if updated == nil {
		logger.Error("Deactivate: category not found", fmt.Errorf("id: %s", categoryID.String()))
		return nil, category.ErrCategoryNotFound
	}

	// ========== Calculate level + children_count ==========
	levelQuery := `
		WITH RECURSIVE parent_chain AS (
			SELECT id, parent_id, 0 as depth
			FROM categories
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, pc.depth + 1
			FROM categories c
			INNER JOIN parent_chain pc ON c.id = pc.parent_id
			WHERE pc.parent_id IS NOT NULL
		)
		SELECT 
			(SELECT MAX(depth) FROM parent_chain) + 1 as level,
			COALESCE((
				SELECT COUNT(*)
				FROM categories
				WHERE parent_id = $1 AND is_active = true
			), 0) as children_count
	`

	var level int
	var childrenCount int
	err = r.pool.QueryRow(ctx, levelQuery, updated.ID).Scan(&level, &childrenCount)
	if err != nil {
		level = 1
		childrenCount = 0
	}

	updated.Level = &level
	updated.ChildCount = &childrenCount

	return updated, nil
}

// ============================================================
// UPDATE: BulkActivate
// ============================================================
func (r *postgresRepository) BulkActivate(
	ctx context.Context,
	categoryIDs []uuid.UUID,
) (int64, error) {
	const query = `
		UPDATE categories
		SET is_active = true, updated_at = NOW()
		WHERE id = ANY($1::uuid[])
	`

	result, err := r.pool.Exec(ctx, query, categoryIDs)
	if err != nil {
		fmt.Println("BulkActivate: database error", err)
		return 0, fmt.Errorf("failed to bulk activate: %w", err)
	}

	count := result.RowsAffected()

	return count, nil
}

// ============================================================
// UPDATE: BulkDeactivate
// ============================================================
func (r *postgresRepository) BulkDeactivate(
	ctx context.Context,
	categoryIDs []uuid.UUID,
) (int64, error) {
	// ========== Get All Descendants ==========
	// For each ID, get descendants
	// Then deactivate all
	allIDs := make(map[uuid.UUID]bool)

	// Add main IDs
	for _, id := range categoryIDs {
		allIDs[id] = true
	}

	// Get descendants for each
	for _, id := range categoryIDs {
		descendants, err := r.GetDescendants(ctx, id)
		if err != nil {
			fmt.Println("BulkDeactivate: failed to get descendants", err)
			continue // Continue with others
		}

		for _, d := range descendants {
			allIDs[d.ID] = true
		}
	}

	// Convert map to slice
	ids := make([]uuid.UUID, 0, len(allIDs))
	for id := range allIDs {
		ids = append(ids, id)
	}

	// ========== Update All ==========
	const query = `
		UPDATE categories
		SET is_active = false, updated_at = NOW()
		WHERE id = ANY($1::uuid[])
	`

	result, err := r.pool.Exec(ctx, query, ids)
	if err != nil {
		fmt.Println("BulkDeactivate: database error", err)
		return 0, fmt.Errorf("failed to bulk deactivate: %w", err)
	}

	count := result.RowsAffected()

	return count, nil
}

// ============================================================
// DELETE: Delete
// ============================================================
func (r *postgresRepository) Delete(
	ctx context.Context,
	categoryID uuid.UUID,
) error {
	const query = `
		DELETE FROM categories
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, categoryID)
	if err != nil {
		fmt.Println("Delete: database error", err)
		return fmt.Errorf("failed to delete category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return category.ErrCategoryNotFound
	}

	return nil
}

// ============================================================
// DELETE: BulkDelete
// ============================================================
func (r *postgresRepository) BulkDelete(
	ctx context.Context,
	categoryIDs []uuid.UUID,
) (int64, error) {
	const query = `
		DELETE FROM categories
		WHERE id = ANY($1::uuid[])
	`

	result, err := r.pool.Exec(ctx, query, categoryIDs)
	if err != nil {
		fmt.Println("BulkDelete: database error", err)
		return 0, fmt.Errorf("failed to bulk delete: %w", err)
	}

	count := result.RowsAffected()

	return count, nil
}

// ============================================================
// VALIDATION: ExistsBySlug
// ============================================================
// ExistsBySlug checks if slug exists (excluding specific ID)
//
// DATABASE:
// SELECT EXISTS(
//
//	SELECT 1 FROM categories
//	WHERE slug = $1 AND id != $2
//
// )
//
// EXCLUDE ID:
// When updating, exclude current ID
// So doesn't match itself
//
// USAGE:
// Create: ExistsBySlug(slug, nil) => check if exists
// Update: ExistsBySlug(slug, categoryID) => exclude self
func (r *postgresRepository) ExistsBySlug(
	ctx context.Context,
	slug string,
	excludeID *uuid.UUID,
) (bool, error) {
	var query string
	var args []interface{}

	if excludeID == nil {
		// Create case: check if slug exists anywhere
		query = "SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1)"
		args = []interface{}{slug}
	} else {
		// Update case: exclude this ID
		query = "SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1 AND id != $2)"
		args = []interface{}{slug, *excludeID}
	}

	var exists bool
	err := r.pool.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}

	return exists, nil
}

// ============================================================
// VALIDATION: ExistsByID
// ============================================================
func (r *postgresRepository) ExistsByID(
	ctx context.Context,
	id uuid.UUID,
) (bool, error) {
	const query = "SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)"

	var exists bool
	err := r.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check category existence: %w", err)
	}

	return exists, nil
}

// ============================================================
// VALIDATION: HasChildren
// ============================================================
func (r *postgresRepository) HasChildren(
	ctx context.Context,
	categoryID uuid.UUID,
) (bool, error) {
	const query = "SELECT EXISTS(SELECT 1 FROM categories WHERE parent_id = $1)"

	var hasChildren bool
	err := r.pool.QueryRow(ctx, query, categoryID).Scan(&hasChildren)
	if err != nil {
		return false, fmt.Errorf("failed to check children: %w", err)
	}

	return hasChildren, nil
}

// ============================================================
// BOOK-RELATED: GetBooksInCategory
// ============================================================
// GetBooksInCategory lấy books trong category (bao gồm descendants)
//
// DATABASE:
// SELECT DISTINCT b.id FROM books b
// INNER JOIN categories c ON b.category_id = c.id
// WHERE c.id IN (descendants of categoryID)
// AND b.is_active = true
func (r *postgresRepository) GetBooksInCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	limit int,
	offset int,
) ([]uuid.UUID, int64, error) {
	// ========== Get Active Descendants ONLY ==========
	// Tạo recursive CTE để get active descendants
	const descendantsQuery = `
		WITH RECURSIVE descendants AS (
			SELECT id
			FROM categories
			WHERE id = $1 AND is_active = true

			UNION ALL

			SELECT c.id
			FROM categories c
			INNER JOIN descendants d ON c.parent_id = d.id
			WHERE c.is_active = true
		)
		SELECT id FROM descendants
	`

	rows, err := r.pool.Query(ctx, descendantsQuery, categoryID)
	if err != nil {
		logger.Error("GetBooksInCategory: failed to get descendants", err)
		return nil, 0, fmt.Errorf("failed to get books: %w", err)
	}
	defer rows.Close()

	categoryIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			logger.Error("GetBooksInCategory: scan error", err)
			return nil, 0, fmt.Errorf("failed to get books: %w", err)
		}
		categoryIDs = append(categoryIDs, id)
	}

	if err = rows.Err(); err != nil {
		logger.Error("GetBooksInCategory: rows error", err)
		return nil, 0, fmt.Errorf("failed to get books: %w", err)
	}

	// If no active categories found
	if len(categoryIDs) == 0 {
		return []uuid.UUID{}, 0, nil
	}

	// ========== Count Query ==========
	const countQuery = `
		SELECT COUNT(DISTINCT b.id)
		FROM books b
		WHERE b.category_id = ANY($1::uuid[])
		AND b.is_active = true
	`

	var total int64
	err = r.pool.QueryRow(ctx, countQuery, categoryIDs).Scan(&total)
	if err != nil {
		logger.Error("GetBooksInCategory: count query failed", err)
		return nil, 0, fmt.Errorf("failed to get books: %w", err)
	}

	// ========== List Query ==========
	const listQuery = `
		SELECT DISTINCT b.id
		FROM books b
		WHERE b.category_id = ANY($1::uuid[])
		AND b.is_active = true
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err = r.pool.Query(ctx, listQuery, categoryIDs, limit, offset)
	if err != nil {
		logger.Error("GetBooksInCategory: query failed", err)
		return nil, 0, fmt.Errorf("failed to get books: %w", err)
	}
	defer rows.Close()

	bookIDs := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var bookID uuid.UUID
		err := rows.Scan(&bookID)
		if err != nil {
			logger.Error("GetBooksInCategory: scan error", err)
			return nil, 0, fmt.Errorf("failed to scan book: %w", err)
		}
		bookIDs = append(bookIDs, bookID)
	}

	if err = rows.Err(); err != nil {
		logger.Error("GetBooksInCategory: rows error", err)
		return nil, 0, fmt.Errorf("failed to get books: %w", err)
	}

	return bookIDs, total, nil
}

// ============================================================
// BOOK-RELATED: GetCategoryBookCount
// ============================================================
func (r *postgresRepository) GetCategoryBookCount(
	ctx context.Context,
	categoryID uuid.UUID,
) (int64, error) {
	// ========== Get Active Descendants ONLY ==========
	const descendantsQuery = `
		WITH RECURSIVE descendants AS (
			SELECT id
			FROM categories
			WHERE id = $1 AND is_active = true

			UNION ALL

			SELECT c.id
			FROM categories c
			INNER JOIN descendants d ON c.parent_id = d.id
			WHERE c.is_active = true
		)
		SELECT id FROM descendants
	`

	rows, err := r.pool.Query(ctx, descendantsQuery, categoryID)
	if err != nil {
		logger.Error("GetCategoryBookCount: failed to get descendants", err)
		return 0, fmt.Errorf("failed to get book count: %w", err)
	}
	defer rows.Close()

	categoryIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			logger.Error("GetCategoryBookCount: scan error", err)
			return 0, fmt.Errorf("failed to get book count: %w", err)
		}
		categoryIDs = append(categoryIDs, id)
	}

	if err = rows.Err(); err != nil {
		logger.Error("GetCategoryBookCount: rows error", err)
		return 0, fmt.Errorf("failed to get book count: %w", err)
	}

	// If no active categories found
	if len(categoryIDs) == 0 {
		return 0, nil
	}

	// ========== Count Query ==========
	const query = `
		SELECT COUNT(*)
		FROM books
		WHERE category_id = ANY($1::uuid[])
		AND is_active = true
	`

	var count int64
	err = r.pool.QueryRow(ctx, query, categoryIDs).Scan(&count)
	if err != nil {
		logger.Error("GetCategoryBookCount: database error", err)
		return 0, fmt.Errorf("failed to get book count: %w", err)
	}

	return count, nil
}

// FindByNameCaseInsensitive tìm category by name (case-insensitive)
func (r *postgresRepository) FindByNameCaseInsensitive(ctx context.Context, name string) (*category.Category, error) {
	query := `
        SELECT id, name, slug, description, parent_id,
               created_at, updated_at
        FROM categories
        WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
        LIMIT 1
    `

	var category category.Category
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&category.ParentID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	return &category, nil
}

// FindBySlugWithTx tìm category by slug (trong transaction)
func (r *postgresRepository) FindBySlugWithTx(ctx context.Context, tx pgx.Tx, slug string) (*category.Category, error) {
	query := `
        SELECT id, name, slug, description, parent_id,
               created_at, updated_at
        FROM categories
        WHERE slug = $1
        LIMIT 1
    `

	var category category.Category
	err := tx.QueryRow(ctx, query, slug).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&category.ParentID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	return &category, nil
}

// CreateWithTx tạo category trong transaction
func (r *postgresRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, category *category.Category) error {
	query := `
        INSERT INTO categories (id, name, slug, description, parent_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	now := time.Now()
	category.CreatedAt = now
	category.UpdatedAt = now

	_, err := tx.Exec(ctx, query,
		category.ID,
		category.Name,
		category.Slug,
		category.Description,
		category.ParentID,
		category.CreatedAt,
		category.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}
