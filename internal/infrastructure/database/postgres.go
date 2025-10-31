package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig chứa tất cả các thông tin cấu hình để kết nối PostgreSQL// Struct này giúp centralize và organize các parameters thay vì truyền riêng lẻ
type DBConfig struct {
	// Thông tin kết nối cơ bản
	Host     string `mapstructure:"PG_HOST"`     // Địa chỉ server PostgreSQL (vd: localhost, 192.168.1.1)
	Port     int    `mapstructure:"PG_PORT"`     // Port PostgreSQL đang lắng nghe (mặc định: 5432)
	Username string `mapstructure:"PG_USERNAME"` // Tên user để authenticate
	Password string `mapstructure:"PG_PASSWORD"` // Mật khẩu của user
	DBName   string `mapstructure:"PG_DBNAME"`   // Tên database cần kết nối

	// Connection Pool Configuration// Pool giúp tái sử dụng connections thay vì tạo mới mỗi lần, giảm overhead
	MaxConns          int32         `mapstructure:"PG_MAX_CONNS"`           // Số lượng connections tối đa trong pool (tránh quá tải DB)
	MinConns          int32         `mapstructure:"PG_MIN_CONNS"`           // Số connections tối thiểu luôn sẵn sàng (giảm latency)
	MaxConnLifetime   time.Duration `mapstructure:"PG_MAX_CONN_LIFETIME"`   // Thời gian tối đa một connection tồn tại (tránh stale connections)
	MaxConnIdleTime   time.Duration `mapstructure:"PG_MAX_CONN_IDLE_TIME"`  // Thời gian idle tối đa trước khi đóng connection
	HealthCheckPeriod time.Duration `mapstructure:"PG_HEALTH_CHECK_PERIOD"` // Tần suất kiểm tra sức khỏe của connections

	// Retry Configuration
	MaxRetries     int           `mapstructure:"PG_MAX_RETRIES"`     // Số lần retry tối đa khi kết nối thất bại
	RetryDelay     time.Duration `mapstructure:"PG_RETRY_DELAY"`     // Delay ban đầu giữa các lần retry
	ConnectTimeout time.Duration `mapstructure:"PG_CONNECT_TIMEOUT"` // Timeout cho mỗi lần thử kết nối
}

// Connection Pool là kỹ thuật tái sử dụng database connections thay vì tạo mới mỗi request.
// Tạo connection mới tốn kém về CPU, memory và network handshake.
// Pool duy trì một số connections sẵn sàng, giảm latency và tăng throughput.
// PostgresDB là wrapper quản lý connection pool và lifecycle của database// Sử dụng struct giúp encapsulate logic và dễ dàng extend thêm methods
type PostgresDB struct {
	Pool   *pgxpool.Pool // Con trỏ tới connection pool
	Config *DBConfig     // Lưu configuration để reference
}

// buildConnectionString tạo DSN (Data Source Name) string theo định dạng PostgreSQL// Format: postgresql://username:password@host:port/database
func (db *PostgresDB) buildConnectionString() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s",
		db.Config.Username, // %s: format specifier cho string
		db.Config.Password,
		db.Config.Host,
		db.Config.Port, // %d: format specifier cho integer
		db.Config.DBName,
	)
}

