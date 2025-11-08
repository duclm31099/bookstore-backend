// internal/domains/inventory/handler/handler.go
package handler

import (
	"bookstore-backend/internal/domains/inventory/model"
	"bookstore-backend/internal/domains/inventory/service"
	"bookstore-backend/internal/shared/response"
	"errors"
	"net/http"
	"strconv"

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

// ========================================
// INVENTORY CRUD HANDLERS
// ========================================

// CreateInventory handles POST /api/v1/inventories
// @Summary Create new inventory
// @Description Creates inventory for one warehouse or all warehouses
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body model.CreateInventoryRequest true "Create Inventory Request"
// @Success 201 {object} response.SuccessResponse{data=[]model.InventoryResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Router /api/v1/inventories [post]
func (h *Handler) CreateInventory(c *gin.Context) {
	var req model.CreateInventoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	inventories, err := h.service.CreateInventory(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsValidationError(err):
			response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		case errors.Is(err, model.ErrInventoryAlreadyExists):
			response.Error(c, http.StatusConflict, "Inventory already exists", err.Error())
		case errors.Is(err, model.ErrBookNotFound):
			response.Error(c, http.StatusNotFound, "Book not found", err.Error())
		case errors.Is(err, model.ErrWarehouseNotFound):
			response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to create inventory", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Inventory created successfully", inventories)
}

// GetInventoryByWarehouseAndBook handles GET /api/v1/inventories/:warehouse_id/:book_id
// @Summary Get inventory by warehouse and book
// @Description Retrieves specific inventory by composite key
// @Tags Inventory
// @Produce json
// @Param warehouse_id path string true "Warehouse ID (UUID)"
// @Param book_id path string true "Book ID (UUID)"
// @Success 200 {object} response.SuccessResponse{data=model.InventoryResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/{warehouse_id}/{book_id} [get]
func (h *Handler) GetInventoryByWarehouseAndBook(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("warehouse_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	bookID, err := uuid.Parse(c.Param("book_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book ID format", err.Error())
		return
	}

	res, err := h.service.GetInventoryByWarehouseAndBook(c.Request.Context(), warehouseID, bookID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get inventory", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Inventory retrieved successfully", res)
}

// ListInventories handles GET /api/v1/inventories
// @Summary List inventories with pagination and filters
// @Description Retrieves paginated list of inventories
// @Tags Inventory
// @Produce json
// @Param page query int true "Page number (min: 1)" default(1)
// @Param limit query int true "Items per page (1-100)" default(20)
// @Param book_id query string false "Filter by Book ID (UUID)"
// @Param warehouse_id query string false "Filter by Warehouse ID (UUID)"
// @Param is_low_stock query bool false "Filter by low stock status"
// @Param has_available_stock query bool false "Filter items with available stock"
// @Success 200 {object} response.SuccessResponse{data=model.ListInventoryResponse}
// @Failure 400 {object} response.ErrorResponse
// @Router /api/v1/inventories [get]
func (h *Handler) ListInventories(c *gin.Context) {
	var req model.ListInventoryRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	result, err := h.service.ListInventories(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list inventories", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Inventories retrieved successfully", result)
}

// UpdateInventory handles PATCH /api/v1/inventories/:warehouse_id/:book_id
// @Summary Update inventory
// @Description Updates inventory with optimistic locking
// @Tags Inventory
// @Accept json
// @Produce json
// @Param warehouse_id path string true "Warehouse ID (UUID)"
// @Param book_id path string true "Book ID (UUID)"
// @Param request body model.UpdateInventoryRequest true "Update Request"
// @Success 200 {object} response.SuccessResponse{data=model.InventoryResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Version conflict"
// @Router /api/v1/inventories/{warehouse_id}/{book_id} [patch]
func (h *Handler) UpdateInventory(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("warehouse_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	bookID, err := uuid.Parse(c.Param("book_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book ID format", err.Error())
		return
	}

	var req model.UpdateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	res, err := h.service.UpdateInventory(c.Request.Context(), warehouseID, bookID, req)
	if err != nil {
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

	response.Success(c, http.StatusOK, "Inventory updated successfully", res)
}

// DeleteInventory handles DELETE /api/v1/inventories/:warehouse_id/:book_id
// @Summary Delete inventory
// @Description Deletes inventory (only if quantity = 0 and reserved = 0)
// @Tags Inventory
// @Param warehouse_id path string true "Warehouse ID (UUID)"
// @Param book_id path string true "Book ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Cannot delete non-empty inventory"
// @Router /api/v1/inventories/{warehouse_id}/{book_id} [delete]
func (h *Handler) DeleteInventory(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("warehouse_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	bookID, err := uuid.Parse(c.Param("book_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book ID format", err.Error())
		return
	}

	err = h.service.DeleteInventory(c.Request.Context(), warehouseID, bookID)
	if err != nil {
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

	c.Status(http.StatusNoContent)
}

// ========================================
// STOCK OPERATIONS HANDLERS (FR-INV-003)
// ========================================

// ReserveStock handles POST /api/v1/inventories/reserve
// @Summary Reserve stock for checkout
// @Description Reserves stock with pessimistic locking. Expires in 15 minutes.
// @Tags Stock Operations
// @Accept json
// @Produce json
// @Param request body model.ReserveStockRequest true "Reserve Request"
// @Success 200 {object} response.SuccessResponse{data=model.ReserveStockResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Insufficient stock"
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
			response.Error(c, http.StatusNotFound, "Inventory or warehouse not found", err.Error())
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
// @Description Releases stock (order cancelled/timeout). Auto-called by background job after 15min.
// @Tags Stock Operations
// @Accept json
// @Produce json
// @Param request body model.ReleaseStockRequest true "Release Request"
// @Success 200 {object} response.SuccessResponse{data=model.ReleaseStockResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
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
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to release stock", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Stock released successfully", result)
}

// CompleteSale handles POST /api/v1/inventories/complete-sale
// @Summary Complete sale after payment success
// @Description Decreases both quantity and reserved. Creates audit log.
// @Tags Stock Operations
// @Accept json
// @Produce json
// @Param request body model.CompleteSaleRequest true "Complete Sale Request"
// @Success 200 {object} response.SuccessResponse{data=model.CompleteSaleResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/complete-sale [post]
func (h *Handler) CompleteSale(c *gin.Context) {
	var req model.CompleteSaleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.CompleteSale(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to complete sale", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Sale completed successfully", result)
}

// ========================================
// WAREHOUSE SELECTION HANDLERS (FR-INV-002)
// ========================================

// FindOptimalWarehouse handles POST /api/v1/inventories/find-warehouse
// @Summary Find nearest warehouse with stock
// @Description Uses Haversine formula to find closest warehouse with sufficient stock
// @Tags Warehouse Selection
// @Accept json
// @Produce json
// @Param request body model.FindWarehouseRequest true "Find Warehouse Request"
// @Success 200 {object} response.SuccessResponse{data=model.WarehouseRecommendation}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse "No warehouse with sufficient stock"
// @Router /api/v1/inventories/find-warehouse [post]
func (h *Handler) FindOptimalWarehouse(c *gin.Context) {
	var req model.FindWarehouseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.FindOptimalWarehouse(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to find warehouse", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouse found", result)
}

// CheckAvailability handles POST /api/v1/inventories/check-availability
// @Summary Check stock availability for order
// @Description Checks if all items can be fulfilled and recommends warehouse
// @Tags Warehouse Selection
// @Accept json
// @Produce json
// @Param request body model.CheckAvailabilityRequest true "Check Availability Request"
// @Success 200 {object} response.SuccessResponse{data=model.CheckAvailabilityResponse}
// @Failure 400 {object} response.ErrorResponse
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

	// Return 200 even if not fulfillable (informational, not an error)
	response.Success(c, http.StatusOK, "Availability check completed", result)
}

