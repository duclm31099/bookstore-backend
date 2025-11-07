package handler

import (
	"bookstore-backend/internal/domains/inventory/model"
	"bookstore-backend/internal/domains/inventory/service"
	"bookstore-backend/internal/shared/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service service.ServiceInterface
}

// NewHandler creates a new inventory handler
func NewHandler(service service.ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateInventory(c *gin.Context) {
	var req model.CreateInventoryRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Call service layer
	inventories, err := h.service.CreateInventory(c.Request.Context(), req)
	if err != nil {
		// Handle different error types
		switch {
		case model.IsValidationError(err):
			response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		case errors.Is(err, model.ErrInventoryAlreadyExists):
			response.Error(c, http.StatusConflict, "Inventory already exists", err.Error())
		case errors.Is(err, model.ErrBookNotFound):
			response.Error(c, http.StatusNotFound, "Book not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to create inventory", err.Error())
		}
		return
	}

	// Success response
	response.Success(c, http.StatusCreated, "Inventory created successfully", inventories)
}

// GetInventoryByID handles GET /api/v1/inventories/:id
// @Summary Get inventory by ID
func (h *Handler) GetInventoryByID(c *gin.Context) {
	// Parse UUID from path parameter
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid inventory ID format", err.Error())
		return
	}

	// Call service layer
	res, err := h.service.GetInventoryByID(c.Request.Context(), id)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get inventory", err.Error())
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Inventory retrieved successfully", res)
}

// SearchInventory handles GET /api/v1/inventories/search?book_id=xxx&warehouse_location=HN
// @Summary Search inventory by book and warehouse
// @Description Retrieves inventory by unique combination of book_id and warehouse_location
func (h *Handler) SearchInventory(c *gin.Context) {
	var req model.SearchInventoryRequest

	// Bind and validate query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	// Call service layer
	res, err := h.service.SearchInventory(c.Request.Context(), req)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to search inventory", err.Error())
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Inventory found", res)
}

// ListInventories handles GET /api/v1/inventories?page=1&limit=20&book_id=xxx&warehouse_location=HN&is_low_stock=true
// @Summary List inventories with pagination and filters
// @Description Retrieves paginated list of inventories with optional filters
// @Param page query int true "Page number (min: 1)"
// @Param limit query int true "Items per page (min: 1, max: 100)"
// @Param book_id query string false "Filter by Book ID (UUID)"
// @Param warehouse_location query string false "Filter by Warehouse" Enums(HN, HCM, DN, CT)
// @Param is_low_stock query bool false "Filter by low stock status"
func (h *Handler) ListInventories(c *gin.Context) {
	var req model.ListInventoryRequest

	// Bind and validate query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	// Call service layer
	result, err := h.service.ListInventories(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list inventories", err.Error())
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Inventories retrieved successfully", result)
}

func (h *Handler) UpdateInventory(c *gin.Context) {
	// Parse UUID from path parameter
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid inventory ID format", err.Error())
		return
	}

	var req model.UpdateInventoryRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Call service layer
	res, err := h.service.UpdateInventory(c.Request.Context(), id, req)
	if err != nil {
		// Handle different error types
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		case model.IsOptimisticLockError(err):
			response.Error(c, http.StatusConflict, "Version conflict - inventory was modified by another transaction", err.Error())
		case model.IsValidationError(err):
			response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to update inventory", err.Error())
		}
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Inventory updated successfully", res)
}

func (h *Handler) DeleteInventory(c *gin.Context) {
	// Parse UUID from path parameter
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid inventory ID format", err.Error())
		return
	}

	// Call service layer
	err = h.service.DeleteInventory(c.Request.Context(), id)
	if err != nil {
		// Handle different error types
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		case errors.Is(err, model.ErrCannotDeleteNonEmptyInventory):
			response.Error(c, http.StatusConflict, "Cannot delete non-empty inventory", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to delete inventory", err.Error())
		}
		return
	}

	// Success response - 204 No Content
	c.Status(http.StatusNoContent)
}

// ReserveStock handles POST /api/v1/inventories/reserve
// @Summary Reserve stock for pending order
// @Description Atomically reserves available stock. Creates audit trail in inventory_movements.
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body ReserveStockRequest true "Reserve Stock Request"
// @Success 200 {object} SuccessResponse{data=ReserveStockResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Insufficient stock"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/reserve [post]
func (h *Handler) ReserveStock(c *gin.Context) {
	var req model.ReserveStockRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.ReserveStock(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		case model.IsInsufficientStockError(err):
			response.Error(c, http.StatusConflict, "Insufficient stock available", err.Error())
		case model.IsValidationError(err):
			response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to reserve stock", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Stock reserved successfully", result)
}

// ReleaseStock handles POST /api/v1/inventories/release
// @Summary Release reserved stock
// @Description Releases previously reserved stock (order cancelled/expired). Creates audit trail.
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body ReleaseStockRequest true "Release Stock Request"
// @Success 200 {object} SuccessResponse{data=ReleaseStockResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Invalid release quantity"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/release [post]
func (h *Handler) ReleaseStock(c *gin.Context) {
	var req model.ReleaseStockRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.ReleaseStock(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		case errors.Is(err, model.ErrInvalidReleaseQuantity):
			response.Error(c, http.StatusConflict, "Cannot release more than reserved", err.Error())
		case model.IsValidationError(err):
			response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to release stock", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Stock released successfully", result)
}

