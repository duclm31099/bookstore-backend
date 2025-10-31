<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# **SPRINT 6: ASYNQ SETUP + BASIC BACKGROUND JOBS (Tuần 11-12)**


***

## **TỔNG QUAN SPRINT 6 (REVISED)**

### **Mục tiêu**

Xây dựng hạ tầng background job processing với Asynq và triển khai các background jobs cơ bản cho hệ thống (cart cleanup, order sync, daily stats).

### **Deliverables**

- ✅ Asynq client \& server infrastructure
- ✅ Queue configuration (critical, high, default, low)
- ✅ Worker pool management
- ✅ Cart expiration cleanup job
- ✅ Order status sync job
- ✅ Daily statistics job
- ✅ Inventory sync job
- ✅ Low stock alert job
- ✅ Job monitoring \& health checks
- ✅ Graceful shutdown mechanism


### **Tech Stack**

- Asynq v0.24+
- Redis 7+ (queue backend)
- Cron syntax for scheduled jobs


### **Tham khảo URD**

- Phase 2, Task P2-T008: Asynq setup + basic jobs
- Phase 1, Task P1-T034: Send order confirmation (sẽ làm Sprint 8)
- Background Jobs table trong URD Section 8

***

## **TUẦN 11 - NGÀY 1-2: ASYNQ INFRASTRUCTURE SETUP**

### **☐ Task 6.1: Install Asynq Dependencies**

```bash
# Install Asynq
go get -u github.com/hibiken/asynq

# Install Asynq monitoring tool (optional)
go install github.com/hibiken/asynq/tools/asynq@latest
```

File: `go.mod` (verify)

```go
require (
    github.com/hibiken/asynq v0.24.1
    github.com/redis/go-redis/v9 v9.3.0
)
```


***

### **☐ Task 6.2: Create Queue Configuration**

File: `pkg/queue/config.go`

```go
package queue

import (
    "time"
    "github.com/hibiken/asynq"
    "github.com/redis/go-redis/v9"
)

const (
    // Queue names
    QueueCritical = "critical" // Payment processing, inventory reservation
    QueueHigh     = "high"     // Emails, notifications
    QueueDefault  = "default"  // General background tasks
    QueueLow      = "low"      // Analytics, cleanup
)

// QueueConfig defines queue priorities and worker counts
type QueueConfig struct {
    Queues map[string]int // map[queue_name]priority
    Concurrency int         // Total concurrent workers
}

// DefaultQueueConfig returns production-ready configuration
func DefaultQueueConfig() *QueueConfig {
    return &QueueConfig{
        Queues: map[string]int{
            QueueCritical: 10, // Highest priority
            QueueHigh:     5,
            QueueDefault:  3,
            QueueLow:      1,  // Lowest priority
        },
        Concurrency: 20, // Total workers across all queues
    }
}

// RedisConnOpt creates Redis connection options for Asynq
func RedisConnOpt(redisAddr, password string, db int) asynq.RedisConnOpt {
    return asynq.RedisClientOpt{
        Addr:     redisAddr,
        Password: password,
        DB:       db,
    }
}
```


***

### **☐ Task 6.3: Create Asynq Client**

File: `pkg/queue/client.go`

```go
package queue

import (
    "encoding/json"
    "time"
    "github.com/hibiken/asynq"
)

// Client wraps Asynq client for enqueueing tasks
type Client struct {
    asynqClient *asynq.Client
}

// NewClient creates a new queue client
func NewClient(redisOpt asynq.RedisConnOpt) *Client {
    return &Client{
        asynqClient: asynq.NewClient(redisOpt),
    }
}

// Enqueue adds a task to the queue
func (c *Client) Enqueue(taskType string, payload interface{}, opts ...asynq.Option) error {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    task := asynq.NewTask(taskType, payloadBytes, opts...)
    _, err = c.asynqClient.Enqueue(task)
    return err
}

// EnqueueIn schedules a task to run after a duration
func (c *Client) EnqueueIn(taskType string, payload interface{}, delay time.Duration, opts ...asynq.Option) error {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    task := asynq.NewTask(taskType, payloadBytes, opts...)
    _, err = c.asynqClient.Enqueue(task, asynq.ProcessIn(delay))
    return err
}

// EnqueueAt schedules a task to run at a specific time
func (c *Client) EnqueueAt(taskType string, payload interface{}, t time.Time, opts ...asynq.Option) error {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    
    task := asynq.NewTask(taskType, payloadBytes, opts...)
    _, err = c.asynqClient.Enqueue(task, asynq.ProcessAt(t))
    return err
}

// Close closes the client connection
func (c *Client) Close() error {
    return c.asynqClient.Close()
}
```


***

### **☐ Task 6.4: Create Asynq Server**

File: `pkg/queue/server.go`

