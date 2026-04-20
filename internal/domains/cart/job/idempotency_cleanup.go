package job

import (
	"context"

	"bookstore-backend/internal/domains/cart/repository"
	"bookstore-backend/pkg/logger"

	"github.com/hibiken/asynq"
)

type IdempotencyCleanupJob struct {
	repo repository.IdempotencyRepository
}

func NewIdempotencyCleanupJob(repo repository.IdempotencyRepository) *IdempotencyCleanupJob {
	return &IdempotencyCleanupJob{repo: repo}
}

// Run every hour
func (j *IdempotencyCleanupJob) Run(ctx context.Context, t *asynq.Task) error {
	logger.Info("Starting idempotency cleanup job", nil)

	deleted, err := j.repo.CleanupExpired(ctx)
	if err != nil {
		logger.Error("Idempotency cleanup failed", err)
		return err
	}

	logger.Info("Idempotency cleanup completed", map[string]interface{}{
		"deleted_records": deleted,
	})

	return nil
}
