package service

import (
	"context"
	"fmt"
	"strings"

	"bookstore-backend/internal/domains/category"
	"bookstore-backend/pkg/logger"

	"github.com/google/uuid"
)

type categoryServiceImpl struct {
	repository category.CategoryRepository
}

func NewCategoryService(repo category.CategoryRepository) category.CategoryService {
	return &categoryServiceImpl{
		repository: repo,
	}
}

func (s *categoryServiceImpl) Create(ctx context.Context, req *category.CreateCategoryReq) (*category.CategoryResp, error) {
	// ========== STEP 1: Validate Input ==========
	// Safety check: req should be validated by handler
	// But defensive programming: check anyway
	if req == nil {
		logger.Info("Create: request is nil", map[string]interface{}{
			"create error": nil,
		})
		return nil, fmt.Errorf("create category: invalid request")
	}

	// ========== STEP 2: Create Entity with Validation ==========
	// NewCategory does entity-level validation
	// - Check name not empty, length valid
	// - Check description length
	// - Check sortOrder range
	// - Generate slug
	entity, err := category.NewCategory(
		req.Name,
		req.ParentID,
		req.Description,
		req.IconURL,
		req.SortOrder,
	)
	if err != nil {
		// Log the error for debugging
		// Don't expose internal error to client
		logger.Info("NewCategory", map[string]interface{}{
			"error": fmt.Sprintf("Create: entity validation failed: %v", err),
		})
		// Return domain-specific error
		return nil, fmt.Errorf("create category: %w", err)
	}

	// ========== STEP 3: Check Parent Exists (if provided) ==========
	// If ParentID is provided, must verify it exists in DB
	// Otherwise: ERR_PARENT_NOT_FOUND
	if entity.ParentID != nil {
		exists, err := s.repository.ExistsByID(ctx, *entity.ParentID)
		if err != nil {
			logger.Info("category ExistsByID", map[string]interface{}{
				"error": fmt.Sprintf("Create: check parent exists failed: %v", err),
			})
			return nil, fmt.Errorf("create category: failed to verify parent")
		}

		if !exists {
			// Parent not found => return specific error
			logger.Info("Error Parent Not Found", map[string]interface{}{
				"error": fmt.Sprintf("Create: parent not found: %s", entity.ParentID.String()),
			})
			return nil, category.ErrParentNotFound
		}

		// ========== STEP 4: Check Max Depth ==========
		// Get parent to check level
		// If parent.level + 1 > MAX_DEPTH (3) => ERROR
		//
		// WHY CHECK DEPTH?
		// Max 3 levels: root (1) > child (2) > grandchild (3)
		// Prevent deep nesting, keep tree manageable
		parent, err := s.repository.GetByID(ctx, *entity.ParentID)
		if err != nil {
			logger.Info("Verify Parent", map[string]interface{}{
				"error": fmt.Sprintf("Create: failed to get parent details: %v", err),
			})
			return nil, fmt.Errorf("create category: failed to verify parent")
		}

		// Check depth
		const MAX_DEPTH = 3
		parentLevel := parent.GetLevel()
		newLevel := parentLevel + 1
		if newLevel > MAX_DEPTH {
			logger.Info("newLevel", map[string]interface{}{
				"error": fmt.Sprintf("Create: max depth exceeded, new level would be: %d", newLevel),
			})
			return nil, category.ErrMaxDepthExceeded
		}

		// Set level for new entity (for response)
		entity.Level = &newLevel
	} else {
		// Root category => level 1
		level := 1
		entity.Level = &level
	}

	// ========== STEP 5: Check Slug Unique ==========
	// GenerateSlug already called in NewCategory
	// Now verify slug not duplicate
	// excludeID = nil (new category, no existing ID)
	slugExists, err := s.repository.ExistsBySlug(ctx, entity.Slug, nil)
	if err != nil {
		logger.Info("slugExists", map[string]interface{}{
			"error": fmt.Sprintf("Create: check slug exists failed: %v", err),
		})
		return nil, fmt.Errorf("create category: failed to verify slug uniqueness")
	}

	if slugExists {
		logger.Info("slugExists", map[string]interface{}{
			"error": fmt.Sprintf("Create: slug already exists: %s", entity.Slug),
		})
		return nil, category.ErrDuplicateSlug
	}

	// ========== STEP 6: Save to Repository ==========
	// Repository.Create() handles:
	// - INSERT statement
	// - Generate ID (PostgreSQL)
	// - Set timestamps
	// - Return with ID
	created, err := s.repository.Create(ctx, entity)
	if err != nil {
		logger.Info("create category", map[string]interface{}{
			"error": fmt.Sprintf("Create: repository create failed: %v", err),
		})
		// Check if it's a known domain error
		if category.IsDuplicateSlug(err) {
			return nil, category.ErrDuplicateSlug
		}
		// Otherwise return generic error
		return nil, fmt.Errorf("create category: failed to save")
	}

	// ========== STEP 7: Map to Response DTO ==========
	// Convert entity to DTO for HTTP response
	resp := category.CategoryToResp(created)

	logger.Info("create category", map[string]interface{}{
		"created.ID": fmt.Sprintf("Create: category created successfully: %s", created.ID.String()),
	})
	return resp, nil
}