```go
package queue

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"
)

// Server wraps Asynq server for processing tasks
type Server struct {
    asynqServer *asynq.Server
    mux         *asynq.ServeMux
}

// NewServer creates a new queue server
func NewServer(redisOpt asynq.RedisConnOpt, cfg *QueueConfig) *Server {
    serverConfig := asynq.Config{
        Concurrency: cfg.Concurrency,
        Queues:      cfg.Queues,
        
        // Retry configuration
        RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
            // Exponential backoff: 1s, 2s, 4s, 8s, 16s
            return time.Duration(1<<uint(n)) * time.Second
        },
        
        // Error handler
        ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
            log.Error().
                Str("task_type", task.Type()).
                Str("payload", string(task.Payload())).
                Err(err).
                Msg("Task processing failed")
        }),
        
        // Logger
        Logger: NewAsynqLogger(),
        
        // Health check server
        HealthCheckFunc: func(err error) {
            if err != nil {
                log.Error().Err(err).Msg("Health check failed")
            }
        },
        HealthCheckInterval: 15 * time.Second,
    }
    
    return &Server{
        asynqServer: asynq.NewServer(redisOpt, serverConfig),
        mux:         asynq.NewServeMux(),
    }
}

// RegisterHandler registers a task handler
func (s *Server) RegisterHandler(taskType string, handler asynq.Handler) {
    s.mux.Handle(taskType, handler)
}

// RegisterHandlerFunc registers a task handler function
func (s *Server) RegisterHandlerFunc(taskType string, handlerFunc func(context.Context, *asynq.Task) error) {
    s.mux.HandleFunc(taskType, handlerFunc)
}

// Start starts the server with graceful shutdown
func (s *Server) Start() error {
    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    // Start server in goroutine
    errChan := make(chan error, 1)
    go func() {
        log.Info().Msg("Starting Asynq worker server...")
        if err := s.asynqServer.Run(s.mux); err != nil {
            errChan <- err
        }
    }()
    
    // Wait for shutdown signal or error
    select {
    case sig := <-sigChan:
        log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
        s.Shutdown()
        return nil
    case err := <-errChan:
        return fmt.Errorf("server error: %w", err)
    }
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
    log.Info().Msg("Shutting down Asynq worker server...")
    s.asynqServer.Shutdown()
    log.Info().Msg("Asynq worker server stopped")
}

// NewAsynqLogger creates a logger adapter for Asynq
func NewAsynqLogger() asynq.Logger {
    return &asynqLogger{}
}

type asynqLogger struct{}

func (l *asynqLogger) Debug(args ...interface{}) {
    log.Debug().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...interface{}) {
    log.Info().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...interface{}) {
    log.Warn().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...interface{}) {
    log.Error().Msg(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...interface{}) {
    log.Fatal().Msg(fmt.Sprint(args...))
}
```


***

### **☐ Task 6.5: Create Worker Entry Point**

File: `cmd/worker/main.go`

```go
package main

import (
    "log"
    "bookstore-backend/pkg/config"
    "bookstore-backend/pkg/logger"
    "bookstore-backend/pkg/db"
    "bookstore-backend/pkg/queue"
    "bookstore-backend/internal/worker"
)

func main() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Init logger
    logger.Init(cfg.App.Env)
    
    // Connect to PostgreSQL
    pgPool, err := db.NewPostgresPool(
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.User,
        cfg.Database.Password,
        cfg.Database.DBName,
        cfg.Database.MaxConns,
    )
    if err != nil {
        log.Fatal("Failed to connect to PostgreSQL:", err)
    }
    defer pgPool.Close()
    
    // Connect to Redis
    redisClient, err := db.NewRedisClient(
        cfg.Redis.Host,
        cfg.Redis.Password,
        cfg.Redis.DB,
    )
    if err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }
    defer redisClient.Close()
    
    // Create Asynq server
    redisOpt := queue.RedisConnOpt(cfg.Redis.Host, cfg.Redis.Password, cfg.Redis.DB)
    queueCfg := queue.DefaultQueueConfig()
    server := queue.NewServer(redisOpt, queueCfg)
    
    // Initialize repositories
    cartRepo := cartRepository.NewPostgresRepository(pgPool)
    orderRepo := orderRepository.NewPostgresRepository(pgPool)
    
    // Register task handlers
    worker.RegisterHandlers(server, pgPool, redisClient, cartRepo, orderRepo)
    
    // Start server (blocks until shutdown)
    if err := server.Start(); err != nil {
        log.Fatal("Worker server error:", err)
    }
}
```


***

### **☐ Task 6.6: Update Main API Server to Create Queue Client**

File: `cmd/api/main.go` (update)

```go
// ... existing code ...

func main() {
    // ... existing setup ...
    
    // Create Asynq client (for enqueueing tasks from API handlers)
    redisOpt := queue.RedisConnOpt(cfg.Redis.Host, cfg.Redis.Password, cfg.Redis.DB)
    queueClient := queue.NewClient(redisOpt)
    defer queueClient.Close()
    
    // Pass queueClient to services that need to enqueue jobs
    // e.g., orderService := orderService.NewOrderService(orderRepo, addressRepo, cartService, queueClient)
    
    // ... rest of the code ...
}
```


***

## **TUẦN 11 - NGÀY 3-4: CART EXPIRATION CLEANUP JOB**

### **☐ Task 6.7: Create Cart Cleanup Task Definition**

File: `internal/worker/tasks/cart_cleanup.go`

```go
package tasks

const (
    TypeCartCleanup = "cart:cleanup"
)

// CartCleanupPayload - empty payload for scheduled cleanup
type CartCleanupPayload struct{}
```


***

### **☐ Task 6.8: Create Cart Cleanup Handler**

File: `internal/worker/handlers/cart_cleanup_handler.go`

