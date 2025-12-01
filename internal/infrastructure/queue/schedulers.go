package queue

import (
	cartModel "bookstore-backend/internal/domains/cart/model"
	"bookstore-backend/internal/domains/user/job"
	"bookstore-backend/internal/shared"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

type Scheduler struct {
	scheduler *asynq.Scheduler
}

func NewScheduler(redisAddress string) *Scheduler {
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddress},
		&asynq.SchedulerOpts{
			Location: time.UTC,
			LogLevel: asynq.InfoLevel,
		},
	)

	return &Scheduler{
		scheduler: scheduler,
	}
}

func (s *Scheduler) RegisterCleanupJobs() error {
	// ================================================
	// JOB 1: Cleanup Expired Tokens (Daily at 2 AM)
	// ================================================
	payload, err := json.Marshal(job.CleanupExpiredTokensPayload{})
	if err != nil {
		return err
	}
	task := asynq.NewTask(shared.TypeCleanupExpiredToken, payload)

	_, err = s.scheduler.Register(
		"0 2 * * *",
		task,
		asynq.Queue("low"),
		asynq.MaxRetry(1),
		asynq.Timeout(5*time.Minute),
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to register CleanupExpiredTokens job")
		return err
	}
	log.Info().Msg("Registered CleanupExpiredTokens cron job: daily at 2 AM")

	// ================================================
	// JOB 2: Remove Expired Promotions (Every 3 hours)
	// ================================================
	// WHY EVERY 3 HOURS?
	// - Balance between timeliness and resource usage
	// - Active users get checked frequently (every 3h)
	// - Inactive users get checked when needed (24h logic inside job)
	// - Prevents users from checking out with expired promotions
	promoPayload, err := json.Marshal(cartModel.RemoveExpiredPromotionsPayload{})
	if err != nil {
		return err
	}
	promoTask := asynq.NewTask(shared.TypeRemoveExpiredPromotions, promoPayload)

	_, err = s.scheduler.Register(
		"0 */3 * * *", // Every 3 hours at minute 0 (00:00, 03:00, 06:00, etc.)
		promoTask,
		asynq.Queue("default"),        // Default queue (medium priority)
		asynq.MaxRetry(2),             // Retry twice if fails
		asynq.Timeout(10*time.Minute), // 10 min timeout (handles large datasets)
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to register RemoveExpiredPromotions job")
		return err
	}
	log.Info().Msg("Registered RemoveExpiredPromotions cron job: every 3 hours")

	// NEW: Gửi notification pending
	// Ví dụ: chạy mỗi 1 phút
	// Cron format: "* * * * *" => mỗi phút
	notificationTask := asynq.NewTask(shared.TypeRemoveExpiredPromotions, promoPayload)

	_, err = s.scheduler.Register(
		"0 */24 * * *", // mỗi 24 giờ
		notificationTask,
		asynq.Queue("default"), // hoặc queue riêng cho notification
	)
	if err != nil {
		log.Printf("[Scheduler] Failed to register send_pending_notifications: %v", err)
		return err
	}

	if _, err := s.scheduler.Register(
		"0 2 * * *", // hàng ngày vào 2h
		asynq.NewTask(shared.TypeCleanupOldNotifications, nil),
		asynq.Queue("default"),
	); err != nil {
		log.Printf("[Scheduler] Failed to register cleanup_old_notifications: %v", err)
		return err
	}

	return nil
}

func (s *Scheduler) Start() error {
	return s.scheduler.Run()
}

func (s *Scheduler) Shutdown() {
	s.scheduler.Shutdown()
}