// ========== READ: GetByID ==========
func (s *categoryServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*category.CategoryResp, error) {
	// ========== Validate Input ==========
	// UUID should never be nil (it's not pointer type)
	// But check for zero UUID (default value)
	if id == uuid.Nil {
		return nil, fmt.Errorf("get category: invalid id")
	}

	// ========== Fetch from Repository ==========
	// Repository.GetByID() handles:
	// - SELECT query
	// - Scan into entity
	// - Return error if not found
	entity, err := s.repository.GetByID(ctx, id)
	if err != nil {
		logger.Info("GetByID Failed", map[string]interface{}{
			"error": fmt.Sprintf("GetByID: repository get failed: %v", err),
		})
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category: failed to fetch")
	}

	// ========== Map to Response DTO ==========
	resp := category.CategoryToResp(entity)
	return resp, nil
}

// ========== READ: GetBySlug ==========
func (s *categoryServiceImpl) GetBySlug(ctx context.Context, slug string) (*category.CategoryResp, error) {
	// ========== Validate Input ==========
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("get category: invalid slug")
	}

	// ========== Fetch from Repository ==========
	entity, err := s.repository.GetBySlug(ctx, slug)
	if err != nil {
		logger.Info("GetBySlug", map[string]interface{}{
			"error": fmt.Sprintf("GetBySlug: repository get failed: %v", err),
		})
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category: failed to fetch")
	}

	// ========== Map to Response DTO ==========
	resp := category.CategoryToResp(entity)
	return resp, nil
}

// ========== READ: GetAll ==========
func (s *categoryServiceImpl) GetAll(
	ctx context.Context,
	isActive *bool,
	parentID *uuid.UUID,
	limit int,
	offset int,
) (*category.CategoryListResp, error) {
	// ========== Validate Pagination ==========
	// Limit: must be > 0, typically max 100
	if limit <= 0 || limit > 100 {
		limit = 10 // Default
	}

	// Offset: must be >= 0
	if offset < 0 {
		offset = 0
	}

	// ========== Build Filter ==========
	// CategoryFilter holds all query conditions
	filter := &category.CategoryFilter{
		IsActive:        isActive,
		ParentID:        parentID,
		IncludeInactive: false, // Default: only active
		Limit:           limit,
		Offset:          offset,
	}

	// ========== Fetch from Repository ==========
	// Returns both list + total count (for pagination calculation)
	entities, total, err := s.repository.GetAll(ctx, filter)
	if err != nil {
		logger.Info("GetAll failed", map[string]interface{}{
			"error": fmt.Sprintf("GetAll: repository get failed: %v", err),
		})
		return nil, fmt.Errorf("get categories: failed to fetch")
	}

	// ========== Map to Response DTOs ==========
	resps := category.CategoriesToResp(entities)

	// ========== Calculate Pagination Info ==========
	// hasMore: if there are more items after current page
	// Example: offset=0, limit=10, total=25
	// hasMore = 0+10 < 25 = true (page 1 of 3)
	//
	// Example: offset=20, limit=10, total=25
	// hasMore = 20+10 < 25 = false (page 3 of 3, no next page)
	hasMore := (offset + limit) < int(total)

	// ========== Build Response ==========
	resp := &category.CategoryListResp{
		Categories: resps,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		HasMore:    hasMore,
	}

	return resp, nil
}

// ========== READ: GetTree ==========
func (s *categoryServiceImpl) GetTree(ctx context.Context) ([]category.CategoryTreeItemResp, error) {
	// ========== Fetch from Repository ==========
	// Repository.GetTree() uses Materialized View
	// Already pre-computed, just SELECT
	entities, err := s.repository.GetTree(ctx)
	if err != nil {
		logger.Info("GetTree failed", map[string]interface{}{
			"error": fmt.Sprintf("GetTree: repository get failed: %v", err),
		})
		return nil, fmt.Errorf("get category tree: failed to fetch")
	}

	// ========== Map to Response DTOs ==========
	// Returns tree items (not full category response)
	// Includes level, full_path for breadcrumb
	resps := category.CategoriesToTreeItems(entities)

	return resps, nil
}