```go
package handlers

import (
    "context"
    "time"
    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"
    "bookstore-backend/internal/domains/cart/repository"
    "bookstore-backend/internal/worker/tasks"
)

type CartCleanupHandler struct {
    cartRepo repository.CartRepository
}

func NewCartCleanupHandler(cartRepo repository.CartRepository) *CartCleanupHandler {
    return &CartCleanupHandler{cartRepo: cartRepo}
}

func (h *CartCleanupHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    startTime := time.Now()
    
    log.Info().
        Str("task_type", task.Type()).
        Msg("Starting cart cleanup job")
    
    // Delete expired carts
    err := h.cartRepo.DeleteExpired(ctx)
    if err != nil {
        log.Error().
            Err(err).
            Msg("Failed to delete expired carts")
        return err
    }
    
    duration := time.Since(startTime)
    log.Info().
        Dur("duration_ms", duration).
        Msg("Cart cleanup job completed")
    
    return nil
}
```

**Update Cart Repository:**

File: `internal/domains/cart/repository/postgres.go` (add method)

```go
func (r *postgresRepository) DeleteExpired(ctx context.Context) error {
    query := `
        DELETE FROM carts 
        WHERE expires_at < NOW()
    `
    
    result, err := r.db.Exec(ctx, query)
    if err != nil {
        return err
    }
    
    rowsAffected := result.RowsAffected()
    if rowsAffected > 0 {
        log.Info().Int64("deleted_carts", rowsAffected).Msg("Expired carts deleted")
    }
    
    return nil
}
```


***

### **☐ Task 6.9: Schedule Cart Cleanup Cron Job**

File: `internal/worker/scheduler.go`

```go
package worker

import (
    "github.com/hibiken/asynq"
    "bookstore-backend/internal/worker/tasks"
)

// RegisterScheduledTasks registers periodic/cron tasks
func RegisterScheduledTasks(scheduler *asynq.Scheduler) error {
    // Cart cleanup every 10 minutes
    _, err := scheduler.Register(
        "*/10 * * * *", // Cron: every 10 minutes
        asynq.NewTask(tasks.TypeCartCleanup, nil),
        asynq.Queue(queue.QueueLow),
    )
    if err != nil {
        return err
    }
    
    return nil
}
```

**Create Scheduler in worker main:**

File: `cmd/worker/main.go` (update)

```go
// ... after creating server ...

// Create scheduler for periodic tasks
location, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
    Location: location,
    LogLevel: asynq.InfoLevel,
})

// Register scheduled tasks
if err := worker.RegisterScheduledTasks(scheduler); err != nil {
    log.Fatal("Failed to register scheduled tasks:", err)
}

// Start scheduler in goroutine
go func() {
    if err := scheduler.Run(); err != nil {
        log.Fatal("Scheduler error:", err)
    }
}()

// ... rest of the code ...
```


***

## **TUẦN 11 - NGÀY 5-6: ORDER STATUS SYNC JOB**

### **☐ Task 6.10: Create Order Status Update Task**

File: `internal/worker/tasks/order_status.go`

```go
package tasks

const (
    TypeUpdateOrderStatus = "order:update_status"
)

type UpdateOrderStatusPayload struct {
    OrderID   string `json:"order_id"`
    NewStatus string `json:"new_status"`
    Note      string `json:"note,omitempty"`
}
```


***

### **☐ Task 6.11: Create Order Status Update Handler**

File: `internal/worker/handlers/order_status_handler.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "github.com/hibiken/asynq"
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    "bookstore-backend/internal/domains/order/repository"
    "bookstore-backend/internal/domains/order/model"
    "bookstore-backend/internal/worker/tasks"
)

type OrderStatusHandler struct {
    orderRepo repository.OrderRepository
}

func NewOrderStatusHandler(orderRepo repository.OrderRepository) *OrderStatusHandler {
    return &OrderStatusHandler{orderRepo: orderRepo}
}

func (h *OrderStatusHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    var payload tasks.UpdateOrderStatusPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %w", err)
    }
    
    log.Info().
        Str("order_id", payload.OrderID).
        Str("new_status", payload.NewStatus).
        Msg("Processing order status update")
    
    orderID, err := uuid.Parse(payload.OrderID)
    if err != nil {
        return fmt.Errorf("invalid order ID: %w", err)
    }
    
    // Get current order
    order, err := h.orderRepo.FindByID(ctx, orderID)
    if err != nil {
        return err
    }
    
    // Update status
    if err := h.orderRepo.UpdateStatus(ctx, orderID, payload.NewStatus); err != nil {
        return err
    }
    
    // Add status history
    history := &model.OrderStatusHistory{
        OrderID:   orderID,
        OldStatus: &order.Status,
        NewStatus: payload.NewStatus,
        Note:      &payload.Note,
    }
    
    if err := h.orderRepo.AddStatusHistory(ctx, history); err != nil {
        log.Error().Err(err).Msg("Failed to add status history")
        // Don't fail the job if history fails
    }
    
    log.Info().
        Str("order_id", payload.OrderID).
        Str("old_status", order.Status).
        Str("new_status", payload.NewStatus).
        Msg("Order status updated successfully")
    
    return nil
}
```


***

### **☐ Task 6.12: Enqueue Order Status Update from Services**

**Example usage in Order Service:**

File: `internal/domains/order/service/order_service.go` (update)