// configurePool tạo và cấu hình connection pool config với các best practices
func (db *PostgresDB) configurePool(ctx context.Context) (*pgxpool.Config, error) {
	// Parse connection string thành config object// ParseConfig validate và convert string thành struct có type-safe
	config, err := pgxpool.ParseConfig(db.buildConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// === POOL SIZE MANAGEMENT ===// MaxConns: Giới hạn connections để tránh exhaust database resources// Rule of thumb: MaxConns = ((core_count * 2) + effective_spindle_count)
	config.MaxConns = db.Config.MaxConns

	// MinConns: Pre-warm pool để sẵn sàng handle traffic spikes// Trade-off: Tốn resource nhưng giảm latency cho requests đầu tiên
	config.MinConns = db.Config.MinConns

	// === CONNECTION LIFECYCLE ===// MaxConnLifetime: Refresh connections định kỳ để tránh:// - Stale connections (bị DB server đóng nhưng client không biết)// - Connection leaks trong network middleboxes
	config.MaxConnLifetime = db.Config.MaxConnLifetime

	// MaxConnIdleTime: Đóng idle connections để free resources// Connections không dùng lâu sẽ bị reclaimed
	config.MaxConnIdleTime = db.Config.MaxConnIdleTime

	// === HEALTH CHECKS ===// HealthCheckPeriod: Tần suất ping connections để detect broken connections// pgx tự động remove bad connections khỏi pool
	config.HealthCheckPeriod = db.Config.HealthCheckPeriod

	// === TIMEOUTS ===// ConnectTimeout áp dụng cho mỗi lần establish connection// Tránh hang indefinitely nếu DB không response
	config.ConnConfig.ConnectTimeout = db.Config.ConnectTimeout

	return config, nil
}

// connectWithRetry thực hiện retry logic với exponential backoff// Exponential backoff: Tăng delay theo cấp số nhân giữa các lần retry// Pattern này tránh overwhelm DB khi nó đang recover
func (db *PostgresDB) connectWithRetry(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var lastErr error

	// Vòng lặp retry với counter
	for attempt := 1; attempt <= db.Config.MaxRetries; attempt++ {
		log.Printf("[DATABASE] Connection attempt %d/%d", attempt, db.Config.MaxRetries)

		// Tạo context với timeout cho mỗi attempt riêng biệt// WithTimeout return context mới + cancel function
		connectCtx, cancel := context.WithTimeout(ctx, db.Config.ConnectTimeout)

		// Thử establish connection pool
		pool, lastErr = pgxpool.NewWithConfig(connectCtx, config)

		// Cancel context ngay sau khi NewWithConfig return để free resources// Dù success hay fail, context không cần nữa
		cancel()

		// Kiểm tra kết quả
		if lastErr == nil {
			// Success - verify bằng ping
			if err := pool.Ping(ctx); err != nil {
				// Ping failed - connection không stable
				pool.Close() // Cleanup pool đã tạo
				lastErr = err
				log.Printf("[DATABASE] Ping failed: %v", err)
			} else {
				// Ping success - connection OK
				log.Printf("[DATABASE] Successfully connected on attempt %d", attempt)
				return pool, nil
			}
		}

		// Connection failed - log error
		log.Printf("[DATABASE] Attempt %d failed: %v", attempt, lastErr)

		// Nếu chưa đến retry cuối, sleep trước khi retry
		if attempt < db.Config.MaxRetries {
			// === EXPONENTIAL BACKOFF CALCULATION ===// Formula: delay = base_delay * (2 ^ (attempt - 1))// Attempt 1: 1s * 2^0 = 1s// Attempt 2: 1s * 2^1 = 2s// Attempt 3: 1s * 2^2 = 4s// Attempt 4: 1s * 2^3 = 8s
			delay := db.Config.RetryDelay * time.Duration(1<<uint(attempt-1))

			log.Printf("[DATABASE] Retrying in %v...", delay)

			// Sleep với context-aware timer// Nếu ctx bị cancel, sleep sẽ return ngay
			select {
			case <-time.After(delay):
				// Delay completed - tiếp tục retry
			case <-ctx.Done():
				// Context cancelled - abort retry
				return nil, fmt.Errorf("connection cancelled: %w", ctx.Err())
			}
		}
	}

	// Exhausted all retries
	return nil, fmt.Errorf("failed to connect after %d attempts: %w",
		db.Config.MaxRetries, lastErr)
}

// Connect là entry point chính để establish database connection// Function này orchestrate toàn bộ flow: configure -> retry -> verify
func (db *PostgresDB) Connect(ctx context.Context) error {
	log.Println("[DATABASE] Initializing PostgreSQL connection...")

	// Bước 1: Configure connection pool
	config, err := db.configurePool(ctx)
	if err != nil {
		return fmt.Errorf("pool configuration failed: %w", err)
	}

	// Bước 2: Connect với retry logic
	pool, err := db.connectWithRetry(ctx, config)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	// Bước 3: Assign pool vào struct
	db.Pool = pool

	log.Println("[DATABASE] PostgreSQL connection established successfully")
	return nil
}

// HealthCheck verify database connectivity và availability// Function này nên được call định kỳ bởi health check endpoint
func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	// Check 1: Pool có tồn tại không
	if db.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	// Check 2: Tạo timeout context cho health check// Health check không nên chờ quá lâu - 5s là reasonable
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel() // Always cancel context để free resources

	// Check 3: Ping database// Ping gửi lightweight query (như SELECT 1) để verify connection
	if err := db.Pool.Ping(healthCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check 4: Verify pool statistics (optional nhưng recommended)
	stats := db.Pool.Stat()

	// TotalConns() return tổng số connections hiện tại
	if stats.TotalConns() == 0 {
		return fmt.Errorf("no active database connections")
	}

	// Log pool statistics để monitoring
	log.Printf("[DATABASE] Health check passed - Total connections: %d, Idle: %d, Acquired: %d",
		stats.TotalConns(),
		stats.IdleConns(),     // Connections đang rảnh trong pool
		stats.AcquiredConns(), // Connections đang được sử dụng
	)

	return nil
}

// NewPostgresDB tạo instance mới của PostgresDB// Constructor pattern giúp initialize object với validation
func NewPostgresDB(config *DBConfig) *PostgresDB {
	return &PostgresDB{
		Config: config,
		Pool:   nil, // Pool sẽ được set khi Connect() được gọi
	}
}