// ========== READ: GetBreadcrumb ==========
func (s *categoryServiceImpl) GetBreadcrumb(ctx context.Context, categoryID uuid.UUID) (*category.CategoryBreadcrumbResp, error) {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return nil, fmt.Errorf("get breadcrumb: invalid category id")
	}

	// ========== Fetch Ancestors ==========
	// Repository.GetCategoryBreadcrumb() returns ancestors from root to current
	// Example: [Văn học, Tiểu thuyết, Trinh thám]
	entities, err := s.repository.GetCategoryBreadcrumb(ctx, categoryID)
	if err != nil {
		logger.Info("GetCategoryBreadcrumb Failed", map[string]interface{}{
			"error": fmt.Sprintf("GetBreadcrumb: repository get failed: %v", err),
		})
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get breadcrumb: failed to fetch")
	}

	// ========== Build Response ==========
	items := make([]category.BreadcrumbItem, 0, len(entities))
	var fullPath strings.Builder

	for i, entity := range entities {
		item := category.BreadcrumbItem{
			ID:   entity.ID,
			Name: entity.Name,
			Slug: entity.Slug,
		}
		items = append(items, item)

		// Build full path
		if i > 0 {
			fullPath.WriteString(" > ")
		}
		fullPath.WriteString(entity.Name)
	}

	resp := &category.CategoryBreadcrumbResp{
		Items:       items,
		CurrentPath: fullPath.String(),
	}
	return resp, nil
}

// ========== UPDATE: Update ==========
func (s *categoryServiceImpl) Update(
	ctx context.Context,
	id uuid.UUID,
	req *category.UpdateCategoryReq,
) (*category.CategoryResp, error) {
	// ========== Validate Input ==========
	if id == uuid.Nil {
		return nil, fmt.Errorf("update category: invalid id")
	}

	if req == nil {
		return nil, fmt.Errorf("update category: invalid request")
	}

	// ========== Check Category Exists ==========
	// Must exist before update
	entity, err := s.repository.GetByID(ctx, id)
	if err != nil {
		logger.Info("GetByID failed", map[string]interface{}{
			"error": fmt.Sprintf("Update: repository get failed: %v", err),
		})
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("update category: failed to fetch")
	}

	// ========== Apply Updates (Partial Update) ==========
	// KHÁI NIỆM - Partial Update:
	// Only update provided fields
	// Use pointer: nil = not provided, value = update to value
	//
	// FLOW:
	// Current: {Name: "Tiểu thuyết", Desc: "...", Icon: "...", Order: 0}
	// Request: {Name: "Tiểu thuyết mới", Desc: nil, Icon: nil, Order: nil}
	// Result: {Name: "Tiểu thuyết mới", Desc: "...", Icon: "...", Order: 0}
	//
	// IMPLEMENTATION:
	// if req.Name != nil => update name
	// if req.Description != nil => update description
	// etc.

	// Build update data
	// Use current values as defaults, override with request values
	name := entity.Name
	if req.Name != nil {
		name = *req.Name
	}

	description := entity.Description
	if req.Description != nil {
		description = *req.Description
	}

	iconURL := entity.IconURL
	if req.IconURL != nil {
		iconURL = *req.IconURL
	}

	sortOrder := entity.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	// ========== Update Entity ==========
	// entity.Update() does:
	// - Validate new values
	// - Generate new slug (if name changed)
	// - Update UpdatedAt
	err = entity.Update(name, description, iconURL, sortOrder)
	if err != nil {
		logger.Info("Update failed", map[string]interface{}{
			"error": fmt.Sprintf("Update: entity update failed: %v", err),
		})
		return nil, fmt.Errorf("update category: %w", err)
	}

	// ========== Check Slug Unique (if Name Changed) ==========
	// If name changed => slug changed
	// Must verify new slug not duplicate
	if req.Name != nil && *req.Name != "" {
		slugExists, err := s.repository.ExistsBySlug(ctx, entity.Slug, &id)
		if err != nil {
			logger.Info("ExistsBySlug", map[string]interface{}{
				"error": fmt.Sprintf("Update: check slug exists failed: %v", err),
			})
			return nil, fmt.Errorf("update category: failed to verify slug")
		}

		if slugExists {
			logger.Info("slugExists", map[string]interface{}{
				"error": fmt.Sprintf("Update: slug already exists: %s", entity.Slug),
			})
			return nil, category.ErrDuplicateSlug
		}
	}

	// ========== Save to Repository ==========
	updated, err := s.repository.Update(ctx, entity)
	if err != nil {
		logger.Info("Update failed", map[string]interface{}{
			"error": fmt.Sprintf("Update: repository update failed: %v", err),
		})
		return nil, fmt.Errorf("update category: failed to save")
	}

	// ========== Map to Response DTO ==========
	resp := category.CategoryToResp(updated)

	return resp, nil
}