```go
type OrderService struct {
    orderRepo   repository.OrderRepository
    addressRepo repository.AddressRepository
    cartService *cartService.CartService
    queueClient *queue.Client // ADD THIS
}

func NewOrderService(
    orderRepo repository.OrderRepository,
    addressRepo repository.AddressRepository,
    cartService *cartService.CartService,
    queueClient *queue.Client, // ADD THIS
) *OrderService {
    return &OrderService{
        orderRepo:   orderRepo,
        addressRepo: addressRepo,
        cartService: cartService,
        queueClient: queueClient,
    }
}

// Example: Schedule auto-confirm after 1 hour for COD orders
func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
    // ... existing order creation code ...
    
    // If payment method is COD, auto-confirm after 1 hour
    if req.PaymentMethod == "cod" {
        payload := tasks.UpdateOrderStatusPayload{
            OrderID:   order.ID.String(),
            NewStatus: "confirmed",
            Note:      "Auto-confirmed COD order",
        }
        
        s.queueClient.EnqueueIn(
            tasks.TypeUpdateOrderStatus,
            payload,
            1*time.Hour,
            asynq.Queue(queue.QueueDefault),
            asynq.MaxRetry(3),
        )
    }
    
    return order, nil
}
```


***

## **TUẦN 12 - NGÀY 7-8: DAILY STATISTICS JOB**

### **☐ Task 6.13: Create Daily Stats Task**

File: `internal/worker/tasks/daily_stats.go`

```go
package tasks

import (
    "time"
)

const (
    TypeDailyStats = "stats:daily"
)

type DailyStatsPayload struct {
    Date string `json:"date"` // YYYY-MM-DD
}
```


***

### **☐ Task 6.14: Create Daily Stats Table**

File: `migrations/000013_create_daily_stats.up.sql`

```sql
CREATE TABLE daily_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE UNIQUE NOT NULL,
    
    -- Order stats
    total_orders INT DEFAULT 0,
    completed_orders INT DEFAULT 0,
    cancelled_orders INT DEFAULT 0,
    
    -- Revenue stats
    total_revenue NUMERIC(12,2) DEFAULT 0,
    total_discount NUMERIC(12,2) DEFAULT 0,
    total_shipping_fee NUMERIC(12,2) DEFAULT 0,
    net_revenue NUMERIC(12,2) DEFAULT 0,
    
    -- User stats
    new_users INT DEFAULT 0,
    active_users INT DEFAULT 0,
    
    -- Product stats
    total_books_sold INT DEFAULT 0,
    top_selling_book_id UUID,
    top_selling_book_count INT DEFAULT 0,
    
    -- Payment method breakdown
    cod_orders INT DEFAULT 0,
    vnpay_orders INT DEFAULT 0,
    momo_orders INT DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_daily_stats_date ON daily_statistics(date DESC);

CREATE TRIGGER daily_statistics_updated_at BEFORE UPDATE ON daily_statistics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

File: `migrations/000013_create_daily_stats.down.sql`

```sql
DROP TABLE IF EXISTS daily_statistics;
```

Run migration:

```bash
./scripts/migrate.sh up
```


***

### **☐ Task 6.15: Create Daily Stats Handler**

File: `internal/worker/handlers/daily_stats_handler.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "time"
    "github.com/hibiken/asynq"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog/log"
    "bookstore-backend/internal/worker/tasks"
)

type DailyStatsHandler struct {
    db *pgxpool.Pool
}

func NewDailyStatsHandler(db *pgxpool.Pool) *DailyStatsHandler {
    return &DailyStatsHandler{db: db}
}

