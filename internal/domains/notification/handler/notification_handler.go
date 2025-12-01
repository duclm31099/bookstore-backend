package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"bookstore-backend/internal/domains/notification/model"
	"bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared/response"
)

// ================================================
// NOTIFICATION HANDLER
// ================================================

type notificationHandler struct {
	notificationService service.NotificationService
}

func NewNotificationHandler(notificationService service.NotificationService) NotificationHandler {
	return &notificationHandler{
		notificationService: notificationService,
	}
}

// ================================================
// LIST NOTIFICATIONS
// GET /api/v1/notifications
// ================================================

func (h *notificationHandler) ListNotifications(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE QUERY PARAMETERS
	req := model.ListNotificationsRequest{
		UserID: userID,
	}

	// Type filter
	if typeParam := c.Query("type"); typeParam != "" {
		req.Type = &typeParam
	}

	// IsRead filter
	if isReadParam := c.Query("is_read"); isReadParam != "" {
		isRead := isReadParam == "true"
		req.IsRead = &isRead
	}

	// Channel filter
	if channelParam := c.Query("channel"); channelParam != "" {
		req.Channel = &channelParam
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	req.Page = page
	req.PageSize = pageSize

	// Sorting
	req.SortBy = c.DefaultQuery("sort_by", "created_at")
	req.SortOrder = c.DefaultQuery("sort_order", "desc")

	// 3. CALL SERVICE
	result, err := h.notificationService.ListNotifications(c.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list notifications")
		response.Error(c, http.StatusInternalServerError, "Failed to list notifications", err.Error())
		return
	}

	// 4. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Notifications retrieved successfully", result)
}

// ================================================
// GET NOTIFICATION
// GET /api/v1/notifications/:id
// ================================================

func (h *notificationHandler) GetNotification(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE NOTIFICATION ID
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", err.Error())
		return
	}

	// 3. CALL SERVICE
	notification, err := h.notificationService.GetNotificationByID(c.Request.Context(), userID, notificationID)
	if err != nil {
		if err == model.ErrNotificationNotFound {
			response.Error(c, http.StatusNotFound, "Notification not found", err.Error())
			return
		}
		if err == model.ErrNotificationExpired {
			response.Error(c, http.StatusGone, "Notification expired", err.Error())
			return
		}
		log.Error().Err(err).Msg("Failed to get notification")
		response.Error(c, http.StatusInternalServerError, "Failed to get notification", err.Error())
		return
	}

	// 4. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Notification retrieved successfully", notification)
}

// ================================================
// MARK AS READ
// POST /api/v1/notifications/mark-read
// ================================================

func (h *notificationHandler) MarkAsRead(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE REQUEST BODY
	var req model.MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 3. VALIDATE REQUEST
	if len(req.NotificationIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "No notification IDs provided", "notification_ids is required")
		return
	}

	if len(req.NotificationIDs) > 100 {
		response.Error(c, http.StatusBadRequest, "Too many notification IDs", "Maximum 100 notifications can be marked at once")
		return
	}

	// 4. CALL SERVICE
	if err := h.notificationService.MarkAsRead(c.Request.Context(), userID, req); err != nil {
		log.Error().Err(err).Msg("Failed to mark notifications as read")
		response.Error(c, http.StatusInternalServerError, "Failed to mark notifications as read", err.Error())
		return
	}

	// 5. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Notifications marked as read", nil)
}

// ================================================
// MARK ALL AS READ
// POST /api/v1/notifications/mark-all-read
// ================================================

func (h *notificationHandler) MarkAllAsRead(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. CALL SERVICE
	count, err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to mark all notifications as read")
		response.Error(c, http.StatusInternalServerError, "Failed to mark all notifications as read", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "All notifications marked as read", map[string]interface{}{
		"count": count,
	})
}

// ================================================
// DELETE NOTIFICATION
// DELETE /api/v1/notifications/:id
// ================================================

func (h *notificationHandler) DeleteNotification(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. PARSE NOTIFICATION ID
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", err.Error())
		return
	}

	// 3. CALL SERVICE
	if err := h.notificationService.DeleteNotification(c.Request.Context(), userID, notificationID); err != nil {
		if err == model.ErrNotificationNotFound {
			response.Error(c, http.StatusNotFound, "Notification not found", err.Error())
			return
		}
		log.Error().Err(err).Msg("Failed to delete notification")
		response.Error(c, http.StatusInternalServerError, "Failed to delete notification", err.Error())
		return
	}

	// 4. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Notification deleted successfully", nil)
}

// ================================================
// GET UNREAD COUNT
// GET /api/v1/notifications/unread-count
// ================================================

func (h *notificationHandler) GetUnreadCount(c *gin.Context) {
	// 1. GET USER ID FROM AUTH CONTEXT
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	// 2. CALL SERVICE
	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get unread count")
		response.Error(c, http.StatusInternalServerError, "Failed to get unread count", err.Error())
		return
	}

	// 3. RETURN RESPONSE
	response.Success(c, http.StatusOK, "Unread count retrieved successfully", model.UnreadCountResponse{
		Count: count,
	})
}

// ================================================
// HELPER FUNCTIONS
// ================================================

func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	// Get user ID from JWT token stored in context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user_id not found in context")
	}

	// Try to parse as string first
	if userIDStr, ok := userIDInterface.(string); ok {
		return uuid.Parse(userIDStr)
	}

	// Try to parse as uuid.UUID
	if userID, ok := userIDInterface.(uuid.UUID); ok {
		return userID, nil
	}

	return uuid.Nil, fmt.Errorf("invalid user_id type in context")
}