// ========== UPDATE: MoveToParent ==========
func (s *categoryServiceImpl) MoveToParent(
	ctx context.Context,
	categoryID uuid.UUID,
	req *category.MoveToParentReq,
) (*category.CategoryResp, error) {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return nil, fmt.Errorf("move to parent: invalid category id")
	}

	if req == nil {
		return nil, fmt.Errorf("move to parent: invalid request")
	}

	// ========== Check Category Exists ==========
	_, err := s.repository.GetByID(ctx, categoryID)
	if err != nil {
		logger.Info("MoveToParent", map[string]interface{}{
			"error": fmt.Sprintf(": repository get category failed: %v", err),
		})
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("move to parent: failed to fetch category")
	}

	// ========== Validate New Parent ==========
	// If newParentID provided, must be different from current parentID
	if req.ParentID != nil {
		// Check: newParentID != categoryID
		if *req.ParentID == categoryID {
			return nil, category.ErrInvalidParentID
		}

		// Check: newParent exists
		exists, err := s.repository.ExistsByID(ctx, *req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("move to parent: failed to verify parent")
		}

		if !exists {
			return nil, category.ErrParentNotFound
		}

		// ========== CIRCULAR REFERENCE CHECK ==========
		// CONCEPT:
		// Tree: A (level 1)
		//   └── B (level 2)
		//       └── C (level 3)
		//
		// If MoveToParent(A, C):
		// Result would be: C > A > B > C (cycle!)
		//
		// VALIDATION STRATEGY:
		// 1. Get ancestors of newParent
		// 2. If categoryID in ancestors => ERROR (circular!)
		//
		// EXAMPLE:
		// GetAncestors(C) => [A, B, C]
		// if A in [A, B, C] => Circular reference!
		//
		// WHY THIS WORKS?
		// If C's ancestor list includes A, moving A to C would create cycle
		// Because: A > ... > C and C > A => cycle!

		ancestors, err := s.repository.GetAncestors(ctx, *req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("move to parent: failed to verify tree structure")
		}

		// Check if categoryID is in ancestors list
		for _, ancestor := range ancestors {
			if ancestor.ID == categoryID {
				return nil, category.ErrCircularReference
			}
		}

		// ========== MAX DEPTH CHECK ==========
		// Get new parent to check level
		parent, err := s.repository.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("move to parent: failed to verify parent")
		}

		// Check: parent.level + 1 <= MAX_DEPTH
		const MAX_DEPTH = 3
		parentLevel := parent.GetLevel()
		newLevel := parentLevel + 1
		if newLevel > MAX_DEPTH {
			return nil, category.ErrMaxDepthExceeded
		}
	}

	// ========== Update Parent ==========
	updated, err := s.repository.MoveToParent(ctx, categoryID, req.ParentID)
	if err != nil {
		return nil, fmt.Errorf("move to parent: failed to save")
	}

	// ========== Map to Response DTO ==========
	resp := category.CategoryToResp(updated)

	return resp, nil
}

// ========== UPDATE: Activate ==========
func (s *categoryServiceImpl) Activate(ctx context.Context, categoryID uuid.UUID) (*category.CategoryResp, error) {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return nil, fmt.Errorf("activate category: invalid id")
	}

	// ========== Get Category ==========
	entity, err := s.repository.GetByID(ctx, categoryID)
	if err != nil {
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("activate category: failed to fetch")
	}

	// ========== Check Already Active (Idempotent) ==========
	// CONCEPT - Idempotent:
	// Calling same operation multiple times = same result
	// Activate(active_category) => no change, return successfully
	// Advantage: Safe to retry
	if entity.IsActive {
		resp := category.CategoryToResp(entity)
		return resp, nil
	}

	// ========== Check Parent is Active ==========
	// RULE: Cannot activate child if parent inactive
	// Because parent inactive => child invisible anyway
	if entity.ParentID != nil {
		parent, err := s.repository.GetByID(ctx, *entity.ParentID)
		if err != nil {
			return nil, fmt.Errorf("activate category: failed to verify parent")
		}

		if !parent.IsActive {
			return nil, category.ErrParentInactive
		}
	}

	// ========== Activate in Repository ==========
	updated, err := s.repository.Activate(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("activate category: failed to save")
	}

	// ========== Map to Response DTO ==========
	resp := category.CategoryToResp(updated)

	return resp, nil
}