func (h *DailyStatsHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    var payload tasks.DailyStatsPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return err
    }
    
    date, err := time.Parse("2006-01-02", payload.Date)
    if err != nil {
        return err
    }
    
    log.Info().
        Str("date", payload.Date).
        Msg("Generating daily statistics")
    
    // Calculate statistics
    query := `
        INSERT INTO daily_statistics (
            date,
            total_orders,
            completed_orders,
            cancelled_orders,
            total_revenue,
            total_discount,
            total_shipping_fee,
            net_revenue,
            new_users,
            active_users,
            total_books_sold,
            cod_orders,
            vnpay_orders,
            momo_orders
        )
        SELECT 
            $1::date,
            COUNT(*) FILTER (WHERE o.created_at::date = $1::date),
            COUNT(*) FILTER (WHERE o.status = 'completed' AND o.created_at::date = $1::date),
            COUNT(*) FILTER (WHERE o.status = 'cancelled' AND o.created_at::date = $1::date),
            COALESCE(SUM(o.total) FILTER (WHERE o.created_at::date = $1::date), 0),
            COALESCE(SUM(o.discount_amount) FILTER (WHERE o.created_at::date = $1::date), 0),
            COALESCE(SUM(o.shipping_fee) FILTER (WHERE o.created_at::date = $1::date), 0),
            COALESCE(SUM(o.total - o.discount_amount) FILTER (WHERE o.created_at::date = $1::date), 0),
            (SELECT COUNT(*) FROM users WHERE created_at::date = $1::date),
            (SELECT COUNT(DISTINCT user_id) FROM orders WHERE created_at::date = $1::date),
            COALESCE(SUM(oi.quantity) FILTER (WHERE o.created_at::date = $1::date), 0),
            COUNT(*) FILTER (WHERE o.payment_method = 'cod' AND o.created_at::date = $1::date),
            COUNT(*) FILTER (WHERE o.payment_method = 'vnpay' AND o.created_at::date = $1::date),
            COUNT(*) FILTER (WHERE o.payment_method = 'momo' AND o.created_at::date = $1::date)
        FROM orders o
        LEFT JOIN order_items oi ON o.id = oi.order_id
        WHERE o.created_at::date = $1::date
        ON CONFLICT (date) DO UPDATE SET
            total_orders = EXCLUDED.total_orders,
            completed_orders = EXCLUDED.completed_orders,
            cancelled_orders = EXCLUDED.cancelled_orders,
            total_revenue = EXCLUDED.total_revenue,
            total_discount = EXCLUDED.total_discount,
            total_shipping_fee = EXCLUDED.total_shipping_fee,
            net_revenue = EXCLUDED.net_revenue,
            new_users = EXCLUDED.new_users,
            active_users = EXCLUDED.active_users,
            total_books_sold = EXCLUDED.total_books_sold,
            cod_orders = EXCLUDED.cod_orders,
            vnpay_orders = EXCLUDED.vnpay_orders,
            momo_orders = EXCLUDED.momo_orders,
            updated_at = NOW()
    `
    
    _, err = h.db.Exec(ctx, query, date)
    if err != nil {
        log.Error().Err(err).Str("date", payload.Date).Msg("Failed to generate daily stats")
        return err
    }
    
    // Calculate top selling book
    topBookQuery := `
        WITH daily_book_sales AS (
            SELECT 
                oi.book_id,
                SUM(oi.quantity) as total_sold
            FROM order_items oi
            JOIN orders o ON oi.order_id = o.id
            WHERE o.created_at::date = $1::date
            GROUP BY oi.book_id
            ORDER BY total_sold DESC
            LIMIT 1
        )
        UPDATE daily_statistics
        SET 
            top_selling_book_id = (SELECT book_id FROM daily_book_sales),
            top_selling_book_count = (SELECT total_sold FROM daily_book_sales)
        WHERE date = $1::date
    `
    
    _, err = h.db.Exec(ctx, topBookQuery, date)
    if err != nil {
        log.Error().Err(err).Msg("Failed to update top selling book")
        // Don't fail the job
    }
    
    log.Info().
        Str("date", payload.Date).
        Msg("Daily statistics generated successfully")
    
    return nil
}
```


***

### **☐ Task 6.16: Schedule Daily Stats Cron Job**

File: `internal/worker/scheduler.go` (update)

```go
func RegisterScheduledTasks(scheduler *asynq.Scheduler) error {
    // ... existing cart cleanup ...
    
    // Daily statistics - run every day at 1:00 AM
    _, err := scheduler.Register(
        "0 1 * * *", // Cron: 1:00 AM daily
        asynq.NewTask(tasks.TypeDailyStats, []byte(fmt.Sprintf(`{"date":"%s"}`, time.Now().AddDate(0, 0, -1).Format("2006-01-02")))),
        asynq.Queue(queue.QueueLow),
        asynq.MaxRetry(5),
    )
    if err != nil {
        return err
    }
    
    return nil
}
```


***

## **TUẦN 12 - NGÀY 9: INVENTORY SYNC \& LOW STOCK ALERT**

### **☐ Task 6.17: Create Inventory Sync Task**

File: `internal/worker/tasks/inventory_sync.go`

```go
package tasks

const (
    TypeSyncInventory = "inventory:sync"
)

type SyncInventoryPayload struct {
    WarehouseID string `json:"warehouse_id"`
    BookID      string `json:"book_id"`
}
```


***

### **☐ Task 6.18: Create Inventory Sync Handler**

File: `internal/worker/handlers/inventory_sync_handler.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "github.com/hibiken/asynq"
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    "github.com/jackc/pgx/v5/pgxpool"
    "bookstore-backend/internal/worker/tasks"
)

type InventorySyncHandler struct {
    db *pgxpool.Pool
}

func NewInventorySyncHandler(db *pgxpool.Pool) *InventorySyncHandler {
    return &InventorySyncHandler{db: db}
}

func (h *InventorySyncHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    var payload tasks.SyncInventoryPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return err
    }
    
    warehouseID, err := uuid.Parse(payload.WarehouseID)
    if err != nil {
        return err
    }
    
    bookID, err := uuid.Parse(payload.BookID)
    if err != nil {
        return err
    }
    
    log.Info().
        Str("warehouse_id", payload.WarehouseID).
        Str("book_id", payload.BookID).
        Msg("Syncing inventory")
    
    // In real system, this would call external warehouse API
    // For now, just log
    
    // Update last_restocked_at
    query := `
        UPDATE warehouse_inventory
        SET last_restocked_at = NOW()
        WHERE warehouse_id = $1 AND book_id = $2
    `
    
    _, err = h.db.Exec(ctx, query, warehouseID, bookID)
    if err != nil {
        return err
    }
    
    log.Info().Msg("Inventory synced successfully")
    return nil
}
```


***

### **☐ Task 6.19: Create Low Stock Alert Task**

File: `internal/worker/tasks/low_stock_alert.go`

```go
package tasks

const (
    TypeLowStockAlert = "inventory:low_stock_alert"
)

type LowStockAlertPayload struct {
    BookID      string `json:"book_id"`
    BookTitle   string `json:"book_title"`
    WarehouseID string `json:"warehouse_id"`
    Warehouse   string `json:"warehouse_name"`
    CurrentStock int   `json:"current_stock"`
    Threshold    int   `json:"threshold"`
}
```


***

### **☐ Task 6.20: Create Low Stock Alert Handler**

File: `internal/worker/handlers/low_stock_alert_handler.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"
    "bookstore-backend/internal/worker/tasks"
)

type LowStockAlertHandler struct {
    // In Sprint 8, this will send email
    // For now, just log
}

func NewLowStockAlertHandler() *LowStockAlertHandler {
    return &LowStockAlertHandler{}
}

