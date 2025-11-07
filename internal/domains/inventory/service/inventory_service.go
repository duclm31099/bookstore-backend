package service

import (
	"bookstore-backend/internal/domains/inventory/model"
	"bookstore-backend/internal/domains/inventory/repository"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type InventoryService struct {
	repo repository.RepositoryInterface
}

// NewService creates a new inventory service
func NewService(repo repository.RepositoryInterface) ServiceInterface {
	return &InventoryService{
		repo: repo,
	}
}

// CreateInventory implements Service.CreateInventory
func (s *InventoryService) CreateInventory(ctx context.Context, req model.CreateInventoryRequest) ([]model.InventoryResponse, error) {
	// Validate warehouse location
	if req.WarehouseLocation != "ALL" && !model.IsValidWarehouse(req.WarehouseLocation) {
		return nil, model.ErrInvalidWarehouseLocation
	}

	// Set default low stock threshold if not provided
	lowStockThreshold := 10 // Default value
	if req.LowStockThreshold != nil {
		lowStockThreshold = *req.LowStockThreshold
	}

	now := time.Now()

	// Case 1: Create for ALL warehouses
	if req.WarehouseLocation == "ALL" {
		inventories := make([]model.Inventory, 0, len(model.ValidWarehouseLocations))

		for _, warehouse := range model.ValidWarehouseLocations {
			inventory := model.Inventory{
				ID:                uuid.New(),
				BookID:            req.BookID,
				WarehouseLocation: warehouse,
				Quantity:          req.Quantity,
				ReservedQuantity:  0,
				LowStockThreshold: lowStockThreshold,
				Version:           1, // Initial version
				LastRestockAt:     &now,
				UpdatedAt:         now,
			}
			inventories = append(inventories, inventory)
		}

		// Batch insert using CopyFrom
		if err := s.repo.CreateBatch(ctx, inventories); err != nil {
			return nil, fmt.Errorf("failed to create batch inventories: %w", err)
		}

		// Convert to response DTOs
		return model.ToResponseList(inventories), nil
	}

	// Case 2: Create for single warehouse
	inventory := model.Inventory{
		ID:                uuid.New(),
		BookID:            req.BookID,
		WarehouseLocation: req.WarehouseLocation,
		Quantity:          req.Quantity,
		ReservedQuantity:  0,
		LowStockThreshold: lowStockThreshold,
		Version:           1,
		LastRestockAt:     &now,
		UpdatedAt:         now,
	}

	if err := s.repo.Create(ctx, &inventory); err != nil {
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	return []model.InventoryResponse{inventory.ToResponse()}, nil
}

// GetInventoryByID implements Service.GetInventoryByID
func (s *InventoryService) GetInventoryByID(ctx context.Context, id uuid.UUID) (*model.InventoryResponse, error) {
	inventory, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	response := inventory.ToResponse()
	return &response, nil
}

// SearchInventory implements Service.SearchInventory
func (s *InventoryService) SearchInventory(ctx context.Context, req model.SearchInventoryRequest) (*model.InventoryResponse, error) {
	// Validate warehouse location
	if !model.IsValidWarehouse(req.WarehouseLocation) {
		return nil, model.ErrInvalidWarehouseLocation
	}

	inventory, err := s.repo.GetByBookAndWarehouse(ctx, req.BookID, req.WarehouseLocation)
	if err != nil {
		return nil, err
	}

	response := inventory.ToResponse()
	return &response, nil
}

// ListInventories implements Service.ListInventories
func (s *InventoryService) ListInventories(ctx context.Context, req model.ListInventoryRequest) (*model.ListInventoryResponse, error) {
	// Validate warehouse location if provided
	if req.WarehouseLocation != nil && !model.IsValidWarehouse(*req.WarehouseLocation) {
		return nil, model.ErrInvalidWarehouseLocation
	}

	// Get inventories and total count
	inventories, totalItems, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventories: %w", err)
	}

	// Calculate total pages
	totalPages := (totalItems + req.Limit - 1) / req.Limit // Ceiling division
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert to response DTOs
	items := model.ToResponseList(inventories)

	return &model.ListInventoryResponse{
		Items:      items,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       req.Page,
		Limit:      req.Limit,
	}, nil
}

// UpdateInventory implements Service.UpdateInventory
func (s *InventoryService) UpdateInventory(ctx context.Context, id uuid.UUID, req model.UpdateInventoryRequest) (*model.InventoryResponse, error) {
	// Fetch current inventory (for validation and version check)
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify version matches (optimistic locking check)
	if current.Version != req.Version {
		return nil, model.NewOptimisticLockError(req.Version, current.Version)
	}

	// Build update object with only changed fields
	updated := &model.Inventory{
		ID:                current.ID,
		BookID:            current.BookID,
		WarehouseLocation: current.WarehouseLocation,
		Quantity:          current.Quantity,
		ReservedQuantity:  current.ReservedQuantity,
		LowStockThreshold: current.LowStockThreshold,
		Version:           current.Version,
		LastRestockAt:     current.LastRestockAt,
	}

	// Apply partial updates
	if req.Quantity != nil {
		if *req.Quantity < 0 {
			return nil, model.ErrInvalidQuantity
		}
		updated.Quantity = *req.Quantity

		// Update last restock timestamp if quantity increased
		if *req.Quantity > current.Quantity {
			now := time.Now()
			updated.LastRestockAt = &now
		}
	}

	if req.ReservedQuantity != nil {
		if *req.ReservedQuantity < 0 {
			return nil, model.ErrInvalidQuantity
		}
		updated.ReservedQuantity = *req.ReservedQuantity
	}

	if req.LowStockThreshold != nil {
		if *req.LowStockThreshold < 0 {
			return nil, model.ErrInvalidQuantity
		}
		updated.LowStockThreshold = *req.LowStockThreshold
	}

	// Validate: reserved cannot exceed total quantity
	if updated.ReservedQuantity > updated.Quantity {
		return nil, model.ErrReservedExceedsQuantity
	}

	// Perform update with version increment
	if err := s.repo.Update(ctx, id, updated); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	response := updated.ToResponse()
	return &response, nil
}

// DeleteInventory implements Service.DeleteInventory
func (s *InventoryService) DeleteInventory(ctx context.Context, id uuid.UUID) error {
	// Fetch current inventory to validate before delete
	inventory, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate: only allow deletion if quantity = 0 and reserved_quantity = 0
	if inventory.Quantity > 0 || inventory.ReservedQuantity > 0 {
		return fmt.Errorf("%w: quantity=%d, reserved=%d",
			model.ErrCannotDeleteNonEmptyInventory,
			inventory.Quantity,
			inventory.ReservedQuantity,
		)
	}

	// Perform delete
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	return nil
}

// ReserveStock implements Service.ReserveStock
func (s *InventoryService) ReserveStock(ctx context.Context, req model.ReserveStockRequest) (*model.ReserveStockResponse, error) {
	// Validate warehouse location
	if !model.IsValidWarehouse(req.WarehouseLocation) {
		return nil, model.ErrInvalidWarehouseLocation
	}

	// Perform atomic reserve operation
	inventory, err := s.repo.ReserveStock(
		ctx,
		req.BookID,
		req.WarehouseLocation,
		req.Quantity,
		req.ReferenceType,
		req.ReferenceID,
	)

	if err != nil {
		return nil, err
	}

	return &model.ReserveStockResponse{
		Success:           true,
		BookID:            inventory.BookID,
		WarehouseLocation: inventory.WarehouseLocation,
		ReservedQuantity:  req.Quantity,
		AvailableQuantity: inventory.AvailableQuantity,
		Message:           fmt.Sprintf("Successfully reserved %d units", req.Quantity),
	}, nil
}

// ReleaseStock implements Service.ReleaseStock
func (s *InventoryService) ReleaseStock(ctx context.Context, req model.ReleaseStockRequest) (*model.ReleaseStockResponse, error) {
	// Validate warehouse location
	if !model.IsValidWarehouse(req.WarehouseLocation) {
		return nil, model.ErrInvalidWarehouseLocation
	}

	// Perform atomic release operation
	inventory, err := s.repo.ReleaseStock(
		ctx,
		req.BookID,
		req.WarehouseLocation,
		req.Quantity,
		req.ReferenceID,
	)

	if err != nil {
		return nil, err
	}

	return &model.ReleaseStockResponse{
		Success:           true,
		BookID:            inventory.BookID,
		WarehouseLocation: inventory.WarehouseLocation,
		ReleasedQuantity:  req.Quantity,
		AvailableQuantity: inventory.AvailableQuantity,
		Message:           fmt.Sprintf("Successfully released %d units", req.Quantity),
	}, nil
}

// CheckAvailability implements Service.CheckAvailability
func (s *InventoryService) CheckAvailability(ctx context.Context, req model.CheckAvailabilityRequest) (*model.CheckAvailabilityResponse, error) {
	// Extract unique book IDs
	bookIDs := make([]uuid.UUID, 0, len(req.Items))
	bookIDMap := make(map[uuid.UUID]struct{})
	for _, item := range req.Items {
		if _, exists := bookIDMap[item.BookID]; !exists {
			bookIDs = append(bookIDs, item.BookID)
			bookIDMap[item.BookID] = struct{}{}
		}
	}

	// Fetch all inventories for these books in single query
	inventoriesByBook, err := s.repo.GetInventoriesByBooks(ctx, bookIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}

	// Process each requested item
	itemResponses := make([]model.CheckAvailabilityItemResponse, 0, len(req.Items))
	overallFulfillable := true
	suggestedWarehouse := ""
	maxFulfillableCount := 0

	for _, item := range req.Items {
		inventories := inventoriesByBook[item.BookID]

		// Calculate per-warehouse availability
		warehouseDetails := make([]model.WarehouseAvailability, 0, len(inventories))
		fulfillableFrom := make([]string, 0)
		totalAvailable := 0
		available := false

		for _, inv := range inventories {
			detail := model.WarehouseAvailability{
				Warehouse:         inv.WarehouseLocation,
				Quantity:          inv.Quantity,
				Reserved:          inv.ReservedQuantity,
				Available:         inv.AvailableQuantity,
				CanFulfill:        inv.AvailableQuantity >= item.Quantity,
				IsLowStock:        inv.IsLowStock,
				LowStockThreshold: inv.LowStockThreshold,
			}

			if detail.CanFulfill {
				fulfillableFrom = append(fulfillableFrom, inv.WarehouseLocation)
				available = true

				// Count for suggested warehouse
				if req.PreferredWarehouse != nil && *req.PreferredWarehouse == inv.WarehouseLocation {
					maxFulfillableCount++
				}
			}

			totalAvailable += inv.AvailableQuantity
			warehouseDetails = append(warehouseDetails, detail)
		}

		// Build item response
		itemResp := model.CheckAvailabilityItemResponse{
			BookID:            item.BookID,
			RequestedQuantity: item.Quantity,
			Available:         available,
			WarehouseDetails:  warehouseDetails,
			TotalAvailable:    totalAvailable,
			FulfillableFrom:   fulfillableFrom,
		}

		// Add recommendation
		if !available {
			itemResp.Recommendation = fmt.Sprintf("Only %d units available, need %d", totalAvailable, item.Quantity)
			overallFulfillable = false
		} else if req.PreferredWarehouse != nil {
			// Check if preferred warehouse can fulfill
			canFulfillFromPreferred := false
			for _, warehouse := range fulfillableFrom {
				if warehouse == *req.PreferredWarehouse {
					canFulfillFromPreferred = true
					break
				}
			}
			if canFulfillFromPreferred {
				itemResp.Recommendation = fmt.Sprintf("Can fulfill from preferred warehouse: %s", *req.PreferredWarehouse)
			} else {
				itemResp.Recommendation = fmt.Sprintf("Preferred warehouse %s unavailable, suggest: %s", *req.PreferredWarehouse, fulfillableFrom[0])
			}
		}

		itemResponses = append(itemResponses, itemResp)
	}

	// Determine suggested warehouse if all items fulfillable
	if overallFulfillable && maxFulfillableCount > 0 {
		suggestedWarehouse = *req.PreferredWarehouse
	} else if overallFulfillable {
		// Suggest most commonly available warehouse
		warehouseCount := make(map[string]int)
		for _, item := range itemResponses {
			for _, warehouse := range item.FulfillableFrom {
				warehouseCount[warehouse]++
			}
		}
		for warehouse, count := range warehouseCount {
			if count > maxFulfillableCount {
				maxFulfillableCount = count
				suggestedWarehouse = warehouse
			}
		}
	}

	return &model.CheckAvailabilityResponse{
		Overall:       overallFulfillable,
		Items:         itemResponses,
		Fulfillable:   overallFulfillable,
		SuggestedFrom: suggestedWarehouse,
	}, nil
}

// GetStockSummary implements Service.GetStockSummary
func (s *InventoryService) GetStockSummary(ctx context.Context, bookID uuid.UUID) (*model.StockSummaryResponse, error) {
	inventories, err := s.repo.GetInventoriesByBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock summary: %w", err)
	}

	if len(inventories) == 0 {
		return nil, model.NewInventoryNotFoundError(bookID)
	}

	// Aggregate across warehouses
	byWarehouse := make([]model.WarehouseStockSummary, 0, len(inventories))
	totalQuantity := 0
	totalReserved := 0
	totalAvailable := 0

	for _, inv := range inventories {
		byWarehouse = append(byWarehouse, model.WarehouseStockSummary{
			Warehouse:         inv.WarehouseLocation,
			Quantity:          inv.Quantity,
			Reserved:          inv.ReservedQuantity,
			Available:         inv.AvailableQuantity,
			IsLowStock:        inv.IsLowStock,
			LowStockThreshold: inv.LowStockThreshold,
		})

		totalQuantity += inv.Quantity
		totalReserved += inv.ReservedQuantity
		totalAvailable += inv.AvailableQuantity
	}

	return &model.StockSummaryResponse{
		BookID:         bookID,
		TotalQuantity:  totalQuantity,
		TotalReserved:  totalReserved,
		TotalAvailable: totalAvailable,
		ByWarehouse:    byWarehouse,
	}, nil
}