// ========== UPDATE: Deactivate ==========
func (s *categoryServiceImpl) Deactivate(ctx context.Context, categoryID uuid.UUID) (*category.CategoryResp, error) {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return nil, fmt.Errorf("deactivate category: invalid id")
	}

	// ========== Get Category ==========
	entity, err := s.repository.GetByID(ctx, categoryID)
	if err != nil {
		if category.IsNotFound(err) {
			return nil, category.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("deactivate category: failed to fetch")
	}

	// ========== Check Already Inactive (Idempotent) ==========
	if !entity.IsActive {
		resp := category.CategoryToResp(entity)
		return resp, nil
	}

	// ========== Deactivate in Repository ==========
	// Repository.Deactivate() handles:
	// - Deactivate this category
	// - Cascade deactivate all descendants
	// - Atomic transaction
	updated, err := s.repository.Deactivate(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("deactivate category: failed to save")
	}

	// ========== Map to Response DTO ==========
	resp := category.CategoryToResp(updated)

	return resp, nil
}

// ========== DELETE: Delete ==========
func (s *categoryServiceImpl) Delete(ctx context.Context, categoryID uuid.UUID) error {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return fmt.Errorf("delete category: invalid id")
	}

	// ========== Get Category ==========
	entity, err := s.repository.GetByID(ctx, categoryID)
	if err != nil {
		if category.IsNotFound(err) {
			return category.ErrCategoryNotFound
		}
		return fmt.Errorf("delete category: failed to fetch")
	}

	// ========== Check Can Delete ==========
	// Entity.CanDelete() checks:
	// - No children (ChildCount == 0)
	// - No books (TotalBooksCount == 0)
	if !entity.CanDelete() {

		// Return appropriate error
		if entity.ChildCount != nil && *entity.ChildCount > 0 {
			return category.ErrHasChildren
		}
		if entity.TotalBooksCount != nil && *entity.TotalBooksCount > 0 {
			return category.ErrHasBooks
		}

		return fmt.Errorf("delete category: cannot delete")
	}
	hasChildren, err := s.repository.HasChildren(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if hasChildren {
		logger.Error("Delete: category has children", fmt.Errorf("id: %s", categoryID.String()))
		return category.ErrHasChildren // ← NEW
	}

	// ========== Delete in Repository ==========
	// Hard delete: completely remove from DB
	err = s.repository.Delete(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("delete category: failed to delete")
	}

	// ========== Success ==========
	return nil
}

// ========== BULK: BulkActivate ==========
func (s *categoryServiceImpl) BulkActivate(ctx context.Context, req *category.BulkCategoryIDsReq) (*category.BulkActionResp, error) {
	// ========== Validate Input ==========
	if req == nil || len(req.CategoryIDs) == 0 {
		return nil, fmt.Errorf("bulk activate: invalid request")
	}

	// ========== Bulk Activate in Repository ==========
	// Repository.BulkActivate() handles:
	// - Single UPDATE query with ANY()
	// - Return count of updated rows
	count, err := s.repository.BulkActivate(ctx, req.CategoryIDs)
	if err != nil {
		return nil, fmt.Errorf("bulk activate: failed to update")
	}

	// ========== Build Response ==========
	// success: count of updated
	// failed: total - success
	failed := int64(len(req.CategoryIDs)) - count

	resp := &category.BulkActionResp{
		Success:     count,
		Failed:      failed,
		FailedItems: []category.BulkActionFailedItem{},
	}

	return resp, nil
}

// ========== BULK: BulkDeactivate ==========
func (s *categoryServiceImpl) BulkDeactivate(ctx context.Context, req *category.BulkCategoryIDsReq) (*category.BulkActionResp, error) {
	// ========== Validate Input ==========
	if req == nil || len(req.CategoryIDs) == 0 {
		return nil, fmt.Errorf("bulk deactivate: invalid request")
	}

	// ========== Bulk Deactivate in Repository ==========
	count, err := s.repository.BulkDeactivate(ctx, req.CategoryIDs)
	if err != nil {
		return nil, fmt.Errorf("bulk deactivate: failed to update")
	}

	// ========== Build Response ==========
	failed := int64(len(req.CategoryIDs)) - count

	resp := &category.BulkActionResp{
		Success:     count,
		Failed:      failed,
		FailedItems: []category.BulkActionFailedItem{},
	}
	return resp, nil
}

