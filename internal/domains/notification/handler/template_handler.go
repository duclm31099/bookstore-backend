package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared/response"
	"bookstore-backend/pkg/logger"
)

// ================================================
// TEMPLATE HANDLER (Admin Only)
// ================================================

type templateHandler struct {
	templateService service.TemplateService
}

func NewTemplateHandler(templateService service.TemplateService) TemplateHandler {
	return &templateHandler{
		templateService: templateService,
	}
}

// ================================================
// CREATE TEMPLATE
// POST /api/v1/admin/notification-templates
// ================================================

func (h *templateHandler) CreateTemplate(c *gin.Context) {
	// 1. GET ADMIN ID FROM AUTH CONTEXT
	adminID, err := getUserIDFromContext(c)
	logger.Info("adminID", map[string]interface{}{
		"adminID": adminID,
		"err":     err,
	})
	if err != nil {
		logger.Error("adminID error", err)
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE REQUEST BODY
	var req model.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. VALIDATE REQUEST
	if err := h.validateCreateTemplateRequest(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// 4. CALL SERVICE
	template, err := h.templateService.CreateTemplate(c.Request.Context(), adminID, req)
	if err != nil {
		if err == model.ErrTemplateCodeExists {
			response.Error(c, http.StatusConflict, "Template code already exists", err.Error())
			return
		}
		logger.Error("Failed to create template", err)
		response.Error(c, http.StatusInternalServerError, "Failed to create template", err.Error())
		return
	}

	// 5. RETURN RESPONSE
	response.Success(c, http.StatusCreated, "Template created successfully", template)
}

// ================================================
// GET TEMPLATE
// GET /api/v1/admin/notification-templates/:id
// ================================================

func (h *templateHandler) GetTemplate(c *gin.Context) {
	// 1. PARSE TEMPLATE ID
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid template ID", err.Error())
		return
	}

	// 2. CALL SERVICE
	template, err := h.templateService.GetTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		if err == model.ErrTemplateNotFound {
			response.Error(c, http.StatusNotFound, "Template not found", err.Error())
			return
		}
		logger.Error("Failed to get template", err)
		response.Error(c, http.StatusInternalServerError, "Failed to get template", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Template retrieved successfully", template)
}

// ================================================
// LIST TEMPLATES
// GET /api/v1/admin/notification-templates
// ================================================

func (h *templateHandler) ListTemplates(c *gin.Context) {
	// 1. PARSE QUERY PARAMETERS
	var category *string
	if categoryParam := c.Query("category"); categoryParam != "" {
		category = &categoryParam
	}

	var isActive *bool
	if isActiveParam := c.Query("is_active"); isActiveParam != "" {
		active := isActiveParam == "true"
		isActive = &active
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 2. CALL SERVICE
	templates, total, err := h.templateService.ListTemplates(c.Request.Context(), category, isActive, page, pageSize)
	if err != nil {
		logger.Error("Failed to list templates", err)
		response.Error(c, http.StatusInternalServerError, "Failed to list templates", err.Error())
		return
	}

	// 3. CALCULATE PAGINATION
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// 4. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Templates retrieved successfully", map[string]interface{}{
		"templates": templates,
		"pagination": map[string]interface{}{
			"current_page":  page,
			"page_size":     pageSize,
			"total_pages":   totalPages,
			"total_records": total,
		},
	})
}

// ================================================
// UPDATE TEMPLATE
// PUT /api/v1/admin/notification-templates/:id
// ================================================

func (h *templateHandler) UpdateTemplate(c *gin.Context) {
	// 1. GET ADMIN ID FROM AUTH CONTEXT
	adminID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE TEMPLATE ID
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid template ID", err.Error())
		return
	}

	// 3. PARSE REQUEST BODY
	var req model.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 4. CALL SERVICE
	template, err := h.templateService.UpdateTemplate(c.Request.Context(), adminID, templateID, req)
	if err != nil {
		if err == model.ErrTemplateNotFound {
			response.Error(c, http.StatusNotFound, "Template not found", err.Error())
			return
		}
		logger.Error("Failed to update template", err)
		response.Error(c, http.StatusInternalServerError, "Failed to update template", err.Error())
		return
	}

	// 5. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Template updated successfully", template)
}

// ================================================
// DELETE TEMPLATE
// DELETE /api/v1/admin/notification-templates/:id
// ================================================

func (h *templateHandler) DeleteTemplate(c *gin.Context) {
	// 1. PARSE TEMPLATE ID
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid template ID", err.Error())
		return
	}

	// 2. CALL SERVICE
	if err := h.templateService.DeleteTemplate(c.Request.Context(), templateID); err != nil {
		if err == model.ErrTemplateNotFound {
			response.Error(c, http.StatusNotFound, "Template not found", err.Error())
			return
		}
		logger.Error("Failed to delete template", err)
		response.Error(c, http.StatusInternalServerError, "Failed to delete template", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Template deleted successfully", nil)
}

// ================================================
// VALIDATION HELPERS
// ================================================

func (h *templateHandler) validateCreateTemplateRequest(req *model.CreateTemplateRequest) error {
	// Validate code format
	if req.Code == "" {
		return fmt.Errorf("code is required")
	}

	// Validate name
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate category
	validCategories := []string{
		model.CategoryTransactional,
		model.CategoryMarketing,
		model.CategorySystem,
	}
	isValidCategory := false
	for _, cat := range validCategories {
		if req.Category == cat {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return fmt.Errorf("invalid category: must be one of %v", validCategories)
	}

	// Validate language
	if req.Language == "" {
		return fmt.Errorf("language is required")
	}

	// Validate default channels
	if len(req.DefaultChannels) == 0 {
		return fmt.Errorf("at least one default channel is required")
	}

	validChannels := []string{
		model.ChannelInApp,
		model.ChannelEmail,
		model.ChannelPush,
		model.ChannelSMS,
	}

	for _, channel := range req.DefaultChannels {
		isValid := false
		for _, vc := range validChannels {
			if channel == vc {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid channel: %s", channel)
		}
	}

	// Validate priority
	if req.DefaultPriority < 1 || req.DefaultPriority > 3 {
		return fmt.Errorf("default_priority must be between 1 and 3")
	}

	return nil
}
