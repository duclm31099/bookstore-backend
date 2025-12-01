package category

import (
	"bookstore-backend/internal/domains/book/model"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================
// REPOSITORY INTERFACE: CategoryRepository
// ============================================================

type CategoryRepository interface {
	Create(ctx context.Context, category *Category) (*Category, error)

	GetByID(ctx context.Context, id uuid.UUID) (*Category, error)

	GetBySlug(ctx context.Context, slug string) (*Category, error)

	GetAll(ctx context.Context, filter *CategoryFilter) ([]Category, int64, error)

	// NEW: Methods for bulk import
	FindByNameCaseInsensitive(ctx context.Context, name string) (*Category, error)
	FindBySlugWithTx(ctx context.Context, tx pgx.Tx, slug string) (*Category, error)
	CreateWithTx(ctx context.Context, tx pgx.Tx, category *Category) error

	// GetTree lấy toàn bộ category tree (recursive)
	// SELECT * FROM category_tree (đã ordered by path)
	// Văn học (path=[0])
	//   Tiểu thuyết (path=[0,0])
	//     Trinh thám (path=[0,0,0])
	//     Tình cảm (path=[0,0,1])
	//   Thơ (path=[0,1])
	// LỢI ÍCH:
	// - Recursive CTE tính 1 lần, cache result
	// - Query nhanh (1ms vs 100ms)
	// - Tree structure đã sắp xếp
	//
	// TRADE-OFF:
	// - Extra storage (duplicate data)
	// - Manual refresh (sau khi insert/update/delete)
	GetTree(ctx context.Context) ([]Category, error)

	// GetChildren lấy tất cả children direct (level+1)
	// PARAMS:
	// - parentID: Category ID
	// RETURN:
	// - []Category: Children (sorted by sort_order)
	// - error
	// - Chỉ direct children (level = parent.level + 1)
	// - Không bao gồm descendants (grandchildren)
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]Category, error)

	// GetDescendants lấy tất cả descendants (con + cháu + ...)
	// PARAMS:
	// - categoryID: Category ID
	//
	// RETURN:
	// - []Category: Tất cả descendants
	// - error
	//
	// DATABASE (Recursive CTE):
	// WITH RECURSIVE descendants AS (
	//   SELECT * FROM categories WHERE id = $1
	//   UNION ALL
	//   SELECT c.* FROM categories c
	//   INNER JOIN descendants d ON c.parent_id = d.id
	// )
	// SELECT * FROM descendants
	//
	// KHÁI NIỆM - Recursive CTE là gì?
	// CTE (Common Table Expression): Subquery named
	// RECURSIVE: Có thể reference chính nó
	//
	// FLOW:
	// 1. Base case: SELECT category với id = $1
	// 2. Recursive case: SELECT con của base case
	//    => Tìm con của con => tìm cháu
	//    => Tiếp tục recursive đến khi không có con
	// 3. UNION ALL: Kết hợp tất cả
	//
	// VÍ DỤ:
	// Iteration 1: [Tiểu thuyết]
	// Iteration 2: [Trinh thám, Tình cảm]
	// Iteration 3: [] (no more)
	// RESULT: [Tiểu thuyết, Trinh thám, Tình cảm]
	//
	// PERFORMANCE:
	// - Recursive: ~100ms cho 10k categories
	// - Limit depth: 3 levels => max 3 iterations
	GetDescendants(ctx context.Context, categoryID uuid.UUID) ([]Category, error)

	// GetAncestors lấy tất cả ancestors (cha + ông + ...)
	//
	// PARAMS:
	// - categoryID: Category ID
	//
	// RETURN:
	// - []Category: Tất cả ancestors (từ root tới parent)
	// - error
	//
	// DATABASE (Recursive CTE từ dưới lên trên):
	// WITH RECURSIVE ancestors AS (
	//   SELECT * FROM categories WHERE id = $1
	//   UNION ALL
	//   SELECT c.* FROM categories c
	//   INNER JOIN ancestors a ON c.id = a.parent_id
	// )
	// SELECT * FROM ancestors ORDER BY level ASC
	//
	// VÍ DỤ:
	// GetAncestors(trinh_thám_id) =>
	// [
	//   {Name: "Văn học", level: 1},
	//   {Name: "Tiểu thuyết", level: 2},
	//   {Name: "Trinh thám", level: 3},
	// ]
	//
	// USE CASE:
	// - Breadcrumb navigation
	// - Validate không move tới descendant
	GetAncestors(ctx context.Context, categoryID uuid.UUID) ([]Category, error)

	// GetCategoryBreadcrumb lấy breadcrumb
	//
	// Giống GetAncestors nhưng có thêm xử lý
	GetCategoryBreadcrumb(ctx context.Context, categoryID uuid.UUID) ([]Category, error)

	// ========== UPDATE ==========

	// Update cập nhật category
	//
	// PARAMS:
	// - ctx: Context
	// - category: *Category với fields update
	//
	// RETURN:
	// - *Category: Category sau update
	// - error
	//
	// DATABASE:
	// UPDATE categories
	// SET name = $1, slug = $2, description = $3, icon_url = $4, sort_order = $5, updated_at = NOW()
	// WHERE id = $6
	//
	// BUSINESS RULES:
	// - Không update ID, CreatedAt
	// - Validate slug không duplicate
	// - Không update ParentID (dùng MoveToParent)
	// - Không update IsActive (dùng Activate/Deactivate)
	Update(ctx context.Context, category *Category) (*Category, error)

	// MoveToParent di chuyển category tới parent khác
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: Category cần move
	// - newParentID: Parent mới (nil = move to root)
	//
	// RETURN:
	// - *Category: Category sau move
	// - error
	//
	// DATABASE:
	// UPDATE categories SET parent_id = $1, updated_at = NOW() WHERE id = $2
	//
	// BUSINESS RULES:
	// - Validate: newParent không phải descendant của categoryID
	//   (Nếu move, sẽ tạo circular reference)
	// - Validate: max depth = 3
	//   if newParent.level + 1 > 3 => ERROR
	//
	// VALIDATION LOGIC:
	// 1. GetAncestors(newParentID) => ancestors_list
	// 2. if categoryID in ancestors_list => ERROR (circular)
	// 3. GetDescendants(categoryID) => descendants_list
	// 4. if newParentID in descendants_list => ERROR (circular)
	//
	// VÍ DỤ - Circular Reference:
	// Tree: A > B > C > D
	// MoveToParent(A, D) => ERROR
	// Vì D là descendant của A, nếu move A vào D, sẽ: D > A > B > C > D (cycle!)
	MoveToParent(ctx context.Context, categoryID uuid.UUID, newParentID *uuid.UUID) (*Category, error)

	// Activate kích hoạt category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: Category ID
	//
	// RETURN:
	// - *Category
	// - error
	//
	// DATABASE:
	// UPDATE categories SET is_active = true, updated_at = NOW() WHERE id = $1
	//
	// BUSINESS RULE:
	// - Không activate nếu parent inactive
	//   (Vì parent inactive => category invisible anyway)
	//
	// VALIDATION:
	// 1. GetByID(categoryID)
	// 2. if parent.is_active = false => ERROR
	// 3. UPDATE is_active = true
	Activate(ctx context.Context, categoryID uuid.UUID) (*Category, error)

	// Deactivate vô hiệu hóa category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: Category ID
	//
	// RETURN:
	// - *Category
	// - error
	//
	// SIDE EFFECT:
	// - Deactivate tất cả descendants (recursive)
	// - Books trong category này không hiển thị
	//
	// DATABASE:
	// 1. GetDescendants(categoryID) => descendants
	// 2. UPDATE categories SET is_active = false
	//    WHERE id IN (categoryID + descendants IDs)
	//
	// TRANSACTION:
	// - Atomic: Tất cả update hoặc không update nếu error
	// - Prevent partial update
	Deactivate(ctx context.Context, categoryID uuid.UUID) (*Category, error)

	// BulkActivate kích hoạt nhiều categories
	//
	// PARAMS:
	// - ctx: Context
	// - categoryIDs: []uuid.UUID
	//
	// RETURN:
	// - int64: Số categories updated
	// - error
	//
	// DATABASE:
	// UPDATE categories SET is_active = true WHERE id = ANY($1::uuid[])
	//
	// OPTIMIZATION:
	// - Dùng ANY() => 1 query thay vì N queries
	// - Return count để client biết bao nhiêu updated
	//
	// PERFORMANCE:
	// - 1000 items: ~50ms (1 query)
	// - vs 1000 items: ~5000ms (1000 queries)
	BulkActivate(ctx context.Context, categoryIDs []uuid.UUID) (int64, error)

	// BulkDeactivate vô hiệu hóa nhiều categories
	BulkDeactivate(ctx context.Context, categoryIDs []uuid.UUID) (int64, error)

	// ========== DELETE ==========

	// Delete xóa 1 category (hard delete)
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: Category ID
	//
	// RETURN:
	// - error
	//
	// DATABASE:
	// DELETE FROM categories WHERE id = $1
	//
	// BUSINESS RULES:
	// - Không có children
	// - Không có books
	//
	// VALIDATION:
	// 1. GetByID(categoryID) => category
	// 2. if !category.CanDelete() => ERROR
	// 3. DELETE
	Delete(ctx context.Context, categoryID uuid.UUID) error

	// BulkDelete xóa nhiều categories
	BulkDelete(ctx context.Context, categoryIDs []uuid.UUID) (int64, error)

	// ========== BOOK-RELATED OPERATIONS ==========

	// GetBooksInCategory lấy tất cả books trong category (bao gồm children)
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: Category ID
	// - limit, offset: Pagination
	//
	// RETURN:
	// - []uuid.UUID: Book IDs
	// - int64: Total count
	// - error
	//
	// DATABASE:
	// SELECT DISTINCT b.id FROM books b
	// INNER JOIN categories c ON b.category_id = c.id
	// WHERE c.id IN (
	//   WITH RECURSIVE d AS (
	//     SELECT id FROM categories WHERE id = $1
	//     UNION ALL
	//     SELECT c2.id FROM categories c2 INNER JOIN d ON c2.parent_id = d.id
	//   )
	//   SELECT id FROM d
	// )
	// AND b.is_active = true
	// ORDER BY b.created_at DESC
	// LIMIT $limit OFFSET $offset
	//
	// FLOW:
	// 1. Get category + descendants
	// 2. Join với books table
	// 3. Filter active books
	// 4. Paginate
	//
	// USE CASE:
	// - Category page: "Tất cả sách trong mục này"
	// - Display: "Có X cuốn sách"
	GetBooksInCategory(ctx context.Context, categoryID uuid.UUID, limit int, offset int) ([]model.Book, int64, error)

	// GetCategoryBookCount lấy số books
	GetCategoryBookCount(ctx context.Context, categoryID uuid.UUID) (int64, error)

	// ========== VALIDATION / CHECK ==========

	// ExistsBySlug kiểm tra slug tồn tại
	ExistsBySlug(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)

	// ExistsByID kiểm tra ID tồn tại
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// HasChildren kiểm tra có children không
	HasChildren(ctx context.Context, categoryID uuid.UUID) (bool, error)
}
