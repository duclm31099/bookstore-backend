package job

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"bookstore-backend/internal/domains/cart/model"
	cartRepo "bookstore-backend/internal/domains/cart/repository"
	notificationModel "bookstore-backend/internal/domains/notification/model"
	notificationService "bookstore-backend/internal/domains/notification/service"
	"bookstore-backend/internal/shared/utils"
	"bookstore-backend/pkg/logger"
)

// ================================================
// REMOVE EXPIRED PROMOTIONS JOB HANDLER
// ================================================

// WHY THIS JOB?
// - Automatically removes expired/invalid promotions from user carts
// - Prevents users from checking out with expired promotions
// - Maintains data integrity and user experience
// - Notifies users via notification system

// RemoveExpiredPromotionsHandler handles the scheduled job
type RemoveExpiredPromotionsHandler struct {
	cartRepo            cartRepo.RepositoryInterface
	notificationService notificationService.NotificationService // ✅ UPDATED: Use correct interface
}

// NewRemoveExpiredPromotionsHandler creates a new handler instance
func NewRemoveExpiredPromotionsHandler(
	cartRepo cartRepo.RepositoryInterface,
	notificationService notificationService.NotificationService, // ✅ UPDATED
) *RemoveExpiredPromotionsHandler {
	return &RemoveExpiredPromotionsHandler{
		cartRepo:            cartRepo,
		notificationService: notificationService,
	}
}

