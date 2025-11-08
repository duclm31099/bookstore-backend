package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresTransactionManager implements TransactionManager
type postgresTransactionManager struct {
	pool *pgxpool.Pool
}

func NewPostgresTransactionManager(pool *pgxpool.Pool) TransactionManager {
	return &postgresTransactionManager{pool: pool}
}

func (m *postgresTransactionManager) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (m *postgresTransactionManager) CommitTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (m *postgresTransactionManager) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
