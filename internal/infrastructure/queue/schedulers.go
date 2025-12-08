package queue

import (
	"bookstore-backend/internal/config"
	cartModel "bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/internal/domains/user/job"
	"bookstore-backend/internal/shared"
	"bookstore-backend/pkg/logger"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

type Scheduler struct {
	scheduler *asynq.Scheduler
	jobConfig config.JobConfig
}

func NewScheduler(redisAddress string, jobConfig config.JobConfig) *Scheduler {
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddress},
		&asynq.SchedulerOpts{
			Location: time.UTC,
			LogLevel: asynq.InfoLevel,
		},
	)

	return &Scheduler{
		scheduler: scheduler,
		jobConfig: jobConfig,
	}
}

func (s *Scheduler) RegisterCleanupJobs() error {
	// Register all scheduled jobs
	if err := s.registerCleanupExpiredTokensJob(); err != nil {
		return err
	}

	if err := s.registerRemoveExpiredPromotionsJob(); err != nil {
		return err
	}

	if err := s.registerSendPendingNotificationsJob(); err != nil {
		return err
	}

	if err := s.registerCleanupOldNotificationsJob(); err != nil {
		return err
	}

	if err := s.registerRetryFailedDeliveriesJob(); err != nil {
		return err
	}

	return nil
}

// ================================================
// JOB 1: Cleanup Expired Tokens (Daily at 2 AM)
// ================================================
func (s *Scheduler) registerCleanupExpiredTokensJob() error {
	payload, err := json.Marshal(job.CleanupExpiredTokensPayload{})
	if err != nil {
		return err
	}

	task := asynq.NewTask(shared.TypeCleanupExpiredToken, payload)

	_, err = s.scheduler.Register(
		"0 2 * * *", // Daily at 2 AM
		task,
		asynq.Queue(shared.QueueUser),
		asynq.MaxRetry(1),
		asynq.Timeout(5*time.Minute),
	)

	if err != nil {
		logger.Error("Failed to register CleanupExpiredTokens job", err)
		return err
	}

	logger.Info("✓ Registered CleanupExpiredTokens: daily at 2 AM", map[string]interface{}{})
	return nil
}

// ================================================
// JOB 2: Remove Expired Promotions (Every 3 hours)
// ================================================
// WHY EVERY 3 HOURS?
// - Balance between timeliness and resource usage
// - Active users get checked frequently (every 3h)
// - Inactive users get checked when needed (24h logic inside job)
// - Prevents users from checking out with expired promotions
func (s *Scheduler) registerRemoveExpiredPromotionsJob() error {
	payload, err := json.Marshal(cartModel.RemoveExpiredPromotionsPayload{})
	if err != nil {
		return err
	}

	task := asynq.NewTask(shared.TypeRemoveExpiredPromotions, payload)

	_, err = s.scheduler.Register(
		"0 */3 * * *", // Every 3 hours at minute 0 (00:00, 03:00, 06:00, etc.)
		task,
		asynq.Queue(shared.QueuePromotion), // Default queue (medium priority)
		asynq.MaxRetry(2),                  // Retry twice if fails
		asynq.Timeout(10*time.Minute),      // 10 min timeout (handles large datasets)
	)

	if err != nil {
		logger.Error("Failed to register RemoveExpiredPromotions job", err)
		return err
	}

	logger.Info("✓ Registered RemoveExpiredPromotions: every 3 hours", map[string]interface{}{})
	return nil
}

// ================================================
// JOB 3: Send Pending Notifications (Daily at 7 AM)
// ================================================
// WHY DAILY AT 7 AM?
// - Process pending notifications that weren't sent immediately
// - Batch processing for efficiency
// - Morning time when users are likely to check notifications
// - Balance between timeliness and resource usage
func (s *Scheduler) registerSendPendingNotificationsJob() error {
	payload, err := json.Marshal(map[string]interface{}{
		"limit": s.jobConfig.SendPendingLimit, // Process 100 notifications per run
	})
	if err != nil {
		return err
	}

	task := asynq.NewTask(shared.TypeSendPendingNotifications, payload)

	_, err = s.scheduler.Register(
		"0 7 * * *", // Daily at 7 AM
		task,
		asynq.Queue(shared.QueueNotification),
		asynq.MaxRetry(3),
		asynq.Timeout(2*time.Minute),
	)

	if err != nil {
		logger.Error("Failed to register SendPendingNotifications job", err)
		return err
	}

	logger.Info("✓ Registered SendPendingNotifications: daily at 7 AM", map[string]interface{}{})
	return nil
}

// ================================================
// JOB 4: Cleanup Old Read Notifications (Daily at 3 AM)
// ================================================
// WHY DAILY AT 3 AM?
// - Low traffic time
// - Cleanup old read notifications (30 days)
// - Keep database size manageable
// - Runs 1 hour after token cleanup to avoid resource contention
func (s *Scheduler) registerCleanupOldNotificationsJob() error {
	payload, err := json.Marshal(map[string]interface{}{
		"older_than_days": s.jobConfig.CleanupRetentionDays, // Cleanup notifications older than 30 days
	})
	if err != nil {
		return err
	}

	task := asynq.NewTask(shared.TypeCleanupOldNotifications, payload)

	_, err = s.scheduler.Register(
		"0 3 * * *", // Daily at 3 AM (staggered from other cleanup jobs)
		task,
		asynq.Queue(shared.QueueNotification), // Low priority, background cleanup
		asynq.MaxRetry(2),
		asynq.Timeout(10*time.Minute),
	)

	if err != nil {
		logger.Error("Failed to register CleanupOldNotifications job", err)
		return err
	}

	logger.Info("✓ Registered CleanupOldNotifications: daily at 3 AM", map[string]interface{}{})
	return nil
}

// ================================================
// JOB 5: Retry Failed Deliveries (Every 5 minutes)
// ================================================
// WHY EVERY 5 MINUTES?
// - Quick retry for transient failures (network, provider issues)
// - Exponential backoff handled by retry_after timestamp
// - Balance between retry frequency and resource usage
func (s *Scheduler) registerRetryFailedDeliveriesJob() error {
	payload := shared.RetryFailedPayload{
		Limit: s.jobConfig.RetryFailedLimit, // Retry max 100 failed deliveries mỗi lần
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error("marshal payload: ", err)
	}
	task := asynq.NewTask(shared.TypeRetryFailedDeliveries, payloadBytes)

	_, err = s.scheduler.Register(
		"*/360 * * * *", // Every 6 hour
		task,
		asynq.Queue(shared.QueueNotification),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)

	if err != nil {
		logger.Error("Failed to register RetryFailedDeliveries job", err)
		return err
	}

	logger.Info("✓ Registered RetryFailedDeliveries: every 5 minutes", map[string]interface{}{})
	return nil
}

func (s *Scheduler) Start() error {
	return s.scheduler.Run()
}

func (s *Scheduler) Shutdown() {
	s.scheduler.Shutdown()
}