func (h *LowStockAlertHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    var payload tasks.LowStockAlertPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return err
    }
    
    log.Warn().
        Str("book_id", payload.BookID).
        Str("book_title", payload.BookTitle).
        Str("warehouse", payload.Warehouse).
        Int("current_stock", payload.CurrentStock).
        Int("threshold", payload.Threshold).
        Msg("LOW STOCK ALERT")
    
    // TODO Sprint 8: Send email to admin
    message := fmt.Sprintf(
        "Low stock alert: '%s' at %s has only %d items left (threshold: %d)",
        payload.BookTitle,
        payload.Warehouse,
        payload.CurrentStock,
        payload.Threshold,
    )
    
    log.Info().Msg(message)
    
    return nil
}
```


***

### **☐ Task 6.21: Schedule Low Stock Check**

File: `internal/worker/scheduler.go` (update)

```go
func RegisterScheduledTasks(scheduler *asynq.Scheduler, queueClient *queue.Client, db *pgxpool.Pool) error {
    // ... existing tasks ...
    
    // Low stock check - every hour
    _, err := scheduler.Register(
        "0 * * * *", // Every hour
        asynq.NewTask("inventory:check_low_stock", nil),
        asynq.Queue(queue.QueueDefault),
    )
    if err != nil {
        return err
    }
    
    return nil
}
```

File: `internal/worker/handlers/check_low_stock_handler.go`

```go
package handlers

import (
    "context"
    "github.com/hibiken/asynq"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog/log"
    "bookstore-backend/pkg/queue"
    "bookstore-backend/internal/worker/tasks"
)

type CheckLowStockHandler struct {
    db          *pgxpool.Pool
    queueClient *queue.Client
}

func NewCheckLowStockHandler(db *pgxpool.Pool, queueClient *queue.Client) *CheckLowStockHandler {
    return &CheckLowStockHandler{
        db:          db,
        queueClient: queueClient,
    }
}

func (h *CheckLowStockHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    log.Info().Msg("Checking for low stock items")
    
    query := `
        SELECT 
            wi.book_id,
            b.title,
            wi.warehouse_id,
            w.name,
            wi.quantity,
            wi.alert_threshold
        FROM warehouse_inventory wi
        JOIN books b ON wi.book_id = b.id
        JOIN warehouses w ON wi.warehouse_id = w.id
        WHERE wi.quantity <= wi.alert_threshold
          AND wi.quantity > 0
          AND b.is_active = true
    `
    
    rows, err := h.db.Query(ctx, query)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    alertCount := 0
    for rows.Next() {
        var bookID, warehouseID, bookTitle, warehouseName string
        var quantity, threshold int
        
        err := rows.Scan(&bookID, &bookTitle, &warehouseID, &warehouseName, &quantity, &threshold)
        if err != nil {
            log.Error().Err(err).Msg("Failed to scan row")
            continue
        }
        
        // Enqueue alert
        payload := tasks.LowStockAlertPayload{
            BookID:       bookID,
            BookTitle:    bookTitle,
            WarehouseID:  warehouseID,
            Warehouse:    warehouseName,
            CurrentStock: quantity,
            Threshold:    threshold,
        }
        
        h.queueClient.Enqueue(
            tasks.TypeLowStockAlert,
            payload,
            asynq.Queue(queue.QueueDefault),
        )
        
        alertCount++
    }
    
    log.Info().Int("alerts", alertCount).Msg("Low stock check completed")
    return nil
}
```


***

## **TUẦN 12 - NGÀY 10: TESTING \& MONITORING**

### **☐ Task 6.22: Register All Handlers**

File: `internal/worker/registry.go`

```go
package worker

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
    "bookstore-backend/pkg/queue"
    "bookstore-backend/internal/domains/cart/repository" as cartRepo
    "bookstore-backend/internal/domains/order/repository" as orderRepo
    "bookstore-backend/internal/worker/handlers"
    "bookstore-backend/internal/worker/tasks"
)

func RegisterHandlers(
    server *queue.Server,
    db *pgxpool.Pool,
    redisClient *redis.Client,
    cartRepo cartRepo.CartRepository,
    orderRepo orderRepo.OrderRepository,
    queueClient *queue.Client,
) {
    // Cart cleanup handler
    cartCleanupHandler := handlers.NewCartCleanupHandler(cartRepo)
    server.RegisterHandlerFunc(tasks.TypeCartCleanup, cartCleanupHandler.ProcessTask)
    
    // Order status update handler
    orderStatusHandler := handlers.NewOrderStatusHandler(orderRepo)
    server.RegisterHandlerFunc(tasks.TypeUpdateOrderStatus, orderStatusHandler.ProcessTask)
    
    // Daily statistics handler
    dailyStatsHandler := handlers.NewDailyStatsHandler(db)
    server.RegisterHandlerFunc(tasks.TypeDailyStats, dailyStatsHandler.ProcessTask)
    
    // Inventory sync handler
    inventorySyncHandler := handlers.NewInventorySyncHandler(db)
    server.RegisterHandlerFunc(tasks.TypeSyncInventory, inventorySyncHandler.ProcessTask)
    
    // Low stock alert handler
    lowStockAlertHandler := handlers.NewLowStockAlertHandler()
    server.RegisterHandlerFunc(tasks.TypeLowStockAlert, lowStockAlertHandler.ProcessTask)
    
    // Check low stock handler (scheduled)
    checkLowStockHandler := handlers.NewCheckLowStockHandler(db, queueClient)
    server.RegisterHandlerFunc("inventory:check_low_stock", checkLowStockHandler.ProcessTask)
}
```


***

### **☐ Task 6.23: Create Health Check Endpoint**

File: `cmd/api/main.go` (update)

```go
// Health check routes
r.GET("/health", func(c *gin.Context) {
    // Check database
    if err := pgPool.Ping(context.Background()); err != nil {
        c.JSON(500, gin.H{"status": "unhealthy", "database": "down"})
        return
    }
    
    // Check Redis
    if err := redisClient.Ping(context.Background()).Err(); err != nil {
        c.JSON(500, gin.H{"status": "unhealthy", "redis": "down"})
        return
    }
    
    c.JSON(200, gin.H{
        "status":   "healthy",
        "database": "up",
        "redis":    "up",
    })
})

