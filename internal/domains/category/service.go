package category

import (
	"bookstore-backend/internal/domains/book/model"
	"context"

	"github.com/google/uuid"
)

// ============================================================
// 📚 KHÁI NIỆM: Service Layer
// ============================================================
// Service layer là nơi chứa business logic
// Nó tập trung các rule, validation, orchestration
//
// SERVICE LAYER RESPONSIBILITIES:
// 1. Business Logic: Règles métier (business rules)
//    - Create category với validation
//    - Move category với check circular reference
//    - Deactivate category với cascade inactive
//
// 2. Orchestration: Điều phối (combine multiple repositories)
//    - Create order: check inventory + deduct stock + charge payment
//    - Move category: check circular + check depth + update
//
// 3. Validation: Xác nhận dữ liệu từ Handler
//    - Validate input (DTO => Entity)
//    - Business rule validation (not just type)
//
// 4. Error Handling: Xử lý lỗi từ Repository
//    - Wrap error với context
//    - Return domain-specific error
//
// 5. Transaction: Quản lý transaction (nếu cần)
//    - Atomic: All or nothing
//    - ACID: Consistency guarantee
//
// LAYER ARCHITECTURE:
// ┌─────────────────────────────────────────────────┐
// │ Handler (HTTP)                                  │
// │ - Route parsing                                 │
// │ - HTTP response formatting                      │
// └────────────────┬────────────────────────────────┘
//                  │ HTTP request body (JSON)
//                  │ c.BindJSON(&req)
//                  ▼
// ┌─────────────────────────────────────────────────┐
// │ Service (Business Logic) ◄── WE ARE HERE       │
// │ - Create/Update/Delete logic                    │
// │ - Validation                                    │
// │ - Orchestration                                 │
// │ - Transaction                                   │
// └────────────────┬────────────────────────────────┘
//                  │ Calls Repository.Create()
//                  ▼
// ┌─────────────────────────────────────────────────┐
// │ Repository (Data Access)                        │
// │ - Database queries                              │
// │ - Cache operations                              │
// └────────────────┬────────────────────────────────┘
//                  │ SQL query
//                  ▼
// ┌─────────────────────────────────────────────────┐
// │ Database                                        │
// │ - PostgreSQL                                    │
// └─────────────────────────────────────────────────┘