// GetStockSummary handles GET /api/v1/inventories/summary/:book_id
// @Summary Get total stock summary for book
// @Description Aggregates stock across all warehouses using VIEW books_total_stock
// @Tags Inventory
// @Produce json
// @Param book_id path string true "Book ID (UUID)"
// @Success 200 {object} response.SuccessResponse{data=model.StockSummaryResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/summary/{book_id} [get]
func (h *Handler) GetStockSummary(c *gin.Context) {
	bookID, err := uuid.Parse(c.Param("book_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book ID format", err.Error())
		return
	}

	result, err := h.service.GetStockSummary(c.Request.Context(), bookID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "No inventory found for this book", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get stock summary", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Stock summary retrieved", result)
}

// ========================================
// STOCK ADJUSTMENT HANDLERS (FR-INV-005)
// ========================================

// AdjustStock handles POST /api/v1/inventories/adjust
// @Summary Manual stock adjustment (admin only)
// @Description Adjusts inventory quantity with reason for audit trail
// @Tags Stock Adjustment
// @Accept json
// @Produce json
// @Param request body model.AdjustStockRequest true "Adjust Stock Request"
// @Success 200 {object} response.SuccessResponse{data=model.AdjustStockResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Version conflict"
// @Router /api/v1/inventories/adjust [post]
func (h *Handler) AdjustStock(c *gin.Context) {
	var req model.AdjustStockRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// TODO: Get changed_by from JWT claims
	// req.ChangedBy = c.GetString("user_id")

	result, err := h.service.AdjustStock(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		case model.IsOptimisticLockError(err):
			response.Error(c, http.StatusConflict, "Version conflict", err.Error())
		case model.IsValidationError(err):
			response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to adjust stock", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Stock adjusted successfully", result)
}

