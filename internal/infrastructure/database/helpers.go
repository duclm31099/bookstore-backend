package database

import (
	"context"
	"fmt"
	"log"
	"time"

	pgx "github.com/jackc/pgx/v5"
)

// Ping kiểm tra database connection có còn sống và responsive không// Function này thường được gọi bởi health check endpoints để verify database availability// Return error nếu database không thể reach được
func (db *PostgresDB) Ping(ctx context.Context) error {
	// === VALIDATION ===// Kiểm tra pool đã được khởi tạo chưa// Nếu pool là nil, có nghĩa Connect() chưa được gọi hoặc đã failed
	if db.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	// === TIMEOUT CONTEXT ===// Tạo derived context với timeout riêng cho ping operation// 5 giây là reasonable timeout - nếu DB không respond trong 5s thì có vấn đề// Điều này tránh ping operation hang indefinitely
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	// defer ensure cancel được gọi khi function return// Cancel function giải phóng resources liên quan đến context:// - Stop timer// - Release goroutines// - Notify child contexts// Không cancel có thể gây memory leak vì timer vẫn running
	defer cancel()

	// === PING EXECUTION ===// Pool.Ping() thực hiện lightweight query để verify connection// Internally, pgx gửi một simple query (thường là SELECT 1)// Ping không chỉ check network connection mà còn verify:// 1. Database server đang running// 2. Authentication credentials vẫn valid// 3. Database còn accept connections
	if err := db.Pool.Ping(pingCtx); err != nil {
		// Wrap error với context message để dễ debug// %w preserves error chain cho errors.Is() và errors.As()
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Ping thành công - database healthy
	return nil
}

// Close đóng tất cả connections trong pool và cleanup resources
// Function này nên được gọi khi application shutdown để graceful cleanup
// Safe to call multiple times - subsequent calls sẽ là no-op
func (db *PostgresDB) Close() error {
	// === VALIDATION ===
	// Check pool có tồn tại không
	if db.Pool == nil {
		// Pool đã closed hoặc chưa initialized
		// Return nil vì đây là desired state (closed)
		// Không nên return error vì Close() idempotent
		log.Println("[DATABASE] Pool is already closed or was never initialized")
		return nil
	}

	// === LOG SHUTDOWN INITIATION ===
	log.Println("[DATABASE] Closing database connection pool...")

	// === CLOSE POOL ===
	// Pool.Close() performs graceful shutdown:
	// Step 1: Mark pool as closed - reject new connection acquisitions
	// Step 2: Wait for all acquired connections to be released
	//         (hoặc timeout based on MaxConnLifetime)
	// Step 3: Close idle connections trong pool
	// Step 4: Terminate TCP connections tới PostgreSQL
	// Step 5: Free memory buffers và goroutines
	db.Pool.Close()

	// === SET POOL TO NIL ===
	// Đảm bảo subsequent calls biết pool đã closed
	// Prevent use-after-free bugs
	db.Pool = nil

	// === LOG SUCCESS ===
	log.Println("[DATABASE] Connection pool closed successfully")

	// Close luôn succeed với pgxpool
	// Nếu có connections đang active, Close sẽ đợi chúng complete
	return nil
}

// PoolStats chứa thống kê chi tiết về connection pool
// Struct này được dùng cho monitoring và debugging performance issues
type PoolStats struct {
	// Connection counts
	AcquireCount            int64         // Tổng số lần acquire connection (lifetime metric)
	AcquireDuration         time.Duration // Tổng thời gian spent acquiring connections
	AcquiredConns           int32         // Số connections hiện đang được used
	CanceledAcquireCount    int64         // Số lần acquire bị cancel (do timeout hoặc context cancel)
	ConstructingConns       int32         // Số connections đang được establish (connecting state)
	EmptyAcquireCount       int64         // Số lần acquire từ empty pool (had to wait)
	IdleConns               int32         // Số connections idle, sẵn sàng dùng
	MaxConns                int32         // Max connections configured
	TotalConns              int32         // Total connections (acquired + idle + constructing)
	NewConnsCount           int64         // Số connections mới đã tạo (lifetime metric)
	MaxLifetimeDestroyCount int64         // Connections closed do exceed MaxConnLifetime
	MaxIdleDestroyCount     int64         // Connections closed do exceed MaxConnIdleTime
}

// Stats trả về snapshot của connection pool statistics
// Function này useful cho monitoring, alerting và performance tuning
func (db *PostgresDB) Stats() (*PoolStats, error) {
	// === VALIDATION ===
	if db.Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	// === GET RAW STATS ===
	// Pool.Stat() return pgxpool.Stat struct với raw metrics
	// Đây là atomic snapshot - all values là consistent tại một thời điểm
	rawStats := db.Pool.Stat()

	// === TRANSFORM TO CUSTOM STRUCT ===
	// Transform pgxpool.Stat sang custom PoolStats struct
	// Custom struct cho phép thêm computed fields hoặc change representation
	stats := &PoolStats{
		// === CURRENT STATE METRICS ===
		// Metrics này show trạng thái hiện tại của pool
		AcquiredConns:     rawStats.AcquiredConns(),     // Connections đang active
		ConstructingConns: rawStats.ConstructingConns(), // Connections đang setup
		IdleConns:         rawStats.IdleConns(),         // Connections ready to use
		TotalConns:        rawStats.TotalConns(),        // Total = Acquired + Idle + Constructing
		MaxConns:          rawStats.MaxConns(),          // Configured limit

		// === LIFETIME CUMULATIVE METRICS ===
		// Metrics này accumulate over pool lifetime - monotonically increasing
		AcquireCount:         rawStats.AcquireCount(),         // Total acquisition attempts
		AcquireDuration:      rawStats.AcquireDuration(),      // Total time waiting for connections
		CanceledAcquireCount: rawStats.CanceledAcquireCount(), // Timeouts/cancellations
		EmptyAcquireCount:    rawStats.EmptyAcquireCount(),    // Had to wait for connection
		NewConnsCount:        rawStats.NewConnsCount(),        // Total connections created

		// === DESTRUCTION METRICS ===
		// Track why connections were closed
		MaxLifetimeDestroyCount: rawStats.MaxLifetimeDestroyCount(), // Aged out
		MaxIdleDestroyCount:     rawStats.MaxIdleDestroyCount(),     // Idle too long
	}

	// === LOG STATISTICS (optional, useful for debugging) ===
	log.Printf(`[DATABASE] Pool Statistics:
        Total Connections: %d (Max: %d)
        Active: %d | Idle: %d | Constructing: %d
        Total Acquires: %d | Empty Acquires: %d | Canceled: %d
        Average Acquire Duration: %v
        New Connections Created: %d
        Destroyed by MaxLifetime: %d | Destroyed by MaxIdleTime: %d`,
		stats.TotalConns, stats.MaxConns,
		stats.AcquiredConns, stats.IdleConns, stats.ConstructingConns,
		stats.AcquireCount, stats.EmptyAcquireCount, stats.CanceledAcquireCount,
		calculateAvgDuration(stats.AcquireDuration, stats.AcquireCount),
		stats.NewConnsCount,
		stats.MaxLifetimeDestroyCount, stats.MaxIdleDestroyCount,
	)

	return stats, nil
}

// calculateAvgDuration là helper để tính average acquire duration
func calculateAvgDuration(totalDuration time.Duration, count int64) time.Duration {
	if count == 0 {
		return 0
	}
	return totalDuration / time.Duration(count)
}

// TxOptions cấu hình transaction behavior
// Struct này map tới PostgreSQL transaction parameters
type TxOptions struct {
	// IsoLevel định nghĩa transaction isolation level
	// Isolation level control cách transaction thấy data changes từ concurrent transactions
	IsoLevel TxIsoLevel

	// AccessMode định nghĩa transaction có thể modify data không
	AccessMode TxAccessMode

	// DeferrableMode chỉ áp dụng cho Serializable isolation level
	// Deferred transaction có thể wait để avoid serialization failures
	DeferrableMode TxDeferrableMode
}

// TxIsoLevel là transaction isolation level
// PostgreSQL hỗ trợ 3 levels (Read Uncommitted maps to Read Committed)
type TxIsoLevel string

const (
	// ReadCommitted (default): Mỗi statement thấy snapshot tại start time của nó
	// Không thấy uncommitted data (no dirty reads)
	// Có thể thấy data changes từ committed transactions (non-repeatable reads possible)
	ReadCommitted TxIsoLevel = "read committed"

	// RepeatableRead: Transaction thấy consistent snapshot tại start time
	// Không thấy changes từ concurrent transactions (no non-repeatable reads)
	// Phantom reads không xảy ra trong PostgreSQL (stronger than SQL standard)
	RepeatableRead TxIsoLevel = "repeatable read"

	// Serializable: Strongest isolation level
	// Transaction hoàn toàn isolated - như thể chạy tuần tự
	// Có thể fail với serialization error nếu có conflicts
	Serializable TxIsoLevel = "serializable"
)

// TxAccessMode định nghĩa transaction read-only hay read-write
type TxAccessMode string

const (
	// ReadWrite (default): Transaction có thể read và write data
	ReadWrite TxAccessMode = "read write"

	// ReadOnly: Transaction chỉ read, không modify data
	// PostgreSQL có thể optimize read-only transactions
	ReadOnly TxAccessMode = "read only"
)

// TxDeferrableMode chỉ meaningful cho Serializable + ReadOnly transactions
type TxDeferrableMode string

const (
	// NotDeferrable (default): Transaction không deferrable
	NotDeferrable TxDeferrableMode = "not deferrable"

	// Deferrable: Transaction có thể delay start để acquire consistent snapshot
	// Tránh serialization failures nhưng có thể increase latency
	Deferrable TxDeferrableMode = "deferrable"
)

// BeginTx starts một database transaction với options specified
// Transaction phải được commit hoặc rollback - nếu không sẽ hold locks và connections
// Returns pgx.Tx interface để execute statements trong transaction context
func (db *PostgresDB) BeginTx(ctx context.Context, opts *TxOptions) (pgx.Tx, error) {
	// === VALIDATION ===
	if db.Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	// === CONVERT TO PGX TX OPTIONS ===
	// Transform custom TxOptions sang pgx.TxOptions
	// pgx.TxOptions là native type của pgx library
	pgxOpts := pgx.TxOptions{}

	if opts != nil {
		// Map isolation level
		switch opts.IsoLevel {
		case ReadCommitted:
			pgxOpts.IsoLevel = pgx.ReadCommitted
		case RepeatableRead:
			pgxOpts.IsoLevel = pgx.RepeatableRead
		case Serializable:
			pgxOpts.IsoLevel = pgx.Serializable
		default:
			// Default to ReadCommitted nếu không specify
			pgxOpts.IsoLevel = pgx.ReadCommitted
		}

		// Map access mode
		switch opts.AccessMode {
		case ReadOnly:
			pgxOpts.AccessMode = pgx.ReadOnly
		case ReadWrite:
			pgxOpts.AccessMode = pgx.ReadWrite
		default:
			pgxOpts.AccessMode = pgx.ReadWrite
		}

		// Map deferrable mode
		switch opts.DeferrableMode {
		case Deferrable:
			pgxOpts.DeferrableMode = pgx.Deferrable
		case NotDeferrable:
			pgxOpts.DeferrableMode = pgx.NotDeferrable
		default:
			pgxOpts.DeferrableMode = pgx.NotDeferrable
		}
	}

	// === BEGIN TRANSACTION ===
	// Pool.BeginTx() performs:
	// 1. Acquire connection từ pool (block nếu pool full)
	// 2. Execute BEGIN statement với specified options
	// 3. Return pgx.Tx interface wrapping connection
	// Context chỉ áp dụng cho BEGIN command, không auto-rollback on cancel
	tx, err := db.Pool.BeginTx(ctx, pgxOpts)
	if err != nil {
		// BEGIN failed - có thể do:
		// - Pool exhausted
		// - Network error
		// - Database refusing connections
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// === LOG TRANSACTION START (optional) ===
	log.Printf("[DATABASE] Transaction started with isolation level: %v", opts.IsoLevel)

	// Return transaction handle
	// Caller chịu trách nhiệm commit hoặc rollback
	return tx, nil
}

// ExecuteInTransaction là wrapper function thực thi logic trong transaction
// Automatically handle commit/rollback based on function return value
// Pattern này giảm boilerplate code và tránh forget rollback
func (db *PostgresDB) ExecuteInTransaction(
	ctx context.Context,
	opts *TxOptions,
	fn func(pgx.Tx) error,
) error {
	// === BEGIN TRANSACTION ===
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	// === SETUP DEFERRED ROLLBACK ===
	// defer ensure rollback được gọi nếu có error
	// Rollback on committed transaction là no-op (safe)
	// Pattern này ensure transaction luôn được cleaned up
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			// Rollback error chỉ log, không return
			// Có thể rollback fail nếu transaction đã committed
			if err != pgx.ErrTxClosed {
				log.Printf("[DATABASE] Transaction rollback error: %v", err)
			}
		}
	}()

	// === EXECUTE USER FUNCTION ===
	// fn chứa business logic cần chạy trong transaction
	if err := fn(tx); err != nil {
		// fn failed - transaction sẽ được rollback bởi defer
		return fmt.Errorf("transaction function failed: %w", err)
	}

	// === COMMIT TRANSACTION ===
	// fn succeeded - commit changes
	if err := tx.Commit(ctx); err != nil {
		// Commit failed - có thể do:
		// - Serialization failure
		// - Constraint violation
		// - Disk full
		return fmt.Errorf("transaction commit failed: %w", err)
	}

	log.Println("[DATABASE] Transaction committed successfully")
	return nil
}

