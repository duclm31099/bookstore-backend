package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/warehouse/model"
	"bookstore-backend/internal/domains/warehouse/service"
	"bookstore-backend/internal/shared/response"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// ==================== ADMIN CRUD ====================

// CreateWarehouse tạo kho mới (admin only)
// POST /admin/warehouses
func (h *Handler) CreateWarehouse(c *gin.Context) {
	var req model.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate basic
	if req.Name == "" || req.Address == "" || req.Province == "" {
		response.Error(c, http.StatusBadRequest, "Name, address, province are required", nil)
		return
	}

	warehouse, err := h.svc.CreateWarehouse(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create warehouse", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Warehouse created successfully", warehouse)
}

// UpdateWarehouse cập nhật thông tin kho
// PUT /admin/warehouses/:id
func (h *Handler) UpdateWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID", err.Error())
		return
	}

	var req model.UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	warehouse, err := h.svc.UpdateWarehouse(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update warehouse", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouse updated successfully", warehouse)
}

// SoftDeleteWarehouse xóa mềm kho (kiểm tra inventory trước)
// DELETE /admin/warehouses/:id
func (h *Handler) SoftDeleteWarehouse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID", err.Error())
		return
	}

	err = h.svc.SoftDeleteWarehouse(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "cannot delete warehouse with existing inventory" {
			response.Error(c, http.StatusConflict, "Cannot delete warehouse with existing inventory", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to delete warehouse", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouse deleted successfully", nil)
}

// GetWarehouseByID lấy chi tiết kho theo ID
// GET /admin/warehouses/:id hoặc /warehouses/:id
func (h *Handler) GetWarehouseByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse ID", err.Error())
		return
	}

	warehouse, err := h.svc.GetWarehouseByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouse retrieved successfully", warehouse)
}

// ListWarehouses list kho với filter và paging (admin)
// GET /admin/warehouses?keyword=&province=&is_active=&limit=&offset=
func (h *Handler) ListWarehouses(c *gin.Context) {
	keyword := c.Query("keyword")
	province := c.Query("province")
	isActiveStr := c.Query("is_active")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	var isActive *bool
	if isActiveStr != "" {
		val := isActiveStr == "true"
		isActive = &val
	}

	filter := model.ListWarehouseFilter{
		Keyword:  keyword,
		Province: province,
		IsActive: isActive,
		Limit:    limit,
		Offset:   offset,
	}

	warehouses, err := h.svc.ListWarehouses(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list warehouses", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouses retrieved successfully", warehouses)
}

// ==================== PUBLIC / INTERNAL API ====================

// GetWarehouseByCode lấy kho theo mã code
// GET /warehouses/code/:code
func (h *Handler) GetWarehouseByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.Error(c, http.StatusBadRequest, "Code is required", nil)
		return
	}

	warehouse, err := h.svc.GetWarehouseByCode(c.Request.Context(), code)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Warehouse not found", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Warehouse retrieved successfully", warehouse)
}

// ListActiveWarehouses list tất cả kho đang active (không paging)
// GET /warehouses
func (h *Handler) ListActiveWarehouses(c *gin.Context) {
	warehouses, err := h.svc.ListActiveWarehouses(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list active warehouses", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Active warehouses retrieved successfully", warehouses)
}

// FindNearestWarehouseWithStock tìm kho gần nhất còn hàng cho 1 book
// GET /warehouses/nearest-with-stock?book_id=xxx&lat=10.762622&lon=106.660172&qty=1
func (h *Handler) FindNearestWarehouseWithStock(c *gin.Context) {
	bookIDStr := c.Query("book_id")
	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	qtyStr := c.DefaultQuery("qty", "1")

	if bookIDStr == "" || latStr == "" || lonStr == "" {
		response.Error(c, http.StatusBadRequest, "book_id, lat, lon are required", nil)
		return
	}

	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book_id", err.Error())
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid latitude", err.Error())
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid longitude", err.Error())
		return
	}

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty <= 0 {
		response.Error(c, http.StatusBadRequest, "Invalid quantity", err.Error())
		return
	}

	warehouse, err := h.svc.FindNearestWarehouseWithStock(c.Request.Context(), bookID, lat, lon, qty)
	if err != nil {
		response.Error(c, http.StatusNotFound, "No warehouse with sufficient stock found", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Nearest warehouse found", warehouse)
}

// ValidateWarehouseHasStock kiểm tra xem kho cụ thể có đủ hàng không (để order service gọi)
// GET /warehouses/validate-stock?warehouse_id=xxx&book_id=xxx&qty=1
func (h *Handler) ValidateWarehouseHasStock(c *gin.Context) {
	warehouseIDStr := c.Query("warehouse_id")
	bookIDStr := c.Query("book_id")
	qtyStr := c.DefaultQuery("qty", "1")

	if warehouseIDStr == "" || bookIDStr == "" {
		response.Error(c, http.StatusBadRequest, "warehouse_id and book_id are required", nil)
		return
	}

	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid warehouse_id", err.Error())
		return
	}

	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid book_id", err.Error())
		return
	}

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty <= 0 {
		response.Error(c, http.StatusBadRequest, "Invalid quantity", err.Error())
		return
	}

	hasStock, err := h.svc.ValidateWarehouseHasStock(c.Request.Context(), warehouseID, bookID, qty)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to validate stock", err.Error())
		return
	}

	result := map[string]interface{}{
		"has_stock":    hasStock,
		"warehouse_id": warehouseID,
		"book_id":      bookID,
		"required_qty": qty,
	}

	response.Success(c, http.StatusOK, "Stock validation completed", result)
}
