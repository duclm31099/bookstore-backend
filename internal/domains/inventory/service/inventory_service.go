// internal/domains/inventory/service/service.go
package service

import (
	"bookstore-backend/internal/domains/inventory/model"
	"bookstore-backend/internal/domains/inventory/repository"
	"bookstore-backend/internal/shared"
	"bookstore-backend/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
)

const (
	ReservationTimeoutMinutes = 15
	DefaultAlertThreshold     = 10
)

type InventoryService struct {
	repo  repository.RepositoryInterface
	asynq *asynq.Client // DI từ container, queue riêng inventory
}

func NewService(repo repository.RepositoryInterface, asynq *asynq.Client) ServiceInterface {
	return &InventoryService{
		repo:  repo,
		asynq: asynq,
	}
}

// ========================================
// INVENTORY MANAGEMENT
// ========================================

func (s *InventoryService) CreateInventory(ctx context.Context, req model.CreateInventoryRequest) ([]model.InventoryResponse, error) {
	// Set default alert threshold
	alertThreshold := DefaultAlertThreshold
	if req.AlertThreshold != nil {
		alertThreshold = *req.AlertThreshold
	}

	now := time.Now()

	// Case 1: Create for all active warehouses
	if req.CreateForAllWarehouses && req.WarehouseID == nil {
		// Get all active warehouses
		warehouses, err := s.repo.ListWarehouses(ctx, model.ListWarehousesRequest{
			IsActive: boolPtr(true),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get warehouses: %w", err)
		}

		if len(warehouses) == 0 {
			return nil, fmt.Errorf("no active warehouses found")
		}

		// Create inventory for each warehouse
		inventories := make([]model.Inventory, 0, len(warehouses))
		for _, wh := range warehouses {
			inv := model.Inventory{
				WarehouseID:    wh.ID,
				BookID:         req.BookID,
				Quantity:       req.Quantity,
				Reserved:       0,
				AlertThreshold: alertThreshold,
				Version:        1,
				LastRestockAt:  &now,
				UpdatedBy:      req.UpdatedBy,
			}
			inventories = append(inventories, inv)
		}

		// Batch insert
		if err := s.repo.CreateBatch(ctx, inventories); err != nil {
			return nil, fmt.Errorf("failed to create batch inventories: %w", err)
		}

		// Convert to responses
		responses := make([]model.InventoryResponse, len(inventories))
		for i, inv := range inventories {
			responses[i] = s.inventoryToResponse(inv, warehouses[i].Name)
		}
		return responses, nil
	}

	// Case 2: Create for specific warehouse
	if req.WarehouseID == nil {
		return nil, fmt.Errorf("warehouse_id is required when create_for_all_warehouses is false")
	}

	inventory := model.Inventory{
		WarehouseID:    *req.WarehouseID,
		BookID:         req.BookID,
		Quantity:       req.Quantity,
		Reserved:       0,
		AlertThreshold: alertThreshold,
		Version:        1,
		LastRestockAt:  &now,
		UpdatedBy:      req.UpdatedBy,
	}

	if err := s.repo.Create(ctx, &inventory); err != nil {
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	// Fetch warehouse name
	warehouse, err := s.repo.GetWarehouseByID(ctx, *req.WarehouseID)
	if err != nil {
		return nil, err
	}

	response := s.inventoryToResponse(inventory, warehouse.Name)
	return []model.InventoryResponse{response}, nil
}

func (s *InventoryService) GetInventoryByWarehouseAndBook(ctx context.Context, warehouseID, bookID uuid.UUID) (*model.InventoryResponse, error) {
	inventory, err := s.repo.GetByWarehouseAndBook(ctx, warehouseID, bookID)
	if err != nil {
		return nil, err
	}

	warehouse, err := s.repo.GetWarehouseByID(ctx, warehouseID)
	if err != nil {
		return nil, err
	}

	response := s.inventoryToResponse(*inventory, warehouse.Name)
	return &response, nil
}

func (s *InventoryService) UpdateInventory(ctx context.Context, warehouseID, bookID uuid.UUID, req model.UpdateInventoryRequest) (*model.InventoryResponse, error) {
	// Fetch current inventory
	current, err := s.repo.GetByWarehouseAndBook(ctx, warehouseID, bookID)
	if err != nil {
		return nil, err
	}

	// Verify version (optimistic lock)
	if current.Version != req.Version {
		return nil, model.NewOptimisticLockError(req.Version, current.Version)
	}

	// Build update
	updated := &model.Inventory{
		WarehouseID:    current.WarehouseID,
		BookID:         current.BookID,
		Quantity:       current.Quantity,
		Reserved:       current.Reserved,
		AlertThreshold: current.AlertThreshold,
		Version:        current.Version,
		LastRestockAt:  current.LastRestockAt,
		UpdatedBy:      req.UpdatedBy,
	}

	// Apply partial updates
	if req.Quantity != nil {
		updated.Quantity = *req.Quantity
		// Update restock timestamp if quantity increased
		if *req.Quantity > current.Quantity {
			now := time.Now()
			updated.LastRestockAt = &now
		}
	}

	if req.Reserved != nil {
		updated.Reserved = *req.Reserved
	}

	if req.AlertThreshold != nil {
		updated.AlertThreshold = *req.AlertThreshold
	}

	// Validate: reserved cannot exceed quantity
	if updated.Reserved > updated.Quantity {
		return nil, model.ErrReservedExceedsQuantity
	}

	// Update
	if err := s.repo.Update(ctx, warehouseID, bookID, updated); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	warehouse, _ := s.repo.GetWarehouseByID(ctx, warehouseID)
	response := s.inventoryToResponse(*updated, warehouse.Name)

	// 2. Enqueue InventorySyncJob
	payload := shared.InventorySyncPayload{
		BookID: bookID.String(),
		Source: "ADMIN_ADJUST",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		logger.Error("InventoryService.UpdateStock: payload marshal error", err)
		// Không cần fail request vì stock đã được cập nhật
	} else {
		task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
		if _, err := s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory)); err != nil {
			logger.Error("InventoryService.UpdateStock: failed to enqueue InventorySyncJob", err)
			// Không cần fail request, log alert là đủ
		}
	}
	return &response, nil
}