// CheckAvailability handles POST /api/v1/inventories/check-availability
// @Summary Check stock availability for multiple items
// @Description Checks if items are fulfillable from inventory across warehouses
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body CheckAvailabilityRequest true "Check Availability Request"
// @Success 200 {object} SuccessResponse{data=CheckAvailabilityResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/check-availability [post]
func (h *Handler) CheckAvailability(c *gin.Context) {
	var req model.CheckAvailabilityRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.CheckAvailability(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to check availability", err.Error())
		return
	}

	// Return 200 even if not fulfillable (this is informational, not an error)
	response.Success(c, http.StatusOK, "Availability check completed", result)
}

// GetStockSummary handles GET /api/v1/inventories/summary?book_id=xxx
// @Summary Get total stock summary for a book
// @Description Retrieves total available stock across all warehouses for a specific book
// @Tags Inventory
// @Produce json
// @Param book_id query string true "Book ID (UUID)"
// @Success 200 {object} SuccessResponse{data=StockSummaryResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/summary [get]
func (h *Handler) GetStockSummary(c *gin.Context) {
	var req model.StockSummaryRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	result, err := h.service.GetStockSummary(c.Request.Context(), req.BookID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "No inventory found for this book", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get stock summary", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Stock summary retrieved successfully", result)
}

// CreateMovement handles POST /api/v1/inventories/:inventory_id/movements
// @Summary Create manual inventory adjustment
// @Description Records manual inventory adjustment with reason and audit trail
// @Tags Inventory
// @Accept json
// @Produce json
// @Param inventory_id path string true "Inventory ID (UUID)"
// @Param request body CreateMovementRequest true "Create Movement Request"
// @Success 201 {object} SuccessResponse{data=MovementResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/{inventory_id}/movements [post]
func (h *Handler) CreateMovement(c *gin.Context) {
	inventoryIDParam := c.Param("inventory_id")
	inventoryID, err := uuid.Parse(inventoryIDParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid inventory ID format", err.Error())
		return
	}

	var req model.CreateMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	req.InventoryID = inventoryID
	result, err := h.service.CreateMovement(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		case model.IsMovementError(err):
			response.Error(c, http.StatusBadRequest, "Invalid movement details", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to create movement", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Movement recorded successfully", result)
}

// ListMovements handles GET /api/v1/inventories/movements?page=1&limit=20&inventory_id=xxx&movement_type=inbound
// @Summary List inventory movements with filters
// @Description Lists all inventory movements (audit trail) with pagination
// @Tags Inventory
// @Produce json
// @Param page query int true "Page number (min: 1)"
// @Param limit query int true "Items per page (min: 1, max: 100)"
// @Param inventory_id query string false "Filter by Inventory ID"
// @Param movement_type query string false "Filter by Movement Type" Enums(inbound,outbound,adjustment,reserve,release,return)
// @Param reference_type query string false "Filter by Reference Type" Enums(order,purchase,manual,return)
// @Success 200 {object} SuccessResponse{data=ListMovementsResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/movements [get]
func (h *Handler) ListMovements(c *gin.Context) {
	var req model.ListMovementsRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	result, err := h.service.ListMovements(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list movements", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Movements retrieved successfully", result)
}

// GetMovementStats handles GET /api/v1/inventories/movements/stats?book_id=xxx
// @Summary Get movement statistics for a book
// @Description Gets aggregated movement statistics across all warehouses for a specific book
// @Tags Inventory
// @Produce json
// @Param book_id query string true "Book ID (UUID)"
// @Success 200 {object} SuccessResponse{data=MovementStatsResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/movements/stats [get]
func (h *Handler) GetMovementStats(c *gin.Context) {
	bookIDParam := c.Query("book_id")
	bookID, err := uuid.Parse(bookIDParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book ID format", err.Error())
		return
	}

	result, err := h.service.GetMovementStats(c.Request.Context(), bookID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get movement stats", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Movement statistics retrieved successfully", result)
}

// GetInventoryDashboard handles GET /api/v1/inventories/dashboard
// @Summary Get inventory dashboard metrics
// @Description Returns comprehensive dashboard with metrics, low stock alerts, trends
// @Tags Inventory
// @Produce json
// @Success 200 {object} SuccessResponse{data=InventoryDashboardResponse}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/dashboard [get]
func (h *Handler) GetInventoryDashboard(c *gin.Context) {
	var req model.DashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	result, err := h.service.GetInventoryDashboard(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get dashboard", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Dashboard retrieved successfully", result)
}

// GetLowStockAlerts handles GET /api/v1/inventories/alerts/low-stock
// @Summary Get low stock alerts
// @Description Returns all items below low stock threshold
// @Tags Inventory
// @Produce json
// @Success 200 {object} SuccessResponse{data=[]LowStockItem}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/alerts/low-stock [get]
func (h *Handler) GetLowStockAlerts(c *gin.Context) {
	items, err := h.service.GetLowStockAlerts(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get low stock alerts", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Low stock alerts retrieved", items)
}

// GetOutOfStockItems handles GET /api/v1/inventories/alerts/out-of-stock
// @Summary Get out of stock items
// @Description Returns all items with zero quantity
// @Tags Inventory
// @Produce json
// @Success 200 {object} SuccessResponse{data=[]OutOfStockItem}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/inventories/alerts/out-of-stock [get]
func (h *Handler) GetOutOfStockItems(c *gin.Context) {
	items, err := h.service.GetOutOfStockItems(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get out of stock items", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Out of stock items retrieved", items)
}
