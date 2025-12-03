package main

import (
	"github.com/hibiken/asynq"

	bookJob "bookstore-backend/internal/domains/book/job"
	cartJob "bookstore-backend/internal/domains/cart/job"
	inventoryJob "bookstore-backend/internal/domains/inventory/job"
	notificationJob "bookstore-backend/internal/domains/notification/job"
	"bookstore-backend/internal/domains/user/job"
	"bookstore-backend/internal/infrastructure/email"
	emailjob "bookstore-backend/internal/infrastructure/email/job"
	"bookstore-backend/internal/shared"
	"bookstore-backend/pkg/container"
)

// HandlerRegistry holds all job handlers
type HandlerRegistry struct {
	// Email handlers
	emailVerification *emailjob.EmailVerificationHandler
	resetPassword     *emailjob.ResetPasswordEmailHandler

	// Security handlers
	securityAlert *job.SecurityAlertHandler
	failedLogin   *job.FailedLoginHandler

	// Maintenance handlers
	cleanup *job.CleanupExpiredTokenHandler

	processBookImage *bookJob.ProcessImageHandler
	deleteBookImages *bookJob.DeleteImagesHandler

	inventorySync          *inventoryJob.InventorySyncHandler
	clearCart              *cartJob.ClearCartHandler
	sendOrderConfirmation  *cartJob.SendOrderConfirmationHandler
	autoReleaseReservation *cartJob.AutoReleaseReservationHandler
	trackCheckout          *cartJob.TrackCheckoutHandler

	// WHY THIS HANDLER?
	// - Automatically removes expired/invalid promotions from carts
	// - Runs every 3 hours with smart scheduling based on user activity
	// - Prevents checkout with expired promotions
	removeExpiredPromotions  *cartJob.RemoveExpiredPromotionsHandler
	sendPendingNotifications *notificationJob.SendPendingNotificationsHandler
	cleanupOldNotifications  *notificationJob.CleanupOldNotificationsHandler // NEW
	retryFailedDeliveries    *notificationJob.RetryFailedDeliveriesHandler
}

// initializeHandlers creates all job handlers with their dependencies
func initializeHandlers(c *container.Container, cfg *Config) *HandlerRegistry {
	// Initialize services
	emailSvc := email.NewDevEmailService(cfg.SMTPHost, cfg.SMTPPort)

	// Create handlers
	return &HandlerRegistry{
		// Email handlers
		emailVerification: emailjob.NewEmailVerificationHandler(emailSvc),
		resetPassword:     emailjob.NewResetPasswordEmailHandler(emailSvc),

		// Security handlers
		securityAlert: job.NewSecurityAlertHandler(emailSvc, c.UserRepo),
		failedLogin:   job.NewFailedLoginHandler(c.Cache, c.UserRepo, c.AsynqClient),

		// Maintenance handlers
		cleanup:          job.NewCleanupExpiredTokenHandler(c.UserRepo),
		processBookImage: bookJob.NewProcessImageHandler(c.ImageBookService),
		deleteBookImages: bookJob.NewDeleteImagesHandler(c.ImageBookService),
		inventorySync: inventoryJob.NewInventorySyncHandler(
			c.InventoryRepo,
			c.Cache,
		),

		// Cart handlers
		clearCart:              cartJob.NewClearCartHandler(c.CartRepo),
		sendOrderConfirmation:  cartJob.NewSendOrderConfirmationHandler(emailSvc),
		autoReleaseReservation: cartJob.NewAutoReleaseReservationHandler(c.OrderRepo, c.InventoryService),
		trackCheckout:          cartJob.NewTrackCheckoutHandler(),

		// WHY CART REPO + NOTIFICATION SERVICE?
		// - Cart repo: Query carts and update them
		// - Notification service: Create notifications when promotions removed
		// - User info comes from JOIN query (no separate user repo needed)
		// - Promotion validation done in model methods (no promotion service needed)
		removeExpiredPromotions:  cartJob.NewRemoveExpiredPromotionsHandler(c.CartRepo, c.NotificationService),
		sendPendingNotifications: notificationJob.NewSendPendingNotificationsHandler(c.NotificationService, c.JobConfig),
		cleanupOldNotifications: notificationJob.NewCleanupOldNotificationsHandler(
			c.NotificationService,
			c.JobConfig,
		),
		retryFailedDeliveries: notificationJob.NewRetryFailedDeliveriesHandler(
			c.DeliveryService,
			c.JobConfig,
		),
	}
}

// RegisterHandlers registers all handlers with the mux
func (h *HandlerRegistry) RegisterHandlers(mux *asynq.ServeMux) {
	// Email tasks
	mux.HandleFunc(shared.TypeSendVerificationEmail, h.emailVerification.ProcessTask)
	mux.HandleFunc(shared.TypeSendResetEmail, h.resetPassword.ProcessTask)

	// Security tasks
	mux.HandleFunc(shared.TypeSendSecurityAlert, h.securityAlert.ProcessTask)
	mux.HandleFunc(shared.TypeProcessFailedLogin, h.failedLogin.ProcessTask)

	// Maintenance tasks
	mux.HandleFunc(shared.TypeCleanupExpiredToken, h.cleanup.ProcessTask)
	mux.HandleFunc(shared.TypeProcessBookImage, h.processBookImage.ProcessTask)
	mux.HandleFunc(shared.TypeDeleteBookImages, h.deleteBookImages.ProcessTask)
	// Inventory
	mux.HandleFunc(shared.TypeInventorySyncBookStock, h.inventorySync.ProcessTask)

	// Cart tasks
	mux.HandleFunc(shared.TypeClearCart, h.clearCart.ProcessTask)
	mux.HandleFunc(shared.TypeSendOrderConfirmation, h.sendOrderConfirmation.ProcessTask)
	mux.HandleFunc(shared.TypeAutoReleaseReservation, h.autoReleaseReservation.ProcessTask)
	mux.HandleFunc(shared.TypeTrackCheckout, h.trackCheckout.ProcessTask)

	// WHY REGISTER?
	// - Maps task type to handler function
	// - When scheduler enqueues task, worker knows which handler to call
	// - Task type: "cart:remove_expired_promotions"
	mux.HandleFunc(shared.TypeRemoveExpiredPromotions, h.removeExpiredPromotions.ProcessTask)
	mux.HandleFunc(shared.TypeSendPendingNotifications, h.sendPendingNotifications.ProcessTask)
	mux.HandleFunc(shared.TypeCleanupOldNotifications, h.cleanupOldNotifications.ProcessTask)
	mux.HandleFunc(shared.TypeRetryFailedDeliveries, h.retryFailedDeliveries.ProcessTask)

}
