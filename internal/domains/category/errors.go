package category

import (
	"errors"
	"fmt"
)

// ============================================================
// 📚 KHÁI NIỆM: Error Handling Strategy
// ============================================================
// Error handling trong Go:
// 1. Explicit: func returns (value, error) => phải check error
// 2. Domain-specific errors: Định nghĩa errors riêng cho domain
// 3. Error wrapping: fmt.Errorf("%w", err) => preserve error chain
//
// LỢI ÍCH CUSTOM ERRORS:
// 1. Semantic: Error type cho biết lỗi từ domain nào
//    ❌ "database error" (không biết gì)
//    ✅ "ErrDuplicateSlug" (biết ngay slug duplicate)
//
// 2. Testable: Dễ check error type bằng errors.Is()
//    if errors.Is(err, ErrDuplicateSlug) { ... }
//
// 3. Mapping: Dễ map tới HTTP status code
//    ErrDuplicateSlug => 409 Conflict
//    ErrCategoryNotFound => 404 Not Found
//    ErrValidation => 400 Bad Request
//
// 4. Chain: fmt.Errorf("%w", err) preserve lỗi gốc
//    Error chain: handler => service => repository => database
//
// ERROR FLOW DIAGRAM:
// ┌─────────────────────────────────────┐
// │ Handler (HTTP Layer)                │
// │ return err => check type => map to HTTP status
// └────────────────┬────────────────────┘
//                  │ calls
//                  ▼
// ┌─────────────────────────────────────┐
// │ Service (Business Logic Layer)      │
// │ if errors.Is(err, ErrNotFound)      │
// │   return err (propagate)            │
// └────────────────┬────────────────────┘
//                  │ calls
//                  ▼
// ┌─────────────────────────────────────┐
// │ Repository (Data Access Layer)      │
// │ if rows.Err() != nil {              │
// │   return fmt.Errorf("%w", err)      │
// └─────────────────────────────────────┘

// ============================================================
// SENTINEL ERRORS (Error Variables)
// ============================================================
// Sentinel errors là error variables được define một lần
// Dùng errors.Is() để compare
//
// VÍ DỤ:
// var ErrNotFound = errors.New("not found")
// err := repo.GetByID(id)
// if errors.Is(err, ErrNotFound) { ... }
//
// LỢI ÍCH:
// - Type-safe: Không phải string matching
// - Performance: Error variable được cache
// - Idiomatic Go: Theo convention

// ErrCategoryNotFound xảy ra khi category không tìm thấy
//
// KỊCH BẢN:
// GET /v1/categories/invalid-id
// => Service.GetByID("invalid-id")
// => Repository.GetByID() => SELECT ... WHERE id = $1
// => No rows found
// => return ErrCategoryNotFound
// => Handler check: if errors.Is(err, ErrCategoryNotFound)
// => return HTTP 404
//
// DATABASE BEHAVIOR:
// rows, err := db.Query("SELECT ... WHERE id = $1", id)
//
//	if err != nil {
//	  return fmt.Errorf("failed to query: %w", err)
//	}
//
//	if !rows.Next() {
//	  return fmt.Errorf("category %w", ErrCategoryNotFound)
//	}
var ErrCategoryNotFound = fmt.Errorf("category not found")
var ErrInvalidCateID = errors.New("Bad request ! Invalid category id")

// ErrDuplicateSlug xảy ra khi slug đã tồn tại
//
// FLOW:
// POST /v1/categories
// Body: {name: "Tiểu Thuyết"}
// => GenerateSlug("Tiểu Thuyết") => "tieu-thuyet"
// => Service check: ExistsBySlug("tieu-thuyet")
// => Nếu exist => return ErrDuplicateSlug
//
// DATABASE CONSTRAINT:
// CREATE UNIQUE INDEX idx_categories_slug ON categories(slug)
// Nếu INSERT duplicate slug => DB error
// Repository catch => return ErrDuplicateSlug
//
// HTTP STATUS: 409 Conflict (hoặc 400 Bad Request)
var ErrDuplicateSlug = fmt.Errorf("category slug already exists")

// ErrInvalidCategoryName xảy ra khi name không hợp lệ
//
// RULES:
// - Không rỗng (after trim)
// - Không quá 255 chars
// - Không chỉ spaces
//
// VALIDATION LAYER:
// Entity.NewCategory() => check name
// if strings.TrimSpace(name) == "" => return ErrInvalidCategoryName
// if len(name) > 255 => return ErrInvalidCategoryName
//
// HTTP STATUS: 400 Bad Request
var ErrInvalidCategoryName = fmt.Errorf("invalid category name")