// ============================================================
// SERVICE INTERFACE: CategoryService
// ============================================================
// Interface là contract: "Bất cứ ai implement tôi phải cung cấp những methods này"
//
// WHY INTERFACE?
//
//  1. Decoupling: Handler không depend vào implementation
//     ❌ handler.service = &PostgresCategoryService{} (tight coupling)
//     ✅ handler.service CategoryService (loose coupling)
//
//  2. Testability: Mock service cho testing
//     type MockCategoryService struct { ... }
//     test không cần real database
//
// 3. Flexibility: Multiple implementations
//   - PostgresCategoryService (real DB)
//   - CachedCategoryService (with cache layer)
//   - MockCategoryService (for testing)
//
// DEPENDENCY INJECTION:
//
//	type Handler struct {
//	  categoryService CategoryService  // Inject interface
//	}
//
//	func NewHandler(svc CategoryService) *Handler {
//	  return &Handler{categoryService: svc}
//	}
type CategoryService interface {
	// ========== CREATE OPERATIONS ==========

	// Create tạo category mới
	//
	// PARAMS:
	// - ctx: Context (timeout, cancellation)
	// - req: *CreateCategoryReq (request DTO từ handler)
	//
	// RETURN:
	// - *CategoryResp: Response DTO
	// - error: Domain error
	//
	// BUSINESS LOGIC:
	// 1. Validate input (req không nil, fields valid)
	// 2. Create entity: NewCategory(req.Name, ...)
	// 3. If ParentID provided:
	//    a. Check parent exists
	//    b. Check max depth not exceeded
	// 4. Check slug not duplicate
	// 5. Repository.Create(category)
	// 6. Map entity to response DTO
	// 7. Return response
	//
	// ERROR CASES:
	// - ErrInvalidCategoryName: Validation fail
	// - ErrParentNotFound: Parent not exist
	// - ErrMaxDepthExceeded: Tree too deep
	// - ErrDuplicateSlug: Slug already exist
	// - Other database errors
	//
	// FLOW DIAGRAM:
	// Request DTO
	//   ↓
	// Validate input
	//   ↓
	// NewCategory (Entity) ← domain validation
	//   ↓
	// Check parent exists
	//   ↓
	// Check depth
	//   ↓
	// Check slug unique
	//   ↓
	// Repository.Create ← DB operation
	//   ↓
	// Map to Response DTO
	//   ↓
	// Return Response
	Create(ctx context.Context, req *CreateCategoryReq) (*CategoryResp, error)

	// ========== READ OPERATIONS ==========

	// GetByID lấy category theo ID
	//
	// PARAMS:
	// - ctx: Context
	// - id: uuid.UUID
	//
	// RETURN:
	// - *CategoryResp
	// - error: ErrCategoryNotFound nếu không tìm thấy
	//
	// BUSINESS LOGIC:
	// 1. Validate ID (không nil)
	// 2. Repository.GetByID(id)
	// 3. Check result not nil => ErrCategoryNotFound
	// 4. Map to response DTO
	// 5. Return response
	//
	// OPTIMIZATION:
	// - Cache result (future): Repository có cache layer
	// - If category inactive => trả về DTO.IsActive = false
	GetByID(ctx context.Context, id uuid.UUID) (*CategoryResp, error)

	// GetBySlug lấy category theo slug
	//
	// PARAMS:
	// - ctx: Context
	// - slug: string (URL-friendly identifier)
	//
	// RETURN:
	// - *CategoryResp
	// - error: ErrCategoryNotFound
	//
	// BUSINESS LOGIC:
	// 1. Validate slug not empty
	// 2. Repository.GetBySlug(slug)
	// 3. Check result not nil => ErrCategoryNotFound
	// 4. Map to response DTO
	// 5. Return response
	//
	// USE CASE:
	// GET /v1/categories/tieu-thuyet
	// => Service.GetBySlug("tieu-thuyet")
	GetBySlug(ctx context.Context, slug string) (*CategoryResp, error)

	// GetAll lấy danh sách categories
	//
	// PARAMS:
	// - ctx: Context
	// - isActive: *bool (filter)
	// - parentID: *uuid.UUID (filter)
	// - limit, offset: Pagination
	//
	// RETURN:
	// - *CategoryListResp: List + pagination info
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate pagination (limit > 0, offset >= 0)
	// 2. Build CategoryFilter từ params
	// 3. Repository.GetAll(filter)
	// 4. Map to response DTOs
	// 5. Build CategoryListResp với total, limit, offset, hasMore
	// 6. Return response
	//
	// PAGINATION LOGIC:
	// hasMore = offset + limit < total
	// VÍ DỤ:
	// - offset=0, limit=10, total=25
	// - hasMore = 0 + 10 < 25 = true
	//
	// - offset=20, limit=10, total=25
	// - hasMore = 20 + 10 < 25 = false
	GetAll(
		ctx context.Context,
		isActive *bool,
		parentID *uuid.UUID,
		limit int,
		offset int,
	) (*CategoryListResp, error)

	// GetTree lấy toàn bộ category tree
	//
	// PARAMS:
	// - ctx: Context
	//
	// RETURN:
	// - []CategoryTreeItemResp: Ordered tree items
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Repository.GetTree() ← Materialized View
	// 2. Map to tree item DTOs
	// 3. Return list
	//
	// PERFORMANCE:
	// - Materialized View: ~1ms
	// - No pagination: Return all (tree size manageable)
	//
	// TREE STRUCTURE:
	// [
	//   {Name: "Văn học", Level: 1, FullPath: "Văn học"},
	//   {Name: "Tiểu thuyết", Level: 2, FullPath: "Văn học > Tiểu thuyết"},
	//   {Name: "Trinh thám", Level: 3, FullPath: "Văn học > Tiểu thuyết > Trinh thám"},
	// ]
	GetTree(ctx context.Context) ([]CategoryTreeItemResp, error)

	// GetBreadcrumb lấy breadcrumb cho 1 category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: uuid.UUID
	//
	// RETURN:
	// - *CategoryBreadcrumbResp: Items + current path
	// - error: ErrCategoryNotFound
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID not nil
	// 2. Repository.GetCategoryBreadcrumb(categoryID)
	// 3. Build response:
	//    a. Items: Array breadcrumb items
	//    b. CurrentPath: FullPath string
	// 4. Return response
	//
	// USE CASE:
	// GET /v1/categories/trinh-tham/breadcrumb
	// Response: {
	//   items: [
	//     {name: "Văn học", ...},
	//     {name: "Tiểu thuyết", ...},
	//     {name: "Trinh thám", ...}
	//   ],
	//   current_path: "Văn học > Tiểu thuyết > Trinh thám"
	// }
	GetBreadcrumb(ctx context.Context, categoryID uuid.UUID) (*CategoryBreadcrumbResp, error)

	// ========== UPDATE OPERATIONS ==========

	// Update cập nhật category
	//
	// PARAMS:
	// - ctx: Context
	// - id: uuid.UUID (category ID)
	// - req: *UpdateCategoryReq (partial update)
	//
	// RETURN:
	// - *CategoryResp: Updated category
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate ID exists: Repository.ExistsByID(id)
	// 2. Validate request: Check fields not nil
	// 3. Get current category: Repository.GetByID(id)
	// 4. Apply updates:
	//    - if req.Name != nil => update name
	//    - if req.Description != nil => update description
	//    - etc.
	// 5. Call entity.Update(name, desc, icon, order)
	// 6. Validate slug not duplicate: if req.Name changed
	// 7. Repository.Update(updated_category)
	// 8. Map to response DTO
	// 9. Return response
	//
	// PARTIAL UPDATE:
	// PUT /v1/categories/123
	// {"name": "New Name"}
	// => Only update name, keep other fields
	//
	// ALGORITHM:
	// - Dùng pointer fields để detect "not provided"
	// - nil = not provided, update to nil (omit)
	// - value = provided, update to value
	Update(ctx context.Context, id uuid.UUID, req *UpdateCategoryReq) (*CategoryResp, error)

	// MoveToParent di chuyển category tới parent khác
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: Category cần move
	// - req: *MoveToParentReq (new parent ID)
	//
	// RETURN:
	// - *CategoryResp: Updated category
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID exists
	// 2. Validate req.ParentID not equal categoryID => ErrInvalidParentID
	// 3. If req.ParentID != nil:
	//    a. Check parent exists
	//    b. Check circular reference:
	//       - GetAncestors(req.ParentID) => ancestors
	//       - if categoryID in ancestors => ERROR (circular!)
	//    c. Check max depth:
	//       - Get new parent: Repository.GetByID(req.ParentID)
	//       - if newParent.level + 1 > MAX_DEPTH => ERROR
	// 4. Repository.MoveToParent(categoryID, req.ParentID)
	// 5. Get updated category
	// 6. Map to response DTO
	// 7. Return response
	//
	// CIRCULAR REFERENCE PREVENTION:
	// Tree: A > B > C
	// MoveToParent(A, C) => ERROR
	// Because: C is descendant of A
	// If move: C > A > B > C (cycle!)
	//
	// VALIDATION:
	// 1. GetAncestors(C) => [A, B, C]
	// 2. if A in [A, B, C] => Circular reference!
	//
	// ALTERNATIVE VALIDATION:
	// 1. GetDescendants(A) => [B, C]
	// 2. if C in [B, C] => Circular reference!
	MoveToParent(ctx context.Context, categoryID uuid.UUID, req *MoveToParentReq) (*CategoryResp, error)

	// Activate kích hoạt category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: uuid.UUID
	//
	// RETURN:
	// - *CategoryResp
	// - error: ErrParentInactive nếu parent inactive
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID exists
	// 2. Get category: Repository.GetByID(categoryID)
	// 3. If already active => return early (idempotent)
	// 4. If has parent:
	//    a. Get parent: Repository.GetByID(parentID)
	//    b. if parent.is_active = false => ERROR (parent inactive)
	// 5. Repository.Activate(categoryID)
	// 6. Get updated category
	// 7. Map to response DTO
	// 8. Return response
	//
	// IDEMPOTENT:
	// Activate(active) => Activate(active) => Activate(active) = same result
	// Safe to call multiple times
	Activate(ctx context.Context, categoryID uuid.UUID) (*CategoryResp, error)

	// Deactivate vô hiệu hóa category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: uuid.UUID
	//
	// RETURN:
	// - *CategoryResp
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID exists
	// 2. Get category: Repository.GetByID(categoryID)
	// 3. If already inactive => return early (idempotent)
	// 4. Repository.Deactivate(categoryID)
	//    - This will cascade deactivate all descendants
	// 5. Get updated category
	// 6. Map to response DTO
	// 7. Return response
	//
	// SIDE EFFECTS:
	// - Deactivate(A) => A inactive, B inactive (children), C inactive (grandchildren)
	// - All descendants become inactive
	// - Books in all descendants become invisible
	//
	// TRANSACTION:
	// - Atomic: All update or none
	Deactivate(ctx context.Context, categoryID uuid.UUID) (*CategoryResp, error)

	// ========== DELETE OPERATIONS ==========

	// Delete xóa category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: uuid.UUID
	//
	// RETURN:
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID exists
	// 2. Get category: Repository.GetByID(categoryID)
	// 3. Check CanDelete():
	//    a. if HasChildren => ErrHasChildren
	//    b. if HasBooks => ErrHasBooks
	// 4. Repository.Delete(categoryID)
	// 5. Return nil (success)
	//
	// HARD DELETE:
	// - Category completely removed from DB
	// - No recovery possible
	// - Cannot have orphaned references (children, books)
	//
	// WHY HARD DELETE?
	// - Category tree nên clean, không "deleted" markers
	// - If has children/books => cannot delete anyway
	// - So hard delete is safe
	Delete(ctx context.Context, categoryID uuid.UUID) error

	// ========== BULK OPERATIONS ==========

	// BulkActivate kích hoạt nhiều categories
	//
	// PARAMS:
	// - ctx: Context
	// - req: *BulkCategoryIDsReq
	//
	// RETURN:
	// - *BulkActionResp: Success/failed counts
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate request (category_ids not empty)
	// 2. Validate all IDs exist
	// 3. Validate no circular issues (parent inactive)
	// 4. Repository.BulkActivate(ids)
	// 5. Return counts
	//
	// OPTIMIZATION:
	// - Single database query: UPDATE ... WHERE id = ANY(ids)
	// - Return count of updated rows
	BulkActivate(ctx context.Context, req *BulkCategoryIDsReq) (*BulkActionResp, error)

	// BulkDeactivate vô hiệu hóa nhiều categories
	//
	// PARAMS:
	// - ctx: Context
	// - req: *BulkCategoryIDsReq
	//
	// RETURN:
	// - *BulkActionResp
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate request
	// 2. Repository.BulkDeactivate(ids) => cascade inactive descendants
	// 3. Return counts
	//
	// SIDE EFFECTS:
	// - Deactivate (A, B, C) => A, B, C + their descendants all inactive
	// - Count includes both direct + descendants
	BulkDeactivate(ctx context.Context, req *BulkCategoryIDsReq) (*BulkActionResp, error)

	// BulkDelete xóa nhiều categories
	//
	// PARAMS:
	// - ctx: Context
	// - req: *BulkCategoryIDsReq
	//
	// RETURN:
	// - *BulkActionResp: success/failed with reasons
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate request
	// 2. For each ID:
	//    a. Get category
	//    b. Check CanDelete()
	//    c. if cannot delete => add to failed_items
	//    d. if can delete => add to delete_list
	// 3. If delete_list not empty:
	//    - Repository.BulkDelete(delete_list)
	// 4. Return BulkActionResp{success, failed, failed_items}
	//
	// PARTIALLY SUCCESSFUL:
	// - Some categories delete, some fail
	// - Return which ones failed + reason
	// - Example response:
	//   {
	//     "success": 48,
	//     "failed": 2,
	//     "failed_items": [
	//       {"id": "...", "reason": "has 5 children"},
	//       {"id": "...", "reason": "has 10 books"}
	//     ]
	//   }
	BulkDelete(ctx context.Context, req *BulkCategoryIDsReq) (*BulkActionResp, error)

	// ========== BOOK-RELATED OPERATIONS ==========

	// GetBooksInCategory lấy tất cả books trong category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: uuid.UUID
	// - limit, offset: Pagination
	//
	// RETURN:
	// - []uuid.UUID: Book IDs
	// - int64: Total count
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID exists
	// 2. Validate pagination
	// 3. Repository.GetBooksInCategory(categoryID, limit, offset)
	// 4. Return book IDs + count
	//
	// USE CASE:
	// GET /v1/categories/tieu-thuyet/books?limit=10&offset=0
	// => Service.GetBooksInCategory(categoryID, 10, 0)
	// => [book_id1, book_id2, ...], total=245
	GetBooksInCategory(ctx context.Context, categoryID uuid.UUID, limit int, page int) ([]model.BookResponse, int64, error)

	// GetCategoryBookCount lấy số books trong category
	//
	// PARAMS:
	// - ctx: Context
	// - categoryID: uuid.UUID
	//
	// RETURN:
	// - int64: Total books (including descendants)
	// - error
	//
	// BUSINESS LOGIC:
	// 1. Validate categoryID exists
	// 2. Repository.GetCategoryBookCount(categoryID)
	// 3. Return count
	//
	// USE CASE:
	// Display badge: "Tiểu thuyết (245 cuốn)"
	// GET /v1/categories/tieu-thuyet/book-count
	// => Service.GetCategoryBookCount(categoryID)
	// => 245
	GetCategoryBookCount(ctx context.Context, categoryID uuid.UUID) (int64, error)
}