// ProcessTask is the main entry point for the scheduled job
// EXECUTION FLOW:
// 1. Parse payload (empty for scheduled job)
// 2. Process carts in batches of 100
// 3. For each cart: check if should process, remove if invalid
// 4. Log results and statistics
func (h *RemoveExpiredPromotionsHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	// Step 1: Parse payload
	var payload model.RemoveExpiredPromotionsPayload
	if err := utils.UnmarshalTask(t, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger.Info("Starting remove expired promotions job", map[string]interface{}{
		"started_at": time.Now(),
	})

	// Statistics tracking
	stats := &JobStatistics{
		StartTime: time.Now(),
	}

	// Step 2: Process carts in batches
	const batchSize = 100
	offset := 0

	for {
		// Fetch next batch of carts with promotions
		carts, err := h.cartRepo.GetCartsWithPromotions(ctx, batchSize, offset)
		if err != nil {
			logger.Error("Failed to fetch carts batch", err)
			return fmt.Errorf("fetch carts batch (offset=%d): %w", offset, err)
		}

		// No more carts to process
		if len(carts) == 0 {
			break
		}

		logger.Info("Processing batch", map[string]interface{}{
			"offset":       offset,
			"batch_size":   len(carts),
			"total_so_far": stats.TotalProcessed,
		})

		// Process each cart in the batch
		for _, cart := range carts {
			if err := h.processCart(ctx, cart, stats); err != nil {
				logger.Error("Failed to process cart", err)
				stats.Errors++
			}
		}

		// Move to next batch
		offset += batchSize
		stats.TotalProcessed += len(carts)

		// Safety check: prevent infinite loop
		if offset >= 10000 {
			logger.Info("Reached safety limit, stopping", map[string]interface{}{
				"offset": offset,
			})
			break
		}
	}

	// Step 3: Log final statistics
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	logger.Info("Completed remove expired promotions job", map[string]interface{}{
		"total_processed": stats.TotalProcessed,
		"removed":         stats.Removed,
		"skipped":         stats.Skipped,
		"errors":          stats.Errors,
		"duration":        stats.Duration.String(),
	})

	return nil
}

// processCart handles a single cart
func (h *RemoveExpiredPromotionsHandler) processCart(
	ctx context.Context,
	cart *model.CartWithPromoInfo,
	stats *JobStatistics,
) error {
	// Step 1: Smart scheduling check
	if !cart.ShouldProcessNow() {
		stats.Skipped++
		logger.Info("Skipping cart (not time yet)", map[string]interface{}{
			"cart_id":        cart.CartID,
			"user_id":        cart.UserID,
			"is_active_user": cart.IsUserActive(),
			"last_checked":   cart.GetLastCheckedAt(),
		})
		return nil
	}

	// Step 2: Check if promotion should be removed
	shouldRemove, reason := cart.ShouldRemovePromotion()

	if shouldRemove {
		// Step 3a: Remove invalid promotion
		logger.Info("Removing invalid promotion", map[string]interface{}{
			"cart_id":    cart.CartID,
			"user_id":    cart.UserID,
			"promo_code": cart.PromoCode,
			"reason":     reason,
		})

		// Build metadata for audit log
		metadata := h.buildRemovalMetadata(cart, reason)

		// Remove promotion and create audit log (atomic transaction)
		err := h.cartRepo.RemovePromotionWithLog(
			ctx,
			cart.CartID,
			cart.UserID,
			cart.PromoCode,
			cart.Discount,
			reason,
			metadata,
		)
		if err != nil {
			return fmt.Errorf("remove promotion: %w", err)
		}

		// ✅ UPDATED: Create notification using SendNotification method
		h.sendPromotionRemovedNotification(ctx, cart, reason, metadata)

		stats.Removed++
	} else {
		// Step 3b: Promotion still valid, update last_checked_at
		logger.Info("Promotion still valid, updating last_checked_at", map[string]interface{}{
			"cart_id":    cart.CartID,
			"user_id":    cart.UserID,
			"promo_code": cart.PromoCode,
		})

		// Update last_checked_at timestamp in promo_metadata
		metadata := map[string]interface{}{
			"last_checked_at": time.Now().Format(time.RFC3339),
		}

		err := h.cartRepo.UpdatePromoMetadata(ctx, cart.CartID, metadata)
		if err != nil {
			logger.Error("Failed to update last_checked_at", err)
		}
	}

	return nil
}

// buildRemovalMetadata creates metadata for audit log
func (h *RemoveExpiredPromotionsHandler) buildRemovalMetadata(
	cart *model.CartWithPromoInfo,
	reason string,
) map[string]interface{} {
	metadata := map[string]interface{}{
		"removal_reason": reason,
		"removed_at":     time.Now().Format(time.RFC3339),
		"promo_code":     cart.PromoCode,
	}

	// Add promotion details if available
	if cart.PromotionID != nil {
		metadata["promotion_id"] = cart.PromotionID.String()
	}
	if cart.ExpiresAt != nil {
		metadata["expires_at"] = cart.ExpiresAt.Format(time.RFC3339)
	}
	if cart.IsActive != nil {
		metadata["is_active"] = *cart.IsActive
	}
	if cart.MaxUses != nil {
		metadata["max_uses"] = *cart.MaxUses
	}
	if cart.CurrentUses != nil {
		metadata["current_uses"] = *cart.CurrentUses
	}

	// Merge with existing promo_metadata if available
	if cart.PromoMetadata != nil {
		for k, v := range cart.PromoMetadata {
			if _, exists := metadata[k]; !exists {
				metadata[k] = v
			}
		}
	}

	return metadata
}

// ✅ UPDATED: sendPromotionRemovedNotification sends notification via template
// WHY USE TEMPLATE SYSTEM?
// - Centralized content management (admin can edit templates)
// - Multi-language support ready
// - Consistent notification styling
// - Variable substitution (promo_code, reason, etc.)
func (h *RemoveExpiredPromotionsHandler) sendPromotionRemovedNotification(
	ctx context.Context,
	cart *model.CartWithPromoInfo,
	reason string,
	metadata map[string]interface{},
) {
	// Build reason text for template
	var reasonText string
	switch reason {
	case "expired":
		reasonText = "đã hết hạn"
	case "disabled":
		reasonText = "đã bị vô hiệu hóa"
	case "max_uses_reached":
		reasonText = "đã đạt giới hạn sử dụng"
	default:
		reasonText = "không còn khả dụng"
	}

	// Prepare template data
	templateData := map[string]interface{}{
		"promo_code": cart.PromoCode,
		"reason":     reasonText,
		"removed_at": time.Now().Format("02/01/2006 15:04"),
		"cart_id":    cart.CartID.String(),
		"discount":   cart.Discount,
	}

	// Merge with existing metadata
	for k, v := range metadata {
		if _, exists := templateData[k]; !exists {
			templateData[k] = v
		}
	}
	priority := notificationModel.PriorityMedium
	// ✅ Create notification request using SendNotification
	req := notificationModel.SendNotificationRequest{
		UserID:       cart.UserID,
		TemplateCode: "promotion_removed", // Template code (must exist in DB)
		Channels: []string{
			notificationModel.ChannelInApp, // Always send in-app
			// notificationModel.ChannelEmail, // Optional: email notification
		},
		Data:          templateData,
		ReferenceType: stringPtr("cart"),
		ReferenceID:   &cart.CartID,
		Priority:      &priority,
	}

	// Send notification (non-blocking, log errors but don't fail job)
	_, err := h.notificationService.SendNotification(ctx, req)
	if err != nil {
		// WHY NOT FAIL JOB?
		// - Notification is secondary to promotion removal
		// - Job should succeed even if notification fails
		// - User can still see removal in audit log
		logger.Error("Failed to send promotion removed notification", err)
		return
	}

	logger.Info("Sent promotion removed notification", map[string]interface{}{
		"user_id":    cart.UserID,
		"cart_id":    cart.CartID,
		"promo_code": cart.PromoCode,
		"reason":     reason,
	})
}

// ================================================
// STATISTICS TRACKING
// ================================================

type JobStatistics struct {
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	TotalProcessed int
	Removed        int
	Skipped        int
	Errors         int
}

// ================================================
// HELPER FUNCTIONS
// ================================================