func (s *InventoryService) DeleteInventory(ctx context.Context, warehouseID, bookID uuid.UUID) error {
	// Fetch to validate
	inventory, err := s.repo.GetByWarehouseAndBook(ctx, warehouseID, bookID)
	if err != nil {
		return err
	}

	// Validate: only delete if empty
	if inventory.Quantity > 0 || inventory.Reserved > 0 {
		return fmt.Errorf("%w: quantity=%d, reserved=%d",
			model.ErrCannotDeleteNonEmptyInventory,
			inventory.Quantity, inventory.Reserved)
	}

	return s.repo.Delete(ctx, warehouseID, bookID)
}

func (s *InventoryService) ListInventories(ctx context.Context, req model.ListInventoryRequest) (*model.ListInventoryResponse, error) {
	inventories, totalItems, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventories: %w", err)
	}

	// Convert to responses
	items := make([]model.InventoryResponse, len(inventories))
	for i, inv := range inventories {
		items[i] = s.inventoryToResponse(inv, inv.WarehouseName) // Name from JOIN
	}

	totalPages := (totalItems + req.Limit - 1) / req.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.ListInventoryResponse{
		Items:      items,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       req.Page,
		Limit:      req.Limit,
	}, nil
}

// ========================================
// STOCK RESERVATION (FR-INV-003)
// ========================================

func (s *InventoryService) ReserveStock(ctx context.Context, req model.ReserveStockRequest) (*model.ReserveStockResponse, error) {
	var warehouseID uuid.UUID
	var warehouseName string

	// Auto-select nearest warehouse if not specified
	if req.WarehouseID == nil {
		if req.CustomerLatitude == nil || req.CustomerLongitude == nil {
			return nil, fmt.Errorf("customer coordinates required for auto warehouse selection")
		}

		nearest, err := s.repo.FindNearestWarehouse(ctx, req.BookID, *req.CustomerLatitude, *req.CustomerLongitude, req.Quantity)
		if err != nil {
			return nil, fmt.Errorf("failed to find warehouse: %w", err)
		}

		warehouseID = nearest.WarehouseID
		warehouseName = nearest.WarehouseName
	} else {
		warehouseID = *req.WarehouseID
		wh, err := s.repo.GetWarehouseByID(ctx, warehouseID)
		if err != nil {
			return nil, err
		}
		warehouseName = wh.Name
	}

	// Call DB function reserve_stock()
	inventory, err := s.repo.ReserveStock(ctx, warehouseID, req.BookID, req.Quantity, req.UserID)
	if err != nil {
		return nil, err
	}
	// Thêm cuối hàm ReserveStock:
	payload := shared.InventorySyncPayload{
		BookID: req.BookID.String(), Source: "RESERVE",
	}
	b, err := json.Marshal(payload)
	if err == nil {
		task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
		s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory))
	}

	expiresAt := time.Now().Add(ReservationTimeoutMinutes * time.Minute)

	return &model.ReserveStockResponse{
		Success:           true,
		WarehouseID:       warehouseID,
		WarehouseName:     warehouseName,
		BookID:            req.BookID,
		ReservedQuantity:  req.Quantity,
		AvailableQuantity: inventory.Quantity - inventory.Reserved,
		ExpiresAt:         expiresAt,
		Message:           fmt.Sprintf("Reserved %d units from %s, expires at %s", req.Quantity, warehouseName, expiresAt.Format(time.RFC3339)),
	}, nil
}

