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
// CAMPAIGN HANDLER (Admin Only)
// ================================================

type campaignHandler struct {
	campaignService service.CampaignService
}

// ✅ MAKE SURE THIS FUNCTION EXISTS AND IS EXPORTED
func NewCampaignHandler(campaignService service.CampaignService) CampaignHandler {
	return &campaignHandler{
		campaignService: campaignService,
	}
}

// ================================================
// CREATE CAMPAIGN
// POST /api/v1/admin/notification-campaigns
// ================================================

func (h *campaignHandler) CreateCampaign(c *gin.Context) {
	// 1. GET ADMIN ID FROM AUTH CONTEXT
	adminID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE REQUEST BODY
	var req model.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. VALIDATE REQUEST
	if err := h.validateCreateCampaignRequest(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// 4. CALL SERVICE
	campaign, err := h.campaignService.CreateCampaign(c.Request.Context(), adminID, req)
	if err != nil {
		if err == model.ErrTemplateNotFound {
			response.Error(c, http.StatusNotFound, "Template not found", err.Error())
			return
		}
		if err == model.ErrTemplateInactive {
			response.Error(c, http.StatusBadRequest, "Template is not active", err.Error())
			return
		}
		logger.Error("Failed to create campaign", err)
		response.Error(c, http.StatusInternalServerError, "Failed to create campaign", err.Error())
		return
	}

	// 5. RETURN RESPONSE
	response.Success(c, http.StatusCreated, "Campaign created successfully", campaign)
}

// ================================================
// GET CAMPAIGN
// GET /api/v1/admin/notification-campaigns/:id
// ================================================

func (h *campaignHandler) GetCampaign(c *gin.Context) {
	// 1. PARSE CAMPAIGN ID
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid campaign ID", err.Error())
		return
	}

	// 2. CALL SERVICE
	campaign, err := h.campaignService.GetCampaignByID(c.Request.Context(), campaignID)
	if err != nil {
		if err == model.ErrCampaignNotFound {
			response.Error(c, http.StatusNotFound, "Campaign not found", err.Error())
			return
		}
		logger.Error("Failed to get campaign", err)
		response.Error(c, http.StatusInternalServerError, "Failed to get campaign", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Campaign retrieved successfully", campaign)
}

// ================================================
// LIST CAMPAIGNS
// GET /api/v1/admin/notification-campaigns
// ================================================

func (h *campaignHandler) ListCampaigns(c *gin.Context) {
	// 1. PARSE QUERY PARAMETERS
	var status *string
	if statusParam := c.Query("status"); statusParam != "" {
		status = &statusParam
	}

	var createdBy *uuid.UUID
	if createdByParam := c.Query("created_by"); createdByParam != "" {
		if id, err := uuid.Parse(createdByParam); err == nil {
			createdBy = &id
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 2. CALL SERVICE
	campaigns, total, err := h.campaignService.ListCampaigns(c.Request.Context(), status, createdBy, page, pageSize)
	if err != nil {
		logger.Error("Failed to list campaigns", err)
		response.Error(c, http.StatusInternalServerError, "Failed to list campaigns", err.Error())
		return
	}

	// 3. CALCULATE PAGINATION
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// 4. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Campaigns retrieved successfully", map[string]interface{}{
		"campaigns": campaigns,
		"pagination": map[string]interface{}{
			"current_page":  page,
			"page_size":     pageSize,
			"total_pages":   totalPages,
			"total_records": total,
		},
	})
}

// ================================================
// START CAMPAIGN
// POST /api/v1/admin/notification-campaigns/:id/start
// ================================================

func (h *campaignHandler) StartCampaign(c *gin.Context) {
	// 1. PARSE CAMPAIGN ID
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid campaign ID", err.Error())
		return
	}

	// 2. CALL SERVICE
	if err := h.campaignService.StartCampaign(c.Request.Context(), campaignID); err != nil {
		if err == model.ErrCampaignNotFound {
			response.Error(c, http.StatusNotFound, "Campaign not found", err.Error())
			return
		}
		logger.Error("Failed to start campaign", err)
		response.Error(c, http.StatusBadRequest, "Failed to start campaign", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Campaign started successfully", nil)
}

// ================================================
// CANCEL CAMPAIGN
// POST /api/v1/admin/notification-campaigns/:id/cancel
// ================================================

func (h *campaignHandler) CancelCampaign(c *gin.Context) {
	// 1. PARSE CAMPAIGN ID
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid campaign ID", err.Error())
		return
	}

	// 2. CALL SERVICE
	if err := h.campaignService.CancelCampaign(c.Request.Context(), campaignID); err != nil {
		if err == model.ErrCampaignNotFound {
			response.Error(c, http.StatusNotFound, "Campaign not found", err.Error())
			return
		}
		logger.Error("Failed to cancel campaign", err)
		response.Error(c, http.StatusBadRequest, "Failed to cancel campaign", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Campaign cancelled successfully", nil)
}

// ================================================
// VALIDATION HELPERS
// ================================================

func (h *campaignHandler) validateCreateCampaignRequest(req *model.CreateCampaignRequest) error {
	// Validate name
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate template code
	if req.TemplateCode == "" {
		return fmt.Errorf("template_code is required")
	}

	// Validate target type
	validTargetTypes := []string{
		model.TargetTypeAllUsers,
		model.TargetTypeSegment,
		model.TargetTypeSpecificUsers,
	}
	isValidTargetType := false
	for _, tt := range validTargetTypes {
		if req.TargetType == tt {
			isValidTargetType = true
			break
		}
	}
	if !isValidTargetType {
		return fmt.Errorf("invalid target_type: must be one of %v", validTargetTypes)
	}

	// Validate channels
	if len(req.Channels) == 0 {
		return fmt.Errorf("at least one channel is required")
	}

	validChannels := []string{
		model.ChannelInApp,
		model.ChannelEmail,
		model.ChannelPush,
		model.ChannelSMS,
	}

	for _, channel := range req.Channels {
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

	// Validate template data
	if req.TemplateData == nil || len(req.TemplateData) == 0 {
		return fmt.Errorf("template_data is required")
	}

	return nil
}