// Worker health check
r.GET("/health/worker", func(c *gin.Context) {
    // Check Asynq queues
    inspector := asynq.NewInspector(redisOpt)
    
    stats, err := inspector.GetQueueStats(queue.QueueCritical)
    if err != nil {
        c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "status": "healthy",
        "queues": gin.H{
            "critical_pending": stats.Pending,
            "critical_active":  stats.Active,
        },
    })
})
```


***

### **☐ Task 6.24: Create Unit Tests for Handlers**

File: `internal/worker/handlers/cart_cleanup_handler_test.go`

```go
package handlers_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/hibiken/asynq"
    "bookstore-backend/internal/worker/handlers"
)

type MockCartRepository struct {
    mock.Mock
}

func (m *MockCartRepository) DeleteExpired(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}

func TestCartCleanupHandler_ProcessTask_Success(t *testing.T) {
    mockRepo := new(MockCartRepository)
    handler := handlers.NewCartCleanupHandler(mockRepo)
    
    mockRepo.On("DeleteExpired", mock.Anything).Return(nil)
    
    task := asynq.NewTask("cart:cleanup", nil)
    err := handler.ProcessTask(context.Background(), task)
    
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}

func TestCartCleanupHandler_ProcessTask_Error(t *testing.T) {
    mockRepo := new(MockCartRepository)
    handler := handlers.NewCartCleanupHandler(mockRepo)
    
    mockRepo.On("DeleteExpired", mock.Anything).Return(errors.New("db error"))
    
    task := asynq.NewTask("cart:cleanup", nil)
    err := handler.ProcessTask(context.Background(), task)
    
    assert.Error(t, err)
    mockRepo.AssertExpectations(t)
}
```


***

### **☐ Task 6.25: Create Integration Tests**

File: `tests/integration/worker_test.go`

```go
package integration_test

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/testcontainers/testcontainers-go"
    "bookstore-backend/pkg/queue"
    "bookstore-backend/internal/worker/tasks"
)

func TestWorker_CartCleanup_Integration(t *testing.T) {
    // Setup testcontainers (PostgreSQL + Redis)
    ctx := context.Background()
    
    pgContainer, _ := setupPostgresContainer(ctx)
    defer pgContainer.Terminate(ctx)
    
    redisContainer, _ := setupRedisContainer(ctx)
    defer redisContainer.Terminate(ctx)
    
    // Create queue client
    redisOpt := queue.RedisConnOpt("localhost:6379", "", 0)
    client := queue.NewClient(redisOpt)
    defer client.Close()
    
    // Enqueue task
    err := client.Enqueue(tasks.TypeCartCleanup, tasks.CartCleanupPayload{})
    assert.NoError(t, err)
    
    // Wait for processing
    time.Sleep(2 * time.Second)
    
    // Verify carts were deleted (check database)
    // ... assertions ...
}
```


***

### **☐ Task 6.26: Monitor Asynq with Web UI**

**Install Asynq Web UI:**

```bash
# Install asynqmon
go install github.com/hibiken/asynqmon/cmd/asynqmon@latest

# Run web UI
asynqmon --redis-addr=localhost:6379
```

Open browser: `http://localhost:8080`

**Features:**

- View queues statistics
- Monitor active/pending/completed tasks
- View failed tasks
- Manually retry failed tasks
- View task details \& payloads

***

### **☐ Task 6.27: Create Monitoring Metrics**

File: `pkg/queue/metrics.go`

```go
package queue

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    TasksEnqueued = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "asynq_tasks_enqueued_total",
            Help: "Total number of tasks enqueued",
        },
        []string{"task_type", "queue"},
    )
    
    TasksProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "asynq_tasks_processed_total",
            Help: "Total number of tasks processed",
        },
        []string{"task_type", "status"}, // status: success, failed
    )
    
    TaskDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "asynq_task_duration_seconds",
            Help:    "Task processing duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"task_type"},
    )
)
```

**Instrument handlers:**

```go
func (h *CartCleanupHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    start := time.Now()
    
    err := h.doCleanup(ctx)
    
    duration := time.Since(start)
    queue.TaskDuration.WithLabelValues(task.Type()).Observe(duration.Seconds())
    
    if err != nil {
        queue.TasksProcessed.WithLabelValues(task.Type(), "failed").Inc()
    } else {
        queue.TasksProcessed.WithLabelValues(task.Type(), "success").Inc()
    }
    
    return err
}
```


***

### **☐ Task 6.28: Create Documentation**

File: `docs/worker/README.md`

````markdown
# Background Worker Documentation

## Overview

Background worker processes asynchronous tasks using Asynq + Redis.

