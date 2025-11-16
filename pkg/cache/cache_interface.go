package cache

import (
	"context"
	"time"
)

// Cache interface định nghĩa contract cho cache layer
// Cho phép swap implementation (Redis, Memcached, In-memory)
type Cache interface {
	// Get lấy data từ cache và unmarshal vào dest
	// Returns: (found bool, error)
	// - found = true: cache hit, data đã unmarshal vào dest
	// - found = false: cache miss, dest không bị thay đổi
	Get(ctx context.Context, key string, dest interface{}) (bool, error)

	// Set lưu data vào cache với TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete xóa các keys khỏi cache
	Delete(ctx context.Context, keys ...string) error

	// Ping kiểm tra connection
	Ping(ctx context.Context) error

	DeletePattern(ctx context.Context, pattern string) error // ← THÊM METHOD MỚI
	// ✅ Thêm methods cho failed login tracking
	Increment(ctx context.Context, key string) (int64, error)
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
}