// CreateMovement implements Service.CreateMovement
func (s *InventoryService) CreateMovement(ctx context.Context, req model.CreateMovementRequest) (*model.MovementResponse, error) {
	// Validate quantity
	if req.Quantity == 0 {
		return nil, fmt.Errorf("%w: quantity must be non-zero", model.ErrInvalidAdjustmentQuantity)
	}

	// Validate movement type
	validTypes := []string{"inbound", "outbound", "adjustment", "return"}
	validType := false
	for _, t := range validTypes {
		if req.MovementType == t {
			validType = true
			break
		}
	}
	if !validType {
		return nil, fmt.Errorf("%w: %s", model.ErrInvalidMovementType, req.MovementType)
	}

	// Fetch inventory to get before state
	inventory, err := s.repo.GetByID(ctx, req.InventoryID)
	if err != nil {
		return nil, err
	}

	// Create movement record
	movement := &model.InventoryMovement{
		ID:             uuid.New(),
		InventoryID:    req.InventoryID,
		MovementType:   req.MovementType,
		Quantity:       req.Quantity,
		QuantityBefore: inventory.Quantity,
		QuantityAfter:  inventory.Quantity + req.Quantity,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		Notes:          &req.Notes,
		CreatedAt:      time.Now(),
	}

	// Save movement
	if err := s.repo.CreateMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf("failed to create movement: %w", err)
	}

	return &model.MovementResponse{
		ID:                movement.ID,
		InventoryID:       movement.InventoryID,
		BookID:            inventory.BookID,
		WarehouseLocation: inventory.WarehouseLocation,
		MovementType:      movement.MovementType,
		Quantity:          movement.Quantity,
		QuantityBefore:    movement.QuantityBefore,
		QuantityAfter:     movement.QuantityAfter,
		ReferenceType:     movement.ReferenceType,
		ReferenceID:       movement.ReferenceID,
		Notes:             movement.Notes,
		CreatedBy:         movement.CreatedBy,
		CreatedAt:         movement.CreatedAt,
	}, nil
}

