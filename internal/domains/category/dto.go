package category

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// REQUEST DTOs (Input Data)
// ============================================================

// CreateCategoryReq là request body khi POST /v1/categories
//
// VALIDATION RULES:
// - Name: required, 1-255 chars
// - ParentID: optional UUID
// - Description: optional, max 1000 chars
// - IconURL: optional URL
// - SortOrder: optional, default 0, range 0-999
//
// FLOW:
// POST /v1/categories
//
//	Body: {
//	  "name": "Tiểu Thuyết",
//	  "parent_id": "550e8400-e29b-41d4-a716-446655440000",
//	  "description": "Sách tiểu thuyết hay",
//	  "icon_url": "https://...",
//	  "sort_order": 1
//	}
type CreateCategoryReq struct {
	Name string `json:"name" binding:"required"`

	// ParentID: UUID của category cha
	ParentID *uuid.UUID `json:"parent_id" binding:"omitempty"`

	// Description: Mô tả category
	// Constraint: optional, max 1000 chars
	Description string `json:"description" binding:"omitempty,max=1000"`

	// IconURL: Link icon
	// Constraint: optional, URL format
	IconURL string `json:"icon_url" binding:"omitempty,url"`

	// SortOrder: Thứ tự sắp xếp
	// Constraint: optional, default 0, range 0-999
	// Note: JSON omitempty, vậy nếu không provide => 0
	SortOrder int `json:"sort_order" binding:"omitempty,gte=0,lte=999"`
}

// UpdateCategoryReq là request body khi PUT /v1/categories/{id}

// PUT /v1/categories/123
//	Body: {
//	  "name": "Tiểu Thuyết Updated",
//	  "description": "..."
//	}

// NOTE - Partial Update:
// Nếu "name" không provide => không update "name"
// Implement cách:
// - Dùng *string (pointer): nil = không update
// - Hoặc check JSON field
// - Hoặc check pointer != nil
type UpdateCategoryReq struct {
	// Name: Tên mới (optional)
	Name *string `json:"name" binding:"omitempty"`

	// Description: Mô tả mới (optional)
	Description *string `json:"description" binding:"omitempty"`

	// IconURL: Icon mới (optional)
	IconURL *string `json:"icon_url" binding:"omitempty"`

	// SortOrder: Thứ tự mới (optional)
	SortOrder *int `json:"sort_order" binding:"omitempty"`
}

// MoveToParentReq là request body khi PATCH /v1/categories/{id}/parent
//
// PURPOSE:
// Di chuyển category tới parent khác
// Separate endpoint vì:
// - Cần validate circular reference
// - Cần validate max depth
// - Cần update parent_id riêng (bảo vệ khỏi accidental change)
//
// FLOW:
// PATCH /v1/categories/trinh-tham/parent
//
//	Body: {
//	  "parent_id": "tieu-thuyet-id"
//	}
//
// Service:
// 1. MoveToParent(categoryID, parentID)
// 2. Validate circular reference
// 3. Validate max depth
// 4. UPDATE parent_id
type MoveToParentReq struct {
	// ParentID: UUID của parent mới
	// nil = move to root
	// UUID = move to parent này
	ParentID *uuid.UUID `json:"parent_id" binding:"omitempty"`
}

// BulkCategoryIDsReq là request body khi POST /v1/categories/bulk/activate
//
// FLOW:
// POST /v1/categories/bulk/activate
//
//	Body: {
//	  "category_ids": ["id1", "id2", "id3"]
//	}
//
// Handler:
// 1. c.BindJSON(&req)
// 2. service.BulkActivate(req.CategoryIDs)
// 3. Return count updated
//
// OPTIMIZATION:
// - Bulk operation: 1 database query (vs N queries)
// - ANY clause: UPDATE ... WHERE id = ANY($1::uuid[])
type BulkCategoryIDsReq struct {
	// CategoryIDs: Danh sách category IDs
	// Constraint: required, not empty, valid UUIDs
	CategoryIDs []uuid.UUID `json:"category_ids" binding:"required,min=1"`
}