// MonitorPoolHealth continuously monitors pool statistics và alert on issues
// Function này nên chạy trong separate goroutine
func (db *PostgresDB) MonitorPoolHealth(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats, err := db.Stats()
			if err != nil {
				log.Printf("[MONITOR] Failed to get stats: %v", err)
				continue
			}

			// === CHECK POOL EXHAUSTION ===
			utilizationPct := float64(stats.AcquiredConns) / float64(stats.MaxConns) * 100
			if utilizationPct > 80 {
				log.Printf("[MONITOR] HIGH POOL UTILIZATION: %.1f%% (%d/%d)",
					utilizationPct, stats.AcquiredConns, stats.MaxConns)
			}

			// === CHECK ACQUIRE WAIT TIME ===
			avgAcquireDuration := calculateAvgDuration(
				stats.AcquireDuration,
				stats.AcquireCount,
			)
			if avgAcquireDuration > 100*time.Millisecond {
				log.Printf("[MONITOR] HIGH ACQUIRE LATENCY: %v", avgAcquireDuration)
			}

			// === CHECK CANCELED ACQUIRES ===
			if stats.CanceledAcquireCount > 0 {
				cancelRate := float64(stats.CanceledAcquireCount) /
					float64(stats.AcquireCount) * 100
				if cancelRate > 5 {
					log.Printf("[MONITOR] HIGH CANCEL RATE: %.1f%%", cancelRate)
				}
			}

		case <-ctx.Done():
			log.Println("[MONITOR] Stopping pool health monitoring")
			return
		}
	}
}