func (s *InventoryService) ReleaseStock(ctx context.Context, req model.ReleaseStockRequest) (*model.ReleaseStockResponse, error) {
	inventory, err := s.repo.ReleaseStock(ctx, req.WarehouseID, req.BookID, req.Quantity, req.UserID)
	if err != nil {
		return nil, err
	}
	// Thêm cuối hàm ReserveStock:
	payload := shared.InventorySyncPayload{
		BookID: req.BookID.String(), Source: "RELEASE",
	}
	b, err := json.Marshal(payload)
	if err == nil {
		task := asynq.NewTask(shared.TypeInventorySyncBookStock, b)
		s.asynq.Enqueue(task, asynq.Queue(shared.QueueInventory))
	}
	return &model.ReleaseStockResponse{
		Success:           true,
		WarehouseID:       req.WarehouseID,
		BookID:            req.BookID,
		ReleasedQuantity:  req.Quantity,
		AvailableQuantity: inventory.Quantity - inventory.Reserved,
		Message:           fmt.Sprintf("Released %d units", req.Quantity),
	}, nil
}

func (s *InventoryService) CompleteSale(ctx context.Context, req model.CompleteSaleRequest) (*model.CompleteSaleResponse, error) {
	inventory, err := s.repo.CompleteSale(ctx, req.WarehouseID, req.BookID, req.Quantity, req.UserID)
	if err != nil {
		return nil, err
	}

	return &model.CompleteSaleResponse{
		Success:      true,
		WarehouseID:  req.WarehouseID,
		BookID:       req.BookID,
		SoldQuantity: req.Quantity,
		Remaining:    inventory.Quantity,
		Message:      fmt.Sprintf("Sold %d units, %d remaining", req.Quantity, inventory.Quantity),
	}, nil
}

// ========================================
// WAREHOUSE SELECTION (FR-INV-002)
// ========================================

func (s *InventoryService) FindOptimalWarehouse(ctx context.Context, req model.FindWarehouseRequest) (*model.WarehouseRecommendation, error) {
	nearest, err := s.repo.FindNearestWarehouse(ctx, req.BookID, req.CustomerLatitude, req.CustomerLongitude, req.RequiredQuantity)
	if err != nil {
		return nil, err
	}

	// Calculate estimated delivery based on distance
	estimatedDelivery := "1-2 days"
	if nearest.DistanceKM > 500 {
		estimatedDelivery = "5-7 days"
	} else if nearest.DistanceKM > 200 {
		estimatedDelivery = "3-5 days"
	}

	return &model.WarehouseRecommendation{
		WarehouseID:       nearest.WarehouseID,
		WarehouseName:     nearest.WarehouseName,
		DistanceKM:        nearest.DistanceKM,
		AvailableQuantity: nearest.AvailableQuantity,
		EstimatedDelivery: estimatedDelivery,
	}, nil
}

