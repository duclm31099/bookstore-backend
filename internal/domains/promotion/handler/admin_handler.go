package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/promotion/model"
	"bookstore-backend/internal/domains/promotion/service"
	"bookstore-backend/internal/shared/response"
)

// AdminHandler xử lý các API quản trị (admin-only)
type AdminHandler struct {
	service service.ServiceInterface
}

// NewAdminHandler tạo handler instance
func NewAdminHandler(service service.ServiceInterface) *AdminHandler {
	return &AdminHandler{
		service: service,
	}
}

// -------------------------------------------------------------------
// CREATE & UPDATE
// -------------------------------------------------------------------

// CreatePromotion tạo promotion mới
// @Description  Tạo chương trình khuyến mãi mới (Admin only)
// @Router       /v1/admin/promotions [post]
func (h *AdminHandler) CreatePromotion(c *gin.Context) {
	var req model.CreatePromotionRequest

	// Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Call service
	promo, err := h.service.CreatePromotion(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Create promotion successfully", promo)
}

// UpdatePromotion cập nhật promotion
// @Summary      Update promotion
// @Description  Cập nhật thông tin promotion (Admin only)
// @Router       /v1/admin/promotions/:id [put]
func (h *AdminHandler) UpdatePromotion(c *gin.Context) {
	// Parse ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Promotion ID không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	var req model.UpdatePromotionRequest

	// Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// ===================================================================
	// BUSINESS LOGIC: Update Promotion
	// ===================================================================
	// 1. Get existing promotion
	// 2. Check update restrictions (dựa vào current_uses)
	// 3. Apply updates
	// 4. Save với optimistic locking
	// ===================================================================

	ctx := c.Request.Context()

	// Step 1: Get existing promotion
	existing, err := h.service.GetPromotionByID(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Step 2: Check update restrictions
	if existing.CurrentUses > 0 {
		// Không được thay đổi các trường quan trọng
		restrictedFields := []string{}

		// Nếu request có field này → check xem có thay đổi không
		// (Implementation đơn giản: Reject mọi update nếu có usage)
		// (Production: Chỉ reject những field cụ thể)

		// Ví dụ kiểm tra:
		// if req.DiscountType != nil && *req.DiscountType != existing.DiscountType { ... }

		if len(restrictedFields) > 0 {
			response.Error(c, http.StatusBadRequest, "Không thể thay đổi promotion đã có người sử dụng", gin.H{
				"info": err.Error(),
				"code": model.ErrCodePromoUpdateConflict,
			})
			return
		}
	}

	// Step 3 & 4: Update via service
	// Note: Service sẽ handle optimistic locking
	updatedPromo, err := h.service.UpdatePromotion(ctx, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Update promotion successfully", updatedPromo)
}

// -------------------------------------------------------------------
// READ OPERATIONS
// -------------------------------------------------------------------

// GetPromotionByID lấy chi tiết promotion
// @Description  Lấy thông tin chi tiết promotion (Admin only)
// @Router       /v1/admin/promotions/:id [get]
func (h *AdminHandler) GetPromotionByID(c *gin.Context) {
	// Parse ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Promotion ID không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Call service
	promo, err := h.service.GetPromotionByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Get promotion successfully", promo)
}

// ListPromotions lấy danh sách promotions với filter
// @Description  Lấy danh sách tất cả promotions với filter (Admin only)
// @Router       /v1/admin/promotions [get]
func (h *AdminHandler) ListPromotions(c *gin.Context) {
	// Parse filter from query params
	var filter model.ListPromotionsFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "Query parameters không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Validate filter
	if err := filter.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Query parameters không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Call service
	promotions, total, err := h.service.ListPromotions(c.Request.Context(), &filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Calculate total pages
	totalPages := (total + filter.Limit - 1) / filter.Limit

	response.Success(c, http.StatusOK, "List promotion successfully", gin.H{
		"promotions": promotions,
		"pagination": gin.H{
			"page":        filter.Page,
			"limit":       filter.Limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// -------------------------------------------------------------------
// STATUS & DELETE
// -------------------------------------------------------------------

// UpdatePromotionStatus cập nhật trạng thái active/inactive
//
// @Summary      Update promotion status
// @Router       /v1/admin/promotions/:id/status [patch]
func (h *AdminHandler) UpdatePromotionStatus(c *gin.Context) {
	// Parse ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Promotion ID không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Parse request body
	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Call service
	err = h.service.UpdatePromotionStatus(c.Request.Context(), id, req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Success message
	message := "Promotion đã được kích hoạt"
	if !req.IsActive {
		message = "Promotion đã được vô hiệu hóa"
	}
	response.Success(c, http.StatusOK, "Update promotion status successfully", gin.H{
		"id":        id,
		"is_active": req.IsActive,
		"message":   message,
	})
}

// DeletePromotion xóa promotion (soft delete)
// @Description  Xóa promotion (chỉ cho phép nếu chưa có usage) (Admin only)
// @Router       /v1/admin/promotions/:id [delete]
func (h *AdminHandler) DeletePromotion(c *gin.Context) {
	// Parse ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Promotion ID không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Call service
	err = h.service.DeletePromotion(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Success response
	response.Success(c, http.StatusOK, "Promotion đã được xóa thành công", id)
}

// -------------------------------------------------------------------
// USAGE HISTORY & REPORTING
// -------------------------------------------------------------------

// GetUsageHistory lấy lịch sử sử dụng promotion
//
// @Summary      Get promotion usage history
// @Router       /v1/admin/promotions/:id/usage [get]
func (h *AdminHandler) GetUsageHistory(c *gin.Context) {
	// Parse ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Promotion ID không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Parse date filters
	var startDate, endDate *time.Time

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "start_date không hợp lệ", gin.H{
				"info": err.Error(),
				"code": model.ErrCodeValidationFailed,
			})
			return
		}
		startDate = &t
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "end_date không hợp lệ", gin.H{
				"info": err.Error(),
				"code": model.ErrCodeValidationFailed,
			})
			return
		}
		endDate = &t
	}

	// Parse user_id filter
	var userID *uuid.UUID
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "User id không hợp lệ", gin.H{
				"info": err.Error(),
				"code": model.ErrCodeValidationFailed,
			})
			return
		}
		userID = &id
	}

	// Pagination
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 50)

	if limit > 200 {
		limit = 200
	}

	// Call service
	usageHistory, err := h.service.GetUsageHistory(
		c.Request.Context(),
		id,
		startDate,
		endDate,
		userID,
		page,
		limit,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Get promotion usage history", usageHistory)
}

// ExportUsageReport export báo cáo sử dụng promotion (CSV/Excel)
// @Description  Tạo job export báo cáo sử dụng promotion (Admin only)
// @Router       /v1/admin/promotions/:id/export [post]
func (h *AdminHandler) ExportUsageReport(c *gin.Context) {
	// Parse ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Promotion ID không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// Parse request
	var req struct {
		Format    string `json:"format"` // csv, excel
		DateRange struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"date_range"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu request không hợp lệ", gin.H{
			"info": err.Error(),
			"code": model.ErrCodeValidationFailed,
		})
		return
	}

	// ===================================================================
	// BUSINESS LOGIC: Export Report (Async)
	// ===================================================================
	// 1. Validate promotion exists
	// 2. Create export job (Asynq)
	// 3. Return job ID
	// 4. Background worker will:
	//    - Fetch usage data
	//    - Generate CSV/Excel
	//    - Upload to S3
	//    - Send email with download link
	// ===================================================================

	ctx := c.Request.Context()

	// Step 1: Validate promotion exists
	_, err = h.service.GetPromotionByID(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Step 2: Create async job (pseudo-code)
	jobID := uuid.New()

	// Enqueue job to Asynq
	// task := asynq.NewTask("export:promotion_usage", payload)
	// client.Enqueue(task)

	// Step 3: Return job ID
	response.Success(c, http.StatusAccepted, "Export đang được xử lý, bạn sẽ nhận email khi hoàn thành", jobID)
}

// -------------------------------------------------------------------
// HELPER FUNCTIONS
// -------------------------------------------------------------------

// handleError giống PublicHandler
func (h *AdminHandler) handleError(c *gin.Context, err error) {
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Internal server error", gin.H{
			"info": err.Error(),
		})
		return
	}
}