// RestockInventory handles POST /api/v1/inventories/restock
// @Summary Restock inventory
// @Description Adds stock from supplier with reason
// @Tags Stock Adjustment
// @Accept json
// @Produce json
// @Param request body model.RestockRequest true "Restock Request"
// @Success 200 {object} response.SuccessResponse{data=model.RestockResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/restock [post]
func (h *Handler) RestockInventory(c *gin.Context) {
	var req model.RestockRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.RestockInventory(c.Request.Context(), req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Inventory not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to restock", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Inventory restocked successfully", result)
}

// BulkUpdateStock handles POST /api/v1/inventories/bulk-update
// @Summary Bulk stock update from CSV (FR-INV-006)
// @Description Uploads CSV file and creates async job for processing
// @Tags Stock Adjustment
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file"
// @Success 202 {object} response.SuccessResponse{data=model.BulkUpdateJobResponse} "Accepted for processing"
// @Failure 400 {object} response.ErrorResponse
// @Router /api/v1/inventories/bulk-update [post]
func (h *Handler) BulkUpdateStock(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "File is required", err.Error())
		return
	}

	// Validate file type
	if file.Header.Get("Content-Type") != "text/csv" {
		response.Error(c, http.StatusBadRequest, "Only CSV files are allowed", "")
		return
	}

	// Save file temporarily
	csvPath := "/tmp/" + file.Filename
	if err := c.SaveUploadedFile(file, csvPath); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save file", err.Error())
		return
	}

	// TODO: Get user ID from JWT claims
	uploadedBy := uuid.New()

	result, err := h.service.BulkUpdateStock(c.Request.Context(), csvPath, uploadedBy)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to start bulk update", err.Error())
		return
	}

	response.Success(c, http.StatusAccepted, "Bulk update job created", result)
}

// GetBulkUpdateStatus handles GET /api/v1/inventories/bulk-update/:job_id
// @Summary Get bulk update job status
// @Description Checks processing status of CSV import job
// @Tags Stock Adjustment
// @Produce json
// @Param job_id path string true "Job ID (UUID)"
// @Success 200 {object} response.SuccessResponse{data=model.BulkUpdateStatusResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/bulk-update/{job_id} [get]
func (h *Handler) GetBulkUpdateStatus(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid job ID format", err.Error())
		return
	}

	result, err := h.service.GetBulkUpdateStatus(c.Request.Context(), jobID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Job not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get job status", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Job status retrieved", result)
}

// ========================================
// ALERTS HANDLERS (FR-INV-004)
// ========================================

// GetLowStockAlerts handles GET /api/v1/inventories/alerts/low-stock
// @Summary Get low stock alerts
// @Description Returns unresolved low stock alerts from trigger-created table
// @Tags Alerts
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]model.LowStockAlert}
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
// @Description Returns all items with quantity = 0
// @Tags Alerts
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]model.OutOfStockItem}
// @Router /api/v1/inventories/alerts/out-of-stock [get]
func (h *Handler) GetOutOfStockItems(c *gin.Context) {
	items, err := h.service.GetOutOfStockItems(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get out of stock items", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Out of stock items retrieved", items)
}