## Queue Priorities

| Queue | Priority | Concurrency | Use Cases |
|-------|----------|-------------|-----------|
| critical | 10 | High | Payment processing, inventory reservation |
| high | 5 | Medium | Emails, notifications |
| default | 3 | Medium | General background tasks |
| low | 1 | Low | Analytics, cleanup |

## Registered Tasks

### cart:cleanup
**Trigger:** Cron (every 10 minutes)  
**Queue:** low  
**Retry:** 1  
**Timeout:** 60s  
**Purpose:** Delete expired shopping carts

### order:update_status
**Trigger:** Manual enqueue  
**Queue:** default  
**Retry:** 3  
**Timeout:** 30s  
**Purpose:** Update order status and add history

### stats:daily
**Trigger:** Cron (1:00 AM daily)  
**Queue:** low  
**Retry:** 5  
**Timeout:** 300s  
**Purpose:** Generate daily statistics

### inventory:sync
**Trigger:** Manual enqueue  
**Queue:** default  
**Retry:** 3  
**Timeout:** 120s  
**Purpose:** Sync inventory with external warehouse

### inventory:low_stock_alert
**Trigger:** Enqueued by check_low_stock  
**Queue:** default  
**Retry:** 2  
**Timeout:** 30s  
**Purpose:** Send low stock alerts

## Running Worker

```
# Development
go run cmd/worker/main.go

# Production
./worker

# With Docker
docker-compose up worker
```

## Monitoring

### Web UI
```
asynqmon --redis-addr=localhost:6379
```

### Metrics Endpoint
```
GET /metrics
```

### Health Check
```
GET /health/worker
```

## Testing

```
# Unit tests
go test ./internal/worker/handlers/... -v

# Integration tests
go test ./tests/integration/worker_test.go -v

# Load test
./scripts/load_test_worker.sh
```

## Troubleshooting

### Worker not processing tasks
1. Check Redis connection
2. Check queue registration
3. Check handler registration
4. View logs: `tail -f logs/worker.log`

### Tasks failing repeatedly
1. Check Asynq web UI for error details
2. View failed tasks payload
3. Manually retry from UI
4. Check database connectivity

### High queue backlog
1. Increase worker concurrency
2. Scale horizontally (multiple worker instances)
3. Optimize slow handlers
```

***

## **TESTING CHECKLIST SPRINT 6**

### **Infrastructure Tests**
- [ ] Asynq client connection successful
- [ ] Asynq server starts without errors
- [ ] All queues registered with correct priorities
- [ ] Graceful shutdown works (no task loss)
- [ ] Health check endpoints return 200

### **Cart Cleanup Tests**
- [ ] Expired carts are deleted
- [ ] Active carts are not deleted
- [ ] Cron job runs every 10 minutes
- [ ] Failed cleanup retries correctly

### **Order Status Tests**
- [ ] Status update task processes successfully
- [ ] Status history is created
- [ ] Invalid order ID handled gracefully
- [ ] Retry works on transient errors

### **Daily Stats Tests**
- [ ] Statistics calculated correctly
- [ ] Cron runs at 1:00 AM
- [ ] Top selling book identified
- [ ] Revenue calculations accurate
- [ ] Idempotent (re-run same date)

### **Inventory Tests**
- [ ] Low stock check identifies items correctly
- [ ] Alerts enqueued for low stock items
- [ ] Sync task updates last_restocked_at
- [ ] No alerts for out-of-stock items (quantity = 0)

### **Monitoring Tests**
- [ ] Prometheus metrics exported
- [ ] Asynq web UI accessible
- [ ] Task duration tracked
- [ ] Success/failure counters working

### **Load Tests**
- [ ] 1000 tasks/minute processed
- [ ] No queue backlog under normal load
- [ ] Graceful degradation under high load
- [ ] No memory leaks after 24h operation

***

## **DEPLOYMENT CHECKLIST**

### **Environment Variables**
```bash
# Add to .env
REDIS_HOST=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

WORKER_CONCURRENCY=20
WORKER_QUEUES_CRITICAL_PRIORITY=10
WORKER_QUEUES_HIGH_PRIORITY=5
WORKER_QUEUES_DEFAULT_PRIORITY=3
WORKER_QUEUES_LOW_PRIORITY=1
```

### **Docker Compose**
```yaml
# docker-compose.yml
services:
  worker:
    build: .
    command: ./worker
    environment:
      - REDIS_HOST=redis:6379
      - DB_HOST=postgres
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
```

### **Systemd Service (Optional)**
```ini
# /etc/systemd/system/bookstore-worker.service
[Unit]
Description=Bookstore Background Worker
After=network.target redis.service postgresql.service

[Service]
Type=simple
User=bookstore
WorkingDirectory=/opt/bookstore
ExecStart=/opt/bookstore/worker
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

***

## **NEXT STEPS - SPRINT 7**

Sprint 7 sẽ focus vào **Payment Integration** (VNPay, Momo, COD) như đã chuẩn bị ở bản TO-DO cũ.

**Deliverables Sprint 7:**
- VNPay payment gateway integration
- Momo payment gateway integration
- Payment webhook handling
- Payment timeout mechanism (sử dụng Asynq đã setup Sprint 6)
- Refund processing

***

**Sprint 6 hoàn thành! Bạn đã có hạ tầng background jobs đầy đủ với Asynq, 6 background jobs hoạt động, monitoring, và sẵn sàng cho Sprint 7 (Payment).**````