func (s *InventoryService) CheckAvailability(ctx context.Context, req model.CheckAvailabilityRequest) (*model.CheckAvailabilityResponse, error) {
	// Extract book IDs
	logger.Info("CheckAvailability", map[string]interface{}{
		"req": req,
	})
	bookIDs := make([]uuid.UUID, len(req.Items))
	for i, item := range req.Items {
		bookIDs[i] = item.BookID
	}

	// Fetch inventories for all books
	inventoriesByBook := make(map[uuid.UUID][]model.Inventory)
	for _, bookID := range bookIDs {
		invs, err := s.repo.GetInventoriesByBook(ctx, bookID)
		if err != nil {
			return nil, err
		}
		inventoriesByBook[bookID] = invs
	}

	// Process each item
	itemResponses := make([]model.CheckAvailabilityItemResponse, 0, len(req.Items))
	overallFulfillable := true
	warehouseScores := make(map[uuid.UUID]int)

	for _, item := range req.Items {
		inventories := inventoriesByBook[item.BookID]

		warehouseDetails := make([]model.WarehouseStockDetail, 0, len(inventories))
		totalAvailable := 0
		canFulfill := false

		for _, inv := range inventories {
			available := inv.Quantity - inv.Reserved
			detail := model.WarehouseStockDetail{
				WarehouseID:   inv.WarehouseID,
				WarehouseName: inv.WarehouseName,
				Available:     available,
				CanFulfill:    available >= item.Quantity,
			}

			if detail.CanFulfill {
				canFulfill = true
				warehouseScores[inv.WarehouseID]++
			}

			totalAvailable += available
			warehouseDetails = append(warehouseDetails, detail)
		}

		itemResp := model.CheckAvailabilityItemResponse{
			BookID:            item.BookID,
			RequestedQuantity: item.Quantity,
			TotalAvailable:    totalAvailable,
			Fulfillable:       canFulfill,
			WarehouseDetails:  warehouseDetails,
		}

		if !canFulfill {
			overallFulfillable = false
			itemResp.Recommendation = fmt.Sprintf("Only %d available, need %d", totalAvailable, item.Quantity)
		}

		itemResponses = append(itemResponses, itemResp)
	}
	logger.Info("itemResponses", map[string]interface{}{
		"itemResponses": itemResponses,
	})
	// Find best warehouse (most items can fulfill)
	var recommendedWarehouse *model.WarehouseRecommendation
	if overallFulfillable {
		maxScore := 0
		var bestWarehouseID uuid.UUID
		for whID, score := range warehouseScores {
			if score > maxScore {
				maxScore = score
				bestWarehouseID = whID
			}
		}
		logger.Info("bestWarehouseID", map[string]interface{}{
			"bestWarehouseID": bestWarehouseID,
		})
		if maxScore > 0 {
			wh, _ := s.repo.GetWarehouseByID(ctx, bestWarehouseID)
			recommendedWarehouse = &model.WarehouseRecommendation{
				WarehouseID:   bestWarehouseID,
				WarehouseName: wh.Name,
			}
		}
	}
	logger.Info("recommendedWarehouse", map[string]interface{}{
		"recommendedWarehouse": recommendedWarehouse,
	})
	return &model.CheckAvailabilityResponse{
		Overall:              overallFulfillable,
		Items:                itemResponses,
		RecommendedWarehouse: recommendedWarehouse,
		RequiresSplit:        false, // TODO: Calculate if need multiple warehouses
	}, nil
}

func (s *InventoryService) GetStockSummary(ctx context.Context, bookID uuid.UUID) (*model.StockSummaryResponse, error) {
	// Use VIEW books_total_stock
	totalStock, err := s.repo.GetTotalStockForBook(ctx, bookID)
	if err != nil {
		return nil, err
	}

	// Get warehouse breakdown
	inventories, err := s.repo.GetInventoriesByBook(ctx, bookID)
	if err != nil {
		return nil, err
	}

	byWarehouse := make([]model.WarehouseStockSummary, len(inventories))
	for i, inv := range inventories {
		byWarehouse[i] = model.WarehouseStockSummary{
			WarehouseID:    inv.WarehouseID,
			WarehouseName:  inv.WarehouseName,
			Quantity:       inv.Quantity,
			Reserved:       inv.Reserved,
			Available:      inv.Quantity - inv.Reserved,
			IsLowStock:     inv.Quantity < inv.AlertThreshold,
			AlertThreshold: inv.AlertThreshold,
		}
	}

	return &model.StockSummaryResponse{
		BookID:         bookID,
		TotalQuantity:  totalStock.TotalQuantity,
		TotalReserved:  totalStock.TotalReserved,
		TotalAvailable: totalStock.TotalAvailable,
		WarehouseCount: totalStock.WarehouseCount,
		ByWarehouse:    byWarehouse,
	}, nil
}