// MarkAlertResolved handles PATCH /api/v1/inventories/alerts/:alert_id/resolve
// @Summary Mark alert as resolved (admin only)
// @Description Manually resolve low stock alert (normally auto-resolved by trigger)
// @Tags Alerts
// @Param alert_id path string true "Alert ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/alerts/{alert_id}/resolve [patch]
func (h *Handler) MarkAlertResolved(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("alert_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid alert ID format", err.Error())
		return
	}

	err = h.service.MarkAlertResolved(c.Request.Context(), alertID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Alert not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to resolve alert", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// ========================================
// AUDIT & REPORTING HANDLERS (FR-INV-005)
// ========================================

// GetAuditTrail handles GET /api/v1/inventories/audit
// @Summary Get audit trail
// @Description Retrieves audit log with filters and pagination
// @Tags Audit
// @Produce json
// @Param page query int true "Page number" default(1)
// @Param limit query int true "Items per page" default(20)
// @Param warehouse_id query string false "Filter by Warehouse ID"
// @Param book_id query string false "Filter by Book ID"
// @Param action query string false "Filter by action (RESTOCK, RESERVE, RELEASE, ADJUSTMENT, SALE)"
// @Param changed_by query string false "Filter by User ID"
// @Param start_date query string false "Start date (RFC3339)"
// @Param end_date query string false "End date (RFC3339)"
// @Success 200 {object} response.SuccessResponse{data=model.AuditTrailResponse}
// @Failure 400 {object} response.ErrorResponse
// @Router /api/v1/inventories/audit [get]
func (h *Handler) GetAuditTrail(c *gin.Context) {
	var req model.AuditTrailRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	result, err := h.service.GetAuditTrail(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get audit trail", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Audit trail retrieved", result)
}

// GetInventoryHistory handles GET /api/v1/inventories/:warehouse_id/:book_id/history
// @Summary Get inventory history
// @Description Full audit history for specific warehouse+book
// @Tags Audit
// @Produce json
// @Param warehouse_id path string true "Warehouse ID (UUID)"
// @Param book_id path string true "Book ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Success 200 {object} response.SuccessResponse{data=model.InventoryHistoryResponse}
// @Failure 400 {object} response.ErrorResponse
// @Router /api/v1/inventories/{warehouse_id}/{book_id}/history [get]
func (h *Handler) GetInventoryHistory(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("warehouse_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	bookID, err := uuid.Parse(c.Param("book_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book ID format", err.Error())
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	result, err := h.service.GetInventoryHistory(c.Request.Context(), warehouseID, bookID, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get inventory history", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Inventory history retrieved", result)
}

// ExportAuditLog handles POST /api/v1/inventories/audit/export
// @Summary Export audit log to CSV/Excel
// @Description Exports audit log for compliance reporting
// @Tags Audit
// @Accept json
// @Produce json
// @Param request body model.ExportAuditRequest true "Export Request"
// @Success 200 {object} response.SuccessResponse{data=model.ExportResponse}
// @Failure 400 {object} response.ErrorResponse
// @Router /api/v1/inventories/audit/export [post]
func (h *Handler) ExportAuditLog(c *gin.Context) {
	var req model.ExportAuditRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.ExportAuditLog(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to export audit log", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Export completed", result)
}

// ========================================
// DASHBOARD & ANALYTICS HANDLERS
// ========================================

// GetDashboardSummary handles GET /api/v1/inventories/dashboard
// @Summary Get comprehensive dashboard
// @Description Returns overall metrics, warehouse breakdown, alerts, trends
// @Tags Dashboard
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=model.DashboardSummaryResponse}
// @Router /api/v1/inventories/dashboard [get]
func (h *Handler) GetDashboardSummary(c *gin.Context) {
	result, err := h.service.GetDashboardSummary(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get dashboard", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Dashboard retrieved", result)
}

// GetWarehousePerformance handles GET /api/v1/inventories/warehouses/:warehouse_id/performance
// @Summary Get warehouse performance metrics
// @Description Warehouse-specific analytics and trends
// @Tags Dashboard
// @Produce json
// @Param warehouse_id path string true "Warehouse ID (UUID)"
// @Success 200 {object} response.SuccessResponse{data=model.WarehousePerformanceResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/inventories/warehouses/{warehouse_id}/performance [get]
func (h *Handler) GetWarehousePerformance(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("warehouse_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	result, err := h.service.GetWarehousePerformance(c.Request.Context(), warehouseID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get performance metrics", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Performance metrics retrieved", result)
}

// GetMovementTrends handles GET /api/v1/inventories/trends
// @Summary Get movement trends
// @Description Daily inbound/outbound trends for charts
// @Tags Dashboard
// @Produce json
// @Param days query int false "Number of days" default(7)
// @Success 200 {object} response.SuccessResponse{data=[]model.MovementTrend}
// @Router /api/v1/inventories/trends [get]
// func (h *Handler) GetMovementTrends(c *gin.Context) {
// 	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
// 	if days < 1 || days > 365 {
// 		days = 7
// 	}

// 	result, err := h.service.GetMovementTrends(c.Request.Context(), days)
// 	if err != nil {
// 		response.Error(c, http.StatusInternalServerError, "Failed to get trends", err.Error())
// 		return
// 	}

// 	response.Success(c, http.StatusOK, "Movement trends retrieved", result)
// }

// GetReservationAnalysis handles GET /api/v1/inventories/analysis/reservations
// @Summary Get reservation analytics
// @Description Reservation rate, duration, conversion metrics
// @Tags Dashboard
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=model.ReservationAnalysisResponse}
// @Router /api/v1/inventories/analysis/reservations [get]
func (h *Handler) GetReservationAnalysis(c *gin.Context) {
	result, err := h.service.GetReservationAnalysis(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get reservation analysis", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Reservation analysis retrieved", result)
}

// GetInventoryValue handles GET /api/v1/inventories/value
// @Summary Get total inventory value
// @Description Financial reporting with value breakdown
// @Tags Dashboard
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=model.InventoryValueResponse}
// @Router /api/v1/inventories/value [get]
// func (h *Handler) GetInventoryValue(c *gin.Context) {
// 	result, err := h.service.GetInventoryValue(c.Request.Context())
// 	if err != nil {
// 		response.Error(c, http.StatusInternalServerError, "Failed to get inventory value", err.Error())
// 		return
// 	}