// ErrInvalidCategoryDescription xảy ra khi description quá dài
var ErrInvalidCategoryDescription = fmt.Errorf("invalid category description")

// ErrInvalidSortOrder xảy ra khi sort_order không hợp lệ
//
// RULES:
// - Phải >= 0
// - Phải <= 999
var ErrInvalidSortOrder = fmt.Errorf("invalid sort order")

// ErrParentNotFound xảy ra khi parent ID không tồn tại
//
// KỊCH BẢN:
// POST /v1/categories
// Body: {name: "Trinh thám", parent_id: "invalid-uuid"}
// => Service check: Repository.ExistsByID(parent_id)
// => Not found => return ErrParentNotFound
//
// DATABASE:
// SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)
// => false
// => return ErrParentNotFound
//
// HTTP STATUS: 400 Bad Request (invalid reference)
var ErrParentNotFound = fmt.Errorf("parent category not found")

// ErrCircularReference xảy ra khi cố set parent là descendant
//
// FLOW:
// Tree hiện tại:
// A (root)
//
//	└── B
//	    └── C
//
// PUT /v1/categories/A
// Body: {parent_id: C_ID}
// => MoveToParent(A, C)
// => Service check: GetAncestors(C) => [A, B, C]
// => if A in ancestors => ERROR (circular!)
//
// LỢI ÍCH DETECT:
// Nếu move A vào C, sẽ:
// C > A > B > C (cycle!)
// Category không thể find path từ root
//
// VALIDATION:
// 1. GetAncestors(newParent) => ancestors_list
// 2. if categoryID in ancestors_list => circular reference!
//
// HTTP STATUS: 400 Bad Request (invalid operation)
var ErrCircularReference = fmt.Errorf("circular reference: cannot move category to its descendant")

// ErrMaxDepthExceeded xảy ra khi vượt max depth (3 levels)
//
// CONSTRAINT:
// Max 3 levels: root (1) => child (2) => grandchild (3)
// Không được level 4 trở lên
//
// VALIDATION LOGIC:
// GET newParent by ID
// if newParent.level + 1 > MAX_DEPTH (3) => ERROR
//
// VÍ DỤ:
// Tree: A (level 1) > B (level 2) > C (level 3)
// CreateCategory(name="D", parent_id=C_ID)
// => C.level = 3
// => new level = 3 + 1 = 4 > 3 => ERROR
//
// LỢI ÍCH: Prevent deep nesting, keep tree manageable
//
// HTTP STATUS: 400 Bad Request
var ErrMaxDepthExceeded = fmt.Errorf("category depth exceeds maximum level of 3")

// ErrHasChildren xảy ra khi cố delete category mà nó có children
//
// FLOW:
// DELETE /v1/categories/{id}
// => Service.Delete(id)
// => Repository check: HasChildren(id)
// => if true => return ErrHasChildren
//
// LỢI ÍCH PREVENTION:
// Database constraint: FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE
// Nếu delete parent => children bị cascade delete
// Nhưng tốt hơn là check trước, return user-friendly error
//
// SOLUTION for user:
// - Delete children first
// - Move children to sibling
// - Or cascade delete (set ON DELETE CASCADE)
//
// HTTP STATUS: 409 Conflict
var ErrHasChildren = fmt.Errorf("cannot delete category that has children")

// ErrHasBooks xảy ra khi cố delete category mà nó có books
//
// FLOW:
// DELETE /v1/categories/{id}
// => Service.Delete(id)
// => Repository check: GetCategoryBookCount(id)
// => if count > 0 => return ErrHasBooks
//
// WHY PREVENT?
// Books reference category
// Nếu delete category => books orphaned (invalid state)
// (Hoặc book.category_id = NULL, nhưng business rule không cho)
//
// SOLUTION for user:
// - Move books to another category
// - Archive category (set is_active = false) instead
//
// HTTP STATUS: 409 Conflict
var ErrHasBooks = fmt.Errorf("cannot delete category that has books")

// ErrParentInactive xảy ra khi cố activate category nhưng parent inactive
//
// RULE:
// - Nếu parent inactive => category phải inactive
// - Không logic activate child khi parent ẩn
//
// FLOW:
// PUT /v1/categories/{id}/activate
// => Service.Activate(id)
// => Repository check: if parent.is_active = false
// => return ErrParentInactive
//
// REASON:
// Parent inactive => category không hiển thị anyway
// Vô ích activate child
// Nên activate parent trước, rồi activate child
//
// HTTP STATUS: 400 Bad Request
var ErrParentInactive = fmt.Errorf("cannot activate category while parent is inactive")