// ========================================
// STOCK ADJUSTMENT (FR-INV-005)
// ========================================

func (s *InventoryService) AdjustStock(ctx context.Context, req model.AdjustStockRequest) (*model.AdjustStockResponse, error) {
	// Fetch current
	current, err := s.repo.GetByWarehouseAndBook(ctx, req.WarehouseID, req.BookID)
	if err != nil {
		return nil, err
	}

	// Verify version
	if current.Version != req.Version {
		return nil, model.NewOptimisticLockError(req.Version, current.Version)
	}

	// Validate: new quantity >= reserved
	if req.NewQuantity < current.Reserved {
		return nil, fmt.Errorf("new quantity (%d) cannot be less than reserved (%d)", req.NewQuantity, current.Reserved)
	}

	// Update
	updated := &model.Inventory{
		WarehouseID:    req.WarehouseID,
		BookID:         req.BookID,
		Quantity:       req.NewQuantity,
		Reserved:       current.Reserved,
		AlertThreshold: current.AlertThreshold,
		Version:        current.Version,
		UpdatedBy:      &req.ChangedBy,
	}

	if err := s.repo.Update(ctx, req.WarehouseID, req.BookID, updated); err != nil {
		return nil, err
	}

	// Audit log created automatically by trigger
	return &model.AdjustStockResponse{
		Success:        true,
		WarehouseID:    req.WarehouseID,
		BookID:         req.BookID,
		OldQuantity:    current.Quantity,
		NewQuantity:    req.NewQuantity,
		QuantityChange: req.NewQuantity - current.Quantity,
		Message:        fmt.Sprintf("Adjusted stock from %d to %d. Reason: %s", current.Quantity, req.NewQuantity, req.Reason),
	}, nil
}

func (s *InventoryService) RestockInventory(ctx context.Context, req model.RestockRequest) (*model.RestockResponse, error) {
	// Fetch current
	current, err := s.repo.GetByWarehouseAndBook(ctx, req.WarehouseID, req.BookID)
	if err != nil {
		return nil, err
	}

	// Calculate new quantity
	newQuantity := current.Quantity + req.QuantityToAdd
	now := time.Now()

	// Update
	updated := &model.Inventory{
		WarehouseID:    req.WarehouseID,
		BookID:         req.BookID,
		Quantity:       newQuantity,
		Reserved:       current.Reserved,
		AlertThreshold: current.AlertThreshold,
		Version:        current.Version,
		LastRestockAt:  &now,
		UpdatedBy:      req.UpdatedBy,
	}

	if err := s.repo.Update(ctx, req.WarehouseID, req.BookID, updated); err != nil {
		return nil, err
	}

	return &model.RestockResponse{
		Success:       true,
		WarehouseID:   req.WarehouseID,
		BookID:        req.BookID,
		QuantityAdded: req.QuantityToAdd,
		NewQuantity:   newQuantity,
		LastRestockAt: now,
		Message:       fmt.Sprintf("Added %d units, total now %d", req.QuantityToAdd, newQuantity),
	}, nil
}

func (s *InventoryService) BulkUpdateStock(ctx context.Context, csvPath string, uploadedBy uuid.UUID) (*model.BulkUpdateJobResponse, error) {
	// TODO: Implement async job processing with Asynq
	// 1. Parse CSV
	// 2. Validate rows
	// 3. Create job with status "queued"
	// 4. Enqueue to Asynq
	// 5. Return job ID
	return nil, fmt.Errorf("not implemented: bulk update requires Asynq integration")
}

func (s *InventoryService) GetBulkUpdateStatus(ctx context.Context, jobID uuid.UUID) (*model.BulkUpdateStatusResponse, error) {
	// TODO: Query job status from Asynq or jobs table
	return nil, fmt.Errorf("not implemented")
}

// ========================================
// ALERTS (FR-INV-004)
// ========================================

func (s *InventoryService) GetLowStockAlerts(ctx context.Context) ([]model.LowStockAlert, error) {
	alerts, err := s.repo.GetLowStockAlerts(ctx, false) // Unresolved only
	if err != nil {
		return nil, err
	}

	// Calculate priority
	for i := range alerts {
		if alerts[i].CurrentQuantity == 0 {
			alerts[i].Priority = "critical"
		} else if alerts[i].CurrentQuantity <= alerts[i].AlertThreshold/2 {
			alerts[i].Priority = "high"
		} else {
			alerts[i].Priority = "medium"
		}
	}

	return alerts, nil
}