// 	response.Success(c, http.StatusOK, "Inventory value retrieved", result)
// }

// ========================================
// WAREHOUSE MANAGEMENT HANDLERS
// ========================================

// CreateWarehouse handles POST /api/v1/warehouses
// @Summary Create new warehouse (admin only)
// @Description Creates warehouse with location coordinates
// @Tags Warehouses
// @Accept json
// @Produce json
// @Param request body model.CreateWarehouseRequest true "Create Warehouse Request"
// @Success 201 {object} response.SuccessResponse{data=model.WarehouseResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Code already exists"
// @Router /api/v1/warehouses [post]
func (h *Handler) CreateWarehouse(c *gin.Context) {
	var req model.CreateWarehouseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.CreateWarehouse(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, model.ErrWarehouseCodeExists) {
			response.Error(c, http.StatusConflict, "Warehouse code already exists", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to create warehouse", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Warehouse created successfully", result)
}

// UpdateWarehouse handles PATCH /api/v1/warehouses/:id
// @Summary Update warehouse (admin only)
// @Description Updates warehouse with optimistic locking
// @Tags Warehouses
// @Accept json
// @Produce json
// @Param id path string true "Warehouse ID (UUID)"
// @Param request body model.UpdateWarehouseRequest true "Update Request"
// @Success 200 {object} response.SuccessResponse{data=model.WarehouseResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Version conflict"
// @Router /api/v1/warehouses/{id} [patch]
func (h *Handler) UpdateWarehouse(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	var req model.UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	result, err := h.service.UpdateWarehouse(c.Request.Context(), warehouseID, req)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
		case model.IsOptimisticLockError(err):
			response.Error(c, http.StatusConflict, "Version conflict", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to update warehouse", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Warehouse updated successfully", result)
}

// ListWarehouses handles GET /api/v1/warehouses
// @Summary List all warehouses
// @Description Retrieves warehouses with filters
// @Tags Warehouses
// @Produce json
// @Param is_active query bool false "Filter by active status"
// @Param province query string false "Filter by province"
// @Success 200 {object} response.SuccessResponse{data=[]model.WarehouseResponse}
// @Router /api/v1/warehouses [get]
func (h *Handler) ListWarehouses(c *gin.Context) {
	var req model.ListWarehousesRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	result, err := h.service.ListWarehouses(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list warehouses", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouses retrieved", result)
}

// GetWarehouseByID handles GET /api/v1/warehouses/:id
// @Summary Get warehouse by ID
// @Description Retrieves warehouse details
// @Tags Warehouses
// @Produce json
// @Param id path string true "Warehouse ID (UUID)"
// @Success 200 {object} response.SuccessResponse{data=model.WarehouseResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/warehouses/{id} [get]
func (h *Handler) GetWarehouseByID(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	result, err := h.service.GetWarehouseByID(c.Request.Context(), warehouseID)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to get warehouse", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouse retrieved", result)
}

// DeactivateWarehouse handles DELETE /api/v1/warehouses/:id
// @Summary Deactivate warehouse (admin only)
// @Description Soft deletes warehouse. Validates no stock remaining.
// @Tags Warehouses
// @Param id path string true "Warehouse ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse "Warehouse has stock"
// @Router /api/v1/warehouses/{id} [delete]
func (h *Handler) DeactivateWarehouse(c *gin.Context) {
	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID format", err.Error())
		return
	}

	err = h.service.DeactivateWarehouse(c.Request.Context(), warehouseID)
	if err != nil {
		switch {
		case model.IsNotFoundError(err):
			response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
		case errors.Is(err, model.ErrCannotDeleteWarehouseWithStock):
			response.Error(c, http.StatusConflict, "Cannot delete warehouse with existing stock", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to deactivate warehouse", err.Error())
		}
		return
	}

	c.Status(http.StatusNoContent)
}