// ErrInvalidParentID xảy ra khi parent_id = category_id (self-reference)
//
// FLOW:
// POST /v1/categories
// Body: {name: "Test", parent_id: THIS_ID}
// => Service check: if categoryID == parentID
// => return ErrInvalidParentID
//
// DATABASE CONSTRAINT:
// Database trigger: CREATE TRIGGER prevent_self_parent
// WHEN (NEW.id = NEW.parent_id)
// RAISE EXCEPTION
//
// LỢI ÍCH DOUBLE CHECK:
// - Entity level: Validate ở NewCategory
// - Database level: Trigger prevent
// - Defense in depth
var ErrInvalidParentID = fmt.Errorf("category cannot be its own parent")

// ============================================================
// ERROR WRAPPERS (Contextual Errors)
// ============================================================
// Wrapper functions để thêm context vào error
// Dùng fmt.Errorf("%w", err) để preserve error chain
//
// LỢI ÍCH:
// - Error chain: Trace lỗi từ dưới lên
// - Stack: "failed to create: failed to query: connection refused"
// - Debugging: Biết lỗi xảy ra ở đâu

// NewValidationError tạo validation error với field + message
//
// USAGE:
//
//	if len(name) > 255 {
//	  return NewValidationError("name", "must not exceed 255 characters")
//	}
//
// RESULT:
// "validation error: field 'name' - must not exceed 255 characters"
func NewValidationError(field, message string) error {
	return fmt.Errorf("validation error: field '%s' - %s", field, message)
}

// NewConflictError tạo conflict error (resource conflict)
//
// USAGE:
//
//	if slugExists {
//	  return NewConflictError(ErrDuplicateSlug, fmt.Sprintf("slug '%s' already exists", slug))
//	}
//
// RESULT:
// "category slug already exists: slug 'tieu-thuyet' already exists"
func NewConflictError(err error, message string) error {
	return fmt.Errorf("%w: %s", err, message)
}

// NewNotFoundError tạo not found error với context
//
// USAGE:
// _, err := repo.GetByID(ctx, id)
//
//	if err != nil {
//	  return NewNotFoundError("Category", id.String())
//	}
//
// RESULT:
// "category not found: Category with ID 123e4567-e89b-12d3-a456-426614174000"
func NewNotFoundError(resource string, id string) error {
	return fmt.Errorf("%w: %s with ID %s", ErrCategoryNotFound, resource, id)
}

// NewConstraintError tạo constraint error (business rule violation)
//
// USAGE:
//
//	if hasChildren {
//	  return NewConstraintError(ErrHasChildren, "move children to sibling first")
//	}
//
// RESULT:
// "cannot delete category that has children: move children to sibling first"
func NewConstraintError(err error, suggestion string) error {
	return fmt.Errorf("%w (%s)", err, suggestion)
}

// ============================================================
// ERROR CHECKING UTILITIES
// ============================================================
// Utility functions để check error type
// Dùng errors.Is() là cách idiomatic
//
// EXAMPLE:
// _, err := service.GetByID(ctx, id)
// if errors.Is(err, ErrCategoryNotFound) {
//   return nil, http.StatusNotFound, "Category not found"
// }
//
// ALTERNATIVE (less idiomatic):
// if err != nil && strings.Contains(err.Error(), "not found") { ... } ❌ BAD
// if errors.Is(err, ErrCategoryNotFound) { ... } ✅ GOOD

// IsNotFound kiểm tra xem error có phải not found không
//
// USAGE:
// err := repo.GetByID(ctx, id)
//
//	if IsNotFound(err) {
//	  // Handle not found case
//	}
func IsNotFound(err error) bool {
	// Dùng error chain: fmt.Errorf("%w", err) preserve error
	// errors.Is() check throughout chain
	// Ví dụ: if wrapped err "%w" not found, Is() vẫn find nó
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrCategoryNotFound)
}

// IsDuplicateSlug kiểm tra xem error có phải duplicate slug
func IsDuplicateSlug(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrDuplicateSlug)
}

// IsCircularReference kiểm tra circular reference
func IsCircularReference(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrCircularReference)
}

// IsMaxDepthExceeded kiểm tra vượt max depth
func IsMaxDepthExceeded(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrMaxDepthExceeded)
}

// IsHasChildren kiểm tra category có children
func IsHasChildren(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrHasChildren)
}

// IsHasBooks kiểm tra category có books
func IsHasBooks(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrHasBooks)
}

// IsParentInactive kiểm tra parent inactive
func IsParentInactive(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint(ErrParentInactive)
}

// IsValidationError kiểm tra xem error có phải validation không
//
// USAGE:
// err := service.Create(ctx, req)
//
//	if IsValidationError(err) {
//	  return nil, http.StatusBadRequest, err.Error()
//	}
func IsValidationError(err error) bool {
	return err != nil && fmt.Sprint(err) == fmt.Sprint("validation error")
}

