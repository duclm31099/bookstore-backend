package queue

import (
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
	return nil
}
func (s *Scheduler) Start() error {
	return s.scheduler.Run()
}

func (s *Scheduler) Shutdown() {
	s.scheduler.Shutdown()
}
