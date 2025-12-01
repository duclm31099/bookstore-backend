package handler

import "github.com/gin-gonic/gin"

// ================================================
// HANDLER INTERFACES
// ================================================

type NotificationHandler interface {
	// User endpoints
	ListNotifications(c *gin.Context)
	GetNotification(c *gin.Context)
	MarkAsRead(c *gin.Context)
	MarkAllAsRead(c *gin.Context)
	DeleteNotification(c *gin.Context)
	GetUnreadCount(c *gin.Context)
}

type PreferencesHandler interface {
	// User preference endpoints
	GetPreferences(c *gin.Context)
	UpdatePreferences(c *gin.Context)
}

type TemplateHandler interface {
	// Admin template endpoints
	CreateTemplate(c *gin.Context)
	GetTemplate(c *gin.Context)
	ListTemplates(c *gin.Context)
	UpdateTemplate(c *gin.Context)
	DeleteTemplate(c *gin.Context)
}

type CampaignHandler interface {
	CreateCampaign(c *gin.Context)
	GetCampaign(c *gin.Context)
	ListCampaigns(c *gin.Context)
	StartCampaign(c *gin.Context)
	CancelCampaign(c *gin.Context)
}