// ListMovements implements Service.ListMovements
func (s *InventoryService) ListMovements(ctx context.Context, req model.ListMovementsRequest) (*model.ListMovementsResponse, error) {
	movements, totalItems, err := s.repo.ListMovements(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list movements: %w", err)
	}

	// Fetch inventory details for each movement (for book_id and warehouse)
	// In production, this could be optimized with JOIN query
	items := make([]model.MovementResponse, 0, len(movements))
	for _, m := range movements {
		inventory, err := s.repo.GetByID(ctx, m.InventoryID)
		if err != nil {
			// Skip if inventory not found (shouldn't happen)
			continue
		}

		items = append(items, model.MovementResponse{
			ID:                m.ID,
			InventoryID:       m.InventoryID,
			BookID:            inventory.BookID,
			WarehouseLocation: inventory.WarehouseLocation,
			MovementType:      m.MovementType,
			Quantity:          m.Quantity,
			QuantityBefore:    m.QuantityBefore,
			QuantityAfter:     m.QuantityAfter,
			ReferenceType:     m.ReferenceType,
			ReferenceID:       m.ReferenceID,
			Notes:             m.Notes,
			CreatedBy:         m.CreatedBy,
			CreatedAt:         m.CreatedAt,
		})
	}

	// Calculate pagination
	totalPages := (totalItems + req.Limit - 1) / req.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.ListMovementsResponse{
		Items:      items,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       req.Page,
		Limit:      req.Limit,
	}, nil
}