func (s *InventoryService) GetOutOfStockItems(ctx context.Context) ([]model.OutOfStockItem, error) {
	// Query inventories with quantity = 0
	filter := model.ListInventoryRequest{
		Page:  1,
		Limit: 100,
	}

	inventories, _, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]model.OutOfStockItem, 0)
	now := time.Now()

	for _, inv := range inventories {
		if inv.Quantity == 0 {
			item := model.OutOfStockItem{
				BookID:          inv.BookID,
				WarehouseID:     inv.WarehouseID,
				WarehouseName:   inv.WarehouseName,
				ReservedStock:   inv.Reserved,
				LastRestockDate: inv.LastRestockAt,
			}

			if inv.LastRestockAt != nil {
				item.DaysSinceStockout = int(now.Sub(*inv.LastRestockAt).Hours() / 24)
			}

			items = append(items, item)
		}
	}

	return items, nil
}

func (s *InventoryService) MarkAlertResolved(ctx context.Context, alertID uuid.UUID) error {
	// TODO: Implement manual alert resolution
	// Normally auto-resolved by trigger when restocked
	return fmt.Errorf("not implemented")
}

// ========================================
// AUDIT & REPORTING (FR-INV-005)
// ========================================

func (s *InventoryService) GetAuditTrail(ctx context.Context, req model.AuditTrailRequest) (*model.AuditTrailResponse, error) {
	logs, totalItems, err := s.repo.GetAuditLog(ctx, req.WarehouseID, req.BookID, req.StartDate, req.EndDate, req.Limit, (req.Page-1)*req.Limit)
	if err != nil {
		return nil, err
	}

	totalPages := (totalItems + req.Limit - 1) / req.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.AuditTrailResponse{
		Items:      logs,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       req.Page,
		Limit:      req.Limit,
	}, nil
}

func (s *InventoryService) GetInventoryHistory(ctx context.Context, warehouseID, bookID uuid.UUID, limit, offset int) (*model.InventoryHistoryResponse, error) {
	logs, totalItems, err := s.repo.GetAuditLog(ctx, &warehouseID, &bookID, nil, nil, limit, offset)
	if err != nil {
		return nil, err
	}

	return &model.InventoryHistoryResponse{
		WarehouseID: warehouseID,
		BookID:      bookID,
		History:     logs,
		TotalItems:  totalItems,
		Page:        (offset / limit) + 1,
		Limit:       limit,
	}, nil
}

func (s *InventoryService) ExportAuditLog(ctx context.Context, req model.ExportAuditRequest) (*model.ExportResponse, error) {
	// TODO: Implement CSV/Excel export
	return nil, fmt.Errorf("not implemented")
}

// ========================================
// DASHBOARD & ANALYTICS
// ========================================

func (s *InventoryService) GetDashboardSummary(ctx context.Context) (*model.DashboardSummaryResponse, error) {
	summary, err := s.repo.GetDashboardMetrics(ctx)
	if err != nil {
		return nil, err
	}

	warehouseMetrics, err := s.repo.GetWarehouseMetrics(ctx)
	if err != nil {
		return nil, err
	}

	lowStockAlerts, err := s.GetLowStockAlerts(ctx)
	if err != nil {
		return nil, err
	}

	// Get recent movements (last 10)
	recentMovements, _, err := s.repo.GetAuditLog(ctx, nil, nil, nil, nil, 10, 0)
	if err != nil {
		return nil, err
	}

	return &model.DashboardSummaryResponse{
		Summary:         *summary,
		ByWarehouse:     warehouseMetrics,
		LowStockAlerts:  lowStockAlerts,
		RecentMovements: recentMovements,
		Timestamp:       time.Now(),
	}, nil
}

// ========================================
// WAREHOUSE MANAGEMENT
// ========================================