// ============================================================
// ERROR CODE MAPPING (For HTTP Responses)
// ============================================================
// GetHTTPStatusCode map domain error tới HTTP status code
//
// KHÁI NIỆM - Status Code Mapping:
// Domain error => HTTP status code
// ErrCategoryNotFound => 404 Not Found
// ErrDuplicateSlug => 409 Conflict
// ErrCircularReference => 400 Bad Request
//
// LỢI ÍCH:
// - Centralized: 1 chỗ để map (dễ thay đổi)
// - Readable: Code rõ ràng
// - Maintainable: Dễ add error sau này
//
// FLOW:
// handler.go:
// _, err := service.Delete(ctx, id)
//
//	if err != nil {
//	  status := GetHTTPStatusCode(err)
//	  return c.JSON(status, ErrorResponse{Message: err.Error()})
//	}
//
// USAGE:
// status := GetHTTPStatusCode(ErrDuplicateSlug)
// => 409
//
// status := GetHTTPStatusCode(fmt.Errorf("wrapped: %w", ErrNotFound))
// => 404
func GetHTTPStatusCode(err error) int {
	const (
		statusBadRequest  = 400
		statusConflict    = 409
		statusNotFound    = 404
		statusServerError = 500
	)

	// Check error type (wrapped hoặc not)
	switch {
	case fmt.Sprint(err) == fmt.Sprint(ErrCategoryNotFound):
		return statusNotFound
	case fmt.Sprint(err) == fmt.Sprint(ErrDuplicateSlug):
		return statusConflict
	case fmt.Sprint(err) == fmt.Sprint(ErrCircularReference):
		return statusBadRequest
	case fmt.Sprint(err) == fmt.Sprint(ErrMaxDepthExceeded):
		return statusBadRequest
	case fmt.Sprint(err) == fmt.Sprint(ErrHasChildren):
		return statusConflict
	case fmt.Sprint(err) == fmt.Sprint(ErrHasBooks):
		return statusConflict
	case fmt.Sprint(err) == fmt.Sprint(ErrParentNotFound):
		return statusBadRequest
	case fmt.Sprint(err) == fmt.Sprint(ErrParentInactive):
		return statusBadRequest
	case fmt.Sprint(err) == fmt.Sprint(ErrInvalidCategoryName):
		return statusBadRequest
	case fmt.Sprint(err) == fmt.Sprint(ErrInvalidSortOrder):
		return statusBadRequest
	case fmt.Sprint(err) == fmt.Sprint(ErrInvalidParentID):
		return statusBadRequest
	case IsValidationError(err):
		return statusBadRequest
	default:
		return statusServerError // Unknown error
	}
}

// GetErrorMessage trả về user-friendly error message
//
// KHÁI NIỆM - Error Message:
// Internal error: "sql: database closed"
// User-friendly: "Service temporarily unavailable"
//
// LỢI ÍCH:
// - Security: Không leak internal details
// - UX: User hiểu được message
// - Consistency: Tất cả error message cùng format
//
// FLOW:
// handler.go:
// _, err := service.Create(ctx, req)
//
//	if err != nil {
//	  message := GetErrorMessage(err)
//	  return c.JSON(GetHTTPStatusCode(err), ErrorResponse{Message: message})
//	}
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errStr := fmt.Sprint(err)

	switch {
	case fmt.Sprint(err) == fmt.Sprint(ErrCategoryNotFound):
		return "Category not found"
	case fmt.Sprint(err) == fmt.Sprint(ErrDuplicateSlug):
		return "Category slug already exists. Please use a different name."
	case fmt.Sprint(err) == fmt.Sprint(ErrCircularReference):
		return "Cannot move category to its own descendant"
	case fmt.Sprint(err) == fmt.Sprint(ErrMaxDepthExceeded):
		return "Category tree depth exceeds maximum of 3 levels"
	case fmt.Sprint(err) == fmt.Sprint(ErrHasChildren):
		return "Cannot delete category that has subcategories. Please move or delete them first."
	case fmt.Sprint(err) == fmt.Sprint(ErrHasBooks):
		return "Cannot delete category that has books. Please move books first."
	case fmt.Sprint(err) == fmt.Sprint(ErrParentNotFound):
		return "Parent category not found"
	case fmt.Sprint(err) == fmt.Sprint(ErrParentInactive):
		return "Cannot activate category while parent is inactive"
	case fmt.Sprint(err) == fmt.Sprint(ErrInvalidCategoryName):
		return "Category name is invalid"
	case IsValidationError(err):
		return errStr // Return full message for validation (includes field name)
	default:
		return "Internal server error"
	}
}