// GetMovementStats implements Service.GetMovementStats
func (s *InventoryService) GetMovementStats(ctx context.Context, bookID uuid.UUID) (*model.MovementStatsResponse, error) {
	stats, err := s.repo.GetMovementStatsForBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movement stats: %w", err)
	}

	return stats, nil
}

// GetInventoryDashboard implements Service.GetInventoryDashboard
func (s *InventoryService) GetInventoryDashboard(ctx context.Context, req model.DashboardRequest) (*model.InventoryDashboardResponse, error) {
	// Get summary metrics
	summary, err := s.repo.GetDashboardMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	// Get warehouse metrics
	warehouseMetrics, err := s.repo.GetWarehouseMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse metrics: %w", err)
	}

	// Get low stock items
	lowStockItems, err := s.repo.GetLowStockItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock items: %w", err)
	}

	// Get out of stock items
	outOfStockItems, err := s.repo.GetOutOfStockItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get out of stock items: %w", err)
	}

	// Get reserved analysis
	reservedAnalysis, err := s.repo.GetReservedAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get reserved analysis: %w", err)
	}

	// Get movement trends (last 7 days)
	movementTrends, err := s.repo.GetMovementTrends(ctx, 7)
	if err != nil {
		return nil, fmt.Errorf("failed to get movement trends: %w", err)
	}

	// Calculate warehouse health scores
	warehouseHealth := make(map[string]float64)
	for _, wh := range warehouseMetrics {
		warehouseHealth[wh.Warehouse] = wh.HealthScore
	}

	return &model.InventoryDashboardResponse{
		Summary:          *summary,
		ByWarehouse:      warehouseMetrics,
		LowStockItems:    lowStockItems,
		OutOfStockItems:  outOfStockItems,
		ReservedAnalysis: *reservedAnalysis,
		MovementTrends:   movementTrends,
		WarehouseHealth:  warehouseHealth,
		Timestamp:        time.Now(),
	}, nil
}

// GetLowStockAlerts implements Service.GetLowStockAlerts
func (s *InventoryService) GetLowStockAlerts(ctx context.Context) ([]model.LowStockItem, error) {
	items, err := s.repo.GetLowStockItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock alerts: %w", err)
	}
	return items, nil
}

// GetOutOfStockItems implements Service.GetOutOfStockItems
func (s *InventoryService) GetOutOfStockItems(ctx context.Context) ([]model.OutOfStockItem, error) {
	items, err := s.repo.GetOutOfStockItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get out of stock items: %w", err)
	}
	return items, nil
}

// GetInventoryValue implements Service.GetInventoryValue
// Note: This requires JOIN with books table for pricing
func (s *InventoryService) GetInventoryValue(ctx context.Context) (*model.InventoryValueResponse, error) {
	// TODO: Implement when books domain is available
	// This would join inventories with books to get pricing info
	return nil, fmt.Errorf("not implemented: requires books domain integration")
}