func (s *InventoryService) GetWarehouseByID(ctx context.Context, warehouseID uuid.UUID) (*model.WarehouseResponse, error) {
	warehouse, err := s.repo.GetWarehouseByID(ctx, warehouseID)
	if err != nil {
		return nil, err
	}

	return &model.WarehouseResponse{
		ID:        warehouse.ID,
		Name:      warehouse.Name,
		Code:      warehouse.Code,
		Address:   warehouse.Address,
		Province:  warehouse.Province,
		Latitude:  warehouse.Latitude,
		Longitude: warehouse.Longitude,
		IsActive:  warehouse.IsActive,
		Version:   warehouse.Version,
		CreatedAt: warehouse.CreatedAt,
		UpdatedAt: warehouse.UpdatedAt,
	}, nil
}

// ========================================
// HELPER METHODS
// ========================================

func (s *InventoryService) inventoryToResponse(inv model.Inventory, warehouseName string) model.InventoryResponse {
	available := inv.Quantity - inv.Reserved
	isLowStock := inv.Quantity < inv.AlertThreshold

	return model.InventoryResponse{
		WarehouseID:    inv.WarehouseID,
		WarehouseName:  warehouseName,
		BookID:         inv.BookID,
		Quantity:       inv.Quantity,
		Reserved:       inv.Reserved,
		Available:      available,
		AlertThreshold: inv.AlertThreshold,
		IsLowStock:     isLowStock,
		Version:        inv.Version,
		LastRestockAt:  inv.LastRestockAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// internal/domains/inventory/service/service.go

// GetReservationAnalysis implements Service.GetReservationAnalysis
func (s *InventoryService) GetReservationAnalysis(ctx context.Context) (*model.ReservationAnalysisResponse, error) {
	metrics, err := s.repo.GetReservationMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation metrics: %w", err)
	}

	return &model.ReservationAnalysisResponse{
		TotalReserved:      metrics.TotalReserved,
		ReservationRate:    metrics.ReservationRate,
		ByWarehouse:        metrics.ByWarehouse,
		AvgDurationMinutes: metrics.AvgDurationMinutes,
		ConversionRate:     metrics.ConversionRate,
	}, nil
}

// CreateWarehouse implements Service.CreateWarehouse
func (s *InventoryService) CreateWarehouse(ctx context.Context, req model.CreateWarehouseRequest) (*model.WarehouseResponse, error) {
	warehouse := &model.Warehouse{
		ID:        uuid.New(),
		Name:      req.Name,
		Code:      req.Code,
		Address:   req.Address,
		Province:  req.Province,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		IsActive:  true,
		Version:   1,
	}

	if err := s.repo.CreateWarehouse(ctx, warehouse); err != nil {
		return nil, fmt.Errorf("failed to create warehouse: %w", err)
	}

	return &model.WarehouseResponse{
		ID:        warehouse.ID,
		Name:      warehouse.Name,
		Code:      warehouse.Code,
		Address:   warehouse.Address,
		Province:  warehouse.Province,
		Latitude:  warehouse.Latitude,
		Longitude: warehouse.Longitude,
		IsActive:  warehouse.IsActive,
		Version:   warehouse.Version,
		CreatedAt: warehouse.CreatedAt,
		UpdatedAt: warehouse.UpdatedAt,
	}, nil
}

// UpdateWarehouse implements Service.UpdateWarehouse
func (s *InventoryService) UpdateWarehouse(ctx context.Context, warehouseID uuid.UUID, req model.UpdateWarehouseRequest) (*model.WarehouseResponse, error) {
	// Fetch current
	current, err := s.repo.GetWarehouseByID(ctx, warehouseID)
	if err != nil {
		return nil, err
	}

	// Verify version
	if current.Version != req.Version {
		return nil, model.NewOptimisticLockError(req.Version, current.Version)
	}

	// Build update
	updated := &model.Warehouse{
		ID:        current.ID,
		Name:      current.Name,
		Code:      current.Code, // Code cannot be updated
		Address:   current.Address,
		Province:  current.Province,
		Latitude:  current.Latitude,
		Longitude: current.Longitude,
		IsActive:  current.IsActive,
		Version:   current.Version,
	}

	// Apply partial updates
	if req.Name != nil {
		updated.Name = *req.Name
	}
	if req.Address != nil {
		updated.Address = *req.Address
	}
	if req.Province != nil {
		updated.Province = *req.Province
	}
	if req.Latitude != nil {
		updated.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		updated.Longitude = req.Longitude
	}
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}

	// Update
	if err := s.repo.UpdateWarehouse(ctx, warehouseID, updated); err != nil {
		return nil, fmt.Errorf("failed to update warehouse: %w", err)
	}

	return &model.WarehouseResponse{
		ID:        updated.ID,
		Name:      updated.Name,
		Code:      updated.Code,
		Address:   updated.Address,
		Province:  updated.Province,
		Latitude:  updated.Latitude,
		Longitude: updated.Longitude,
		IsActive:  updated.IsActive,
		Version:   updated.Version,
		CreatedAt: current.CreatedAt,
		UpdatedAt: updated.UpdatedAt,
	}, nil
}

