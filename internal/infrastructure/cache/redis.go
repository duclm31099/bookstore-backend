package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	// Import cache interface từ pkg
	pkgCache "bookstore-backend/pkg/cache"
	"bookstore-backend/pkg/logger"
)

// RedisCache implements pkg/cache.Cache interface
// Đổi tên từ RedisClient -> RedisCache để rõ ràng hơn
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache tạo Redis cache instance
// QUAN TRỌNG: Return pkg/cache.Cache interface, không phải concrete type
func NewRedisCache(host, password string, db int) pkgCache.Cache {
	client := redis.NewClient(&redis.Options{
		Addr:         host,
		Password:     password,
		DB:           db,
		PoolSize:     10, // Connection pool size
		MinIdleConns: 5,  // Minimum idle connections
		MaxRetries:   3,  // Retry failed commands
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &RedisCache{
		client: client,
	}
}

// Connect khởi tạo kết nối Redis (gọi khi startup)
func (r *RedisCache) Connect(ctx context.Context) error {
	log.Println("[REDIS] Connecting to Redis...")

	// Ping để verify connection
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	log.Println("[REDIS] Connected successfully")
	return nil
}

// HealthCheck kiểm tra Redis health (cho health check endpoint)
func (r *RedisCache) HealthCheck(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	// Set timeout ngắn cho health check
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

// Close đóng Redis connection (gọi khi shutdown)
func (r *RedisCache) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// ========================================
// IMPLEMENT pkg/cache.Cache INTERFACE
// ========================================

// Get implements cache.Cache interface
// Lấy data từ Redis và unmarshal vào dest
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	// Get raw bytes từ Redis
	val, err := r.client.Get(ctx, key).Bytes()

	// Cache miss: key không tồn tại trong Redis
	if err == redis.Nil {
		return false, nil
	}

	// Redis error (network, timeout, etc.)
	if err != nil {
		// Log error nhưng KHÔNG return error
		// Tránh cache failure làm crash application
		message := fmt.Sprintf("[REDIS] Get error for key %s: %v", key, err)
		logger.Info("Get Redis error", map[string]interface{}{
			"error": message,
		})
		return false, nil // Treat as cache miss
	}

	// Unmarshal JSON vào dest struct
	if err := json.Unmarshal(val, dest); err != nil {
		// Unmarshal error: data corrupted hoặc schema changed
		log.Printf("[REDIS] Unmarshal error for key %s: %v", key, err)
		// Delete corrupted data
		_ = r.client.Del(ctx, key)
		return false, nil
	}

	// Cache hit
	return true, nil
}

// Set implements cache.Cache interface
// Lưu data vào Redis với TTL
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Marshal struct thành JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	// Set vào Redis với expiration
	err = r.client.Set(ctx, key, jsonData, ttl).Err()
	if err != nil {
		// Log error nhưng KHÔNG crash application
		log.Printf("[REDIS] Set error for key %s: %v", key, err)
		return nil // Fail silently - cache failure shouldn't break app
	}

	return nil
}

// Delete implements cache.Cache interface
// Xóa một hoặc nhiều keys khỏi Redis
func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		// Log error nhưng không return
		log.Printf("[REDIS] Delete error for keys %v: %v", keys, err)
		return nil // Fail silently
	}

	return nil
}

// Ping implements cache.Cache interface
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// ========================================
// ADDITIONAL HELPER METHODS (OPTIONAL)
// ========================================

// Exists kiểm tra key có tồn tại không
func (r *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Expire set TTL cho key đã tồn tại
func (r *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// GetTTL lấy thời gian còn lại của key
func (r *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// FlushDB xóa toàn bộ database (CHỈ dùng trong testing)
func (r *RedisCache) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// IncrBy atomic increment (cho counters, rate limiting)
func (r *RedisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

// SetNX set key nếu chưa tồn tại (distributed lock)
func (r *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return r.client.SetNX(ctx, key, jsonData, ttl).Result()
}
