package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTransaction function:
//     Begin transaction từ pool
//     Defer rollback - Sẽ tự động rollback nếu:
//         Function fn return error
//         Có panic xảy ra
//     Execute function fn với transaction context
//     Commit nếu không có error

// TxFunc là function type được execute trong transaction
type TxFunc func(pgx.Tx) error

// WithTransaction wraps một function trong transaction
// Auto rollback nếu có error, auto commit nếu success
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback (sẽ bị ignore nếu đã commit)
	defer func() {
		if p := recover(); p != nil {
			// Có panic → rollback
			tx.Rollback(ctx)
			panic(p) // Re-throw panic
		} else if err != nil {
			// Có error → rollback
			tx.Rollback(ctx)
		}
	}()

	// Execute function trong transaction context
	err = fn(tx)
	if err != nil {
		return err // Defer sẽ rollback
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionResult wraps function có return value trong transaction
func WithTransactionResult[T any](ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) (T, error)) (T, error) {
	var result T
	var fnErr error

	err := WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		result, fnErr = fn(tx)
		return fnErr
	})

	if err != nil {
		var zero T
		return zero, err
	}

	return result, nil
}