// ========== BULK: BulkDelete ==========
func (s *categoryServiceImpl) BulkDelete(ctx context.Context, req *category.BulkCategoryIDsReq) (*category.BulkActionResp, error) {
	// ========== Validate Input ==========
	if req == nil || len(req.CategoryIDs) == 0 {
		return nil, fmt.Errorf("bulk delete: invalid request")
	}

	// ========== Validate Each Category ==========
	// Cannot delete all at once without checking
	// Some may have children, some may have books
	// Need to identify which ones can delete, which ones fail

	// ALGORITHM:
	// 1. For each ID:
	//    a. GetByID()
	//    b. Check CanDelete()
	//    c. if yes => add to deletable
	//    d. if no => add to failed_items
	// 2. If deletable not empty:
	//    a. Repository.BulkDelete(deletable)
	// 3. Return both successful + failed

	deletableIDs := make([]uuid.UUID, 0, len(req.CategoryIDs))
	failedItems := make([]category.BulkActionFailedItem, 0)

	for _, id := range req.CategoryIDs {
		entity, err := s.repository.GetByID(ctx, id)
		if err != nil {
			// Not found => add to failed
			failedItems = append(failedItems, category.BulkActionFailedItem{
				ID:     id,
				Reason: "not found",
			})
			continue
		}

		// Check can delete
		if !entity.CanDelete() {
			reason := ""
			if entity.ChildCount != nil && *entity.ChildCount > 0 {
				reason = fmt.Sprintf("has %d children", *entity.ChildCount)
			} else if entity.TotalBooksCount != nil && *entity.TotalBooksCount > 0 {
				reason = fmt.Sprintf("has %d books", *entity.TotalBooksCount)
			}
			failedItems = append(failedItems, category.BulkActionFailedItem{
				ID:     id,
				Reason: reason,
			})
			continue
		}

		// Can delete
		deletableIDs = append(deletableIDs, id)
	}

	// ========== Bulk Delete ==========
	var successCount int64 = 0
	if len(deletableIDs) > 0 {
		count, err := s.repository.BulkDelete(ctx, deletableIDs)
		if err != nil {
			return nil, fmt.Errorf("bulk delete: failed to delete")
		}
		successCount = count
	}

	// ========== Build Response ==========
	resp := &category.BulkActionResp{
		Success:     successCount,
		Failed:      int64(len(failedItems)),
		FailedItems: failedItems,
	}

	return resp, nil
}

// ========== BOOK-RELATED: GetBooksInCategory ==========
func (s *categoryServiceImpl) GetBooksInCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	limit int,
	offset int,
) ([]uuid.UUID, int64, error) {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return nil, 0, fmt.Errorf("get books: invalid category id")
	}

	// Validate pagination
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	// ========== Check Category Exists ==========
	exists, err := s.repository.ExistsByID(ctx, categoryID)
	if err != nil {
		return nil, 0, fmt.Errorf("get books: failed to verify category")
	}

	if !exists {
		return nil, 0, category.ErrCategoryNotFound
	}

	// ========== Get Books from Repository ==========
	bookIDs, total, err := s.repository.GetBooksInCategory(ctx, categoryID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get books: failed to fetch")
	}

	return bookIDs, total, nil
}

// ========== BOOK-RELATED: GetCategoryBookCount ==========
func (s *categoryServiceImpl) GetCategoryBookCount(ctx context.Context, categoryID uuid.UUID) (int64, error) {
	// ========== Validate Input ==========
	if categoryID == uuid.Nil {
		return 0, fmt.Errorf("get book count: invalid category id")
	}

	// ========== Check Category Exists ==========
	exists, err := s.repository.ExistsByID(ctx, categoryID)
	if err != nil {
		return 0, fmt.Errorf("get book count: failed to verify category")
	}

	if !exists {
		return 0, category.ErrCategoryNotFound
	}

	// ========== Get Count from Repository ==========
	count, err := s.repository.GetCategoryBookCount(ctx, categoryID)
	if err != nil {
		return 0, fmt.Errorf("get book count: failed to fetch")
	}

	return count, nil
}
