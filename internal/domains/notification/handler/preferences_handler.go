package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared/response"
)

// ================================================
// PREFERENCES HANDLER
// ================================================

type preferencesHandler struct {
	preferencesService service.PreferencesService
}

func NewPreferencesHandler(preferencesService service.PreferencesService) PreferencesHandler {
	return &preferencesHandler{
		preferencesService: preferencesService,
	}
}

// ================================================
// GET PREFERENCES
// GET /api/v1/notifications/preferences
// ================================================

func (h *preferencesHandler) GetPreferences(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. CALL SERVICE
	preferences, err := h.preferencesService.GetUserPreferences(c.Request.Context(), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get preferences")
		response.Error(c, http.StatusInternalServerError, "Failed to get preferences", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Preferences retrieved successfully", preferences)
}

// ================================================
// UPDATE PREFERENCES
// PUT /api/v1/notifications/preferences
// ================================================

func (h *preferencesHandler) UpdatePreferences(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE REQUEST BODY
	var req model.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. VALIDATE QUIET HOURS FORMAT
	if req.QuietHoursStart != nil {
		if !isValidTimeFormat(*req.QuietHoursStart) {
			response.Error(c, http.StatusBadRequest, "Invalid quiet_hours_start format", "Use HH:MM format (e.g., 22:00)")
			return
		}
	}

	if req.QuietHoursEnd != nil {
		if !isValidTimeFormat(*req.QuietHoursEnd) {
			response.Error(c, http.StatusBadRequest, "Invalid quiet_hours_end format", "Use HH:MM format (e.g., 08:00)")
			return
		}
	}

	// 4. CALL SERVICE
	preferences, err := h.preferencesService.UpdateUserPreferences(c.Request.Context(), userID, req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update preferences")
		response.Error(c, http.StatusInternalServerError, "Failed to update preferences", err.Error())
		return
	}

	// 5. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Preferences updated successfully", preferences)
}

// ================================================
// HELPER FUNCTIONS
// ================================================

func isValidTimeFormat(timeStr string) bool {
	// Validate HH:MM format
	if len(timeStr) != 5 {
		return false
	}
	if timeStr[2] != ':' {
		return false
	}

	// Parse hours
	hour := 0
	if _, err := fmt.Sscanf(timeStr[:2], "%d", &hour); err != nil {
		return false
	}
	if hour < 0 || hour > 23 {
		return false
	}

	// Parse minutes
	minute := 0
	if _, err := fmt.Sscanf(timeStr[3:5], "%d", &minute); err != nil {
		return false
	}
	if minute < 0 || minute > 59 {
		return false
	}

	return true
}