// ListWarehouses implements Service.ListWarehouses
func (s *InventoryService) ListWarehouses(ctx context.Context, req model.ListWarehousesRequest) ([]model.WarehouseResponse, error) {
	warehouses, err := s.repo.ListWarehouses(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w", err)
	}

	responses := make([]model.WarehouseResponse, len(warehouses))
	for i, wh := range warehouses {
		responses[i] = model.WarehouseResponse{
			ID:        wh.ID,
			Name:      wh.Name,
			Code:      wh.Code,
			Address:   wh.Address,
			Province:  wh.Province,
			Latitude:  wh.Latitude,
			Longitude: wh.Longitude,
			IsActive:  wh.IsActive,
			Version:   wh.Version,
			CreatedAt: wh.CreatedAt,
			UpdatedAt: wh.UpdatedAt,
		}
	}

	return responses, nil
}

// DeactivateWarehouse implements Service.DeactivateWarehouse
func (s *InventoryService) DeactivateWarehouse(ctx context.Context, warehouseID uuid.UUID) error {
	return s.repo.DeactivateWarehouse(ctx, warehouseID)
}

// GetWarehousePerformance implements Service.GetWarehousePerformance
func (s *InventoryService) GetWarehousePerformance(ctx context.Context, warehouseID uuid.UUID) (*model.WarehousePerformanceResponse, error) {
	// Get warehouse details
	warehouse, err := s.repo.GetWarehouseByID(ctx, warehouseID)
	if err != nil {
		return nil, err
	}

	// Get metrics for this warehouse
	allMetrics, err := s.repo.GetWarehouseMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var warehouseMetric *model.WarehouseMetrics
	for _, m := range allMetrics {
		if m.WarehouseID == warehouseID {
			warehouseMetric = &m
			break
		}
	}

	if warehouseMetric == nil {
		return nil, fmt.Errorf("metrics not found for warehouse %s", warehouseID)
	}

	// Get movement trends (last 30 days)
	trends, err := s.repo.GetMovementTrends(ctx, 30)
	if err != nil {
		return nil, err
	}

	return &model.WarehousePerformanceResponse{
		WarehouseID:    warehouse.ID,
		WarehouseName:  warehouse.Name,
		Metrics:        *warehouseMetric,
		MovementTrends: trends,
	}, nil
}

// ReserveStockWithTx reserves stock within an existing transaction
func (s *InventoryService) ReserveStockWithTx(
	ctx context.Context,
	tx pgx.Tx,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userid *uuid.UUID,
) error {
	// Validate business rules
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	// Call repository with transaction
	return s.repo.ReserveStockWithTx(ctx, tx, warehouseID, bookID, quantity, userid)
}

// ReleaseStockWithTx releases reserved stock within an existing transaction
func (s *InventoryService) ReleaseStockWithTx(
	ctx context.Context,
	tx pgx.Tx,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	quantity int,
	userid *uuid.UUID,
) error {
	// Validate business rules
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	// Call repository with transaction
	return s.repo.ReleaseStockWithTx(ctx, tx, warehouseID, bookID, quantity, userid)
}

// CheckAvailableStock checks if enough stock is available
func (s *InventoryService) CheckAvailableStock(
	ctx context.Context,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
	requiredQty int,
) (bool, error) {
	// Get current available quantity
	availableQty, err := s.repo.GetAvailableQuantity(ctx, warehouseID, bookID)
	if err != nil {
		return false, fmt.Errorf("failed to get available quantity: %w", err)
	}

	// Check if available >= required
	return availableQty >= requiredQty, nil
}

// GetAvailableQuantity returns the available quantity for a book at a warehouse
func (s *InventoryService) GetAvailableQuantity(
	ctx context.Context,
	warehouseID uuid.UUID,
	bookID uuid.UUID,
) (int, error) {
	return s.repo.GetAvailableQuantity(ctx, warehouseID, bookID)
}
