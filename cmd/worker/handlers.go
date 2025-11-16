package main

import (
	"github.com/hibiken/asynq"

	bookJob "bookstore-backend/internal/domains/book/job"
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
}