// ============================================================
// RESPONSE DTOs (Output Data)
// ============================================================
type CategoryResp struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	ParentID      *uuid.UUID `json:"parent_id,omitempty"`
	Level         int        `json:"level"`
	SortOrder     int        `json:"sort_order"`
	ChildrenCount int        `json:"children_count,omitempty"`
	BooksCount    int64      `json:"books_count"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CategoryTreeItemResp là response item cho tree API
//
// CONTENTS:
// - ID, Name, Slug: Basic info
// - Level: Depth
// - FullPath: Breadcrumb tên
// - ChildrenCount: Số children
// - IsActive: Status
//
// USAGE:
// GET /v1/categories/tree
//
//	Response: {
//	  "categories": [
//	    {
//	      "id": "...",
//	      "name": "Văn học",
//	      "slug": "van-hoc",
//	      "level": 1,
//	      "full_path": "Văn học",
//	      "children_count": 2,
//	      "is_active": true
//	    },
//	    {
//	      "id": "...",
//	      "name": "Tiểu thuyết",
//	      "slug": "tieu-thuyet",
//	      "level": 2,
//	      "full_path": "Văn học > Tiểu thuyết",
//	      "children_count": 2,
//	      "is_active": true
//	    },
//	    ...
//	  ]
//	}
//
// TREE STRUCTURE:
// Không nested JSON (flat array)
// Client render tree dựa vào level + full_path
// LỢI ÍCH:
// - Simpler JSON (không recursive)
// - Easier to flatten/paginate
// - Mobile-friendly (flat data structure)
type CategoryTreeItemResp struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Level         int       `json:"level"`
	FullPath      string    `json:"full_path"`
	ChildrenCount int       `json:"children_count"`
	IsActive      bool      `json:"is_active"`
}

// CategoryBreadcrumbResp là response cho breadcrumb API
//
// CONTENTS:
// - Items: Array của breadcrumb items
// - CurrentPath: Full path string
//
// USAGE:
// GET /v1/categories/trinh-tham/breadcrumb
//
//	Response: {
//	  "items": [
//	    {"id": "...", "name": "Văn học", "slug": "van-hoc"},
//	    {"id": "...", "name": "Tiểu thuyết", "slug": "tieu-thuyet"},
//	    {"id": "...", "name": "Trinh thám", "slug": "trinh-tham"}
//	  ],
//	  "current_path": "Văn học > Tiểu thuyết > Trinh thám"
//	}
//
// UI USAGE:
// Breadcrumb: Home > Văn học > Tiểu thuyết > Trinh thám
// Clickable links: mỗi item là link
type CategoryBreadcrumbResp struct {
	Items       []BreadcrumbItem `json:"items"`
	CurrentPath string           `json:"current_path"`
}

// BreadcrumbItem là 1 item trong breadcrumb
//
// CONTENTS:
// - ID: Category ID
// - Name: Display name
// - Slug: URL slug (dùng làm link)
type BreadcrumbItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// CategoryListResp là response cho list API (pagination)
//
// CONTENTS:
// - Categories: Array category items
// - Total: Total count (dùng để tính page count)
// - Limit: Items per page
// - Offset: Current offset
// - HasMore: Có more data không
//
// USAGE:
// GET /v1/categories?limit=10&offset=0
//
//	Response: {
//	  "categories": [...],
//	  "total": 45,
//	  "limit": 10,
//	  "offset": 0,
//	  "has_more": true
//	}
//
// CLIENT PAGINATION:
// Total items: 45
// Current page: 1 (offset 0)
// Total pages: ceil(45 / 10) = 5
// Has next page: offset + limit < total
type CategoryListResp struct {
	Categories []CategoryResp `json:"categories"`
	Total      int64          `json:"total"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	HasMore    bool           `json:"has_more"`
}

