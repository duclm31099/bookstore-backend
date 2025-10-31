package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(host, password string, db int) *RedisClient {
	return &RedisClient{
		Client: redis.NewClient(&redis.Options{
			Addr:         host,
			Password:     password,
			DB:           db,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}),
	}
}

func (r *RedisClient) Connect(ctx context.Context) error {
	log.Println("[REDIS] Connecting to Redis...")

	// Ping to verify connection
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	log.Println("[REDIS] Connected successfully")
	return nil
}

func (r *RedisClient) HealthCheck(ctx context.Context) error {
	if r.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func (r *RedisClient) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}