// BulkActionResp là response cho bulk operations
//
// CONTENTS:
// - Success: Số items thành công
// - Failed: Số items fail
// - FailedItems: List items fail (optional)
//
// USAGE:
// POST /v1/categories/bulk/activate
//
//	Response: {
//	  "success": 50,
//	  "failed": 0,
//	  "failed_items": []
//	}
//
// ERROR CASE:
// POST /v1/categories/bulk/delete
//
//	Response: {
//	  "success": 48,
//	  "failed": 2,
//	  "failed_items": [
//	    {"id": "...", "reason": "has children"},
//	    {"id": "...", "reason": "has books"}
//	  ]
//	}
type BulkActionResp struct {
	Success     int64                  `json:"success"`
	Failed      int64                  `json:"failed"`
	FailedItems []BulkActionFailedItem `json:"failed_items,omitempty"`
}

// BulkActionFailedItem là 1 item fail trong bulk operation
type BulkActionFailedItem struct {
	ID     uuid.UUID `json:"id"`
	Reason string    `json:"reason"`
}

// ============================================================
// MAPPER FUNCTIONS (Entity <-> DTO)
// ============================================================

// CategoryToResp converts Category entity to CategoryResp DTO
//
// PURPOSE:
// Entity không trực tiếp expose ở API
// Map tới DTO trước
// LỢI ÍCH: Decoupling, control output
//
// FLOW:
// Service return *Category
// Handler call CategoryToResp()
// Return CategoryResp
// Marshal to JSON
//
// FIELDS MAPPING:
// Entity.Level *int => DTO.Level int (dereference pointer)
// Entity.ChildCount *int => DTO.ChildrenCount int
// Entity.TotalBooksCount *int64 => DTO.BooksCount int64
func CategoryToResp(c *Category) *CategoryResp {
	if c == nil {
		return nil
	}

	resp := &CategoryResp{
		ID:         c.ID,
		Name:       c.Name,
		Slug:       c.Slug,
		ParentID:   c.ParentID,
		BooksCount: c.BooksCount,
		SortOrder:  c.SortOrder,
		IsActive:   c.IsActive,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}

	// ========== ADD CHILDREN_COUNT ==========
	if c.Level != nil {
		resp.Level = *c.Level
	}

	if c.ChildCount != nil {
		resp.ChildrenCount = *c.ChildCount // ← ADD THIS
	}

	return resp
}

// CategoryToTreeItem converts Category to tree item
func CategoryToTreeItem(c *Category) *CategoryTreeItemResp {
	if c == nil {
		return nil
	}

	level := 1
	if c.Level != nil {
		level = *c.Level
	}

	// ========== ADD CHILDREN_COUNT ==========
	childrenCount := 0
	if c.ChildCount != nil {
		childrenCount = *c.ChildCount // ← ADD THIS
	}

	fullPath := c.Name
	if c.FullPath != nil {
		fullPath = *c.FullPath
	}

	return &CategoryTreeItemResp{
		ID:            c.ID,
		Name:          c.Name,
		Slug:          c.Slug,
		Level:         level,
		FullPath:      fullPath,
		ChildrenCount: childrenCount, // ← NOW MAPPED
		IsActive:      c.IsActive,
	}
}

// CategoriesToResp converts []Category to []CategoryResp
func CategoriesToResp(categories []Category) []CategoryResp {
	if len(categories) == 0 {
		return []CategoryResp{}
	}

	resps := make([]CategoryResp, 0, len(categories))
	for _, c := range categories {
		resps = append(resps, *CategoryToResp(&c))
	}
	return resps
}

// CategoriesToTreeItems converts []Category to []CategoryTreeItemResp
func CategoriesToTreeItems(categories []Category) []CategoryTreeItemResp {
	if len(categories) == 0 {
		return []CategoryTreeItemResp{}
	}

	items := make([]CategoryTreeItemResp, 0, len(categories))
	for _, c := range categories {
		items = append(items, *CategoryToTreeItem(&c))
	}
	return items
}
