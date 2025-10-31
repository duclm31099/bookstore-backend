package config

import (
	"fmt"
	"strconv"
	"time"

	"bookstore-backend/internal/infrastructure/database"
)

// LoadDatabaseConfig đọc config từ environment variables và trả về DBConfig
func LoadDatabaseConfig() (*database.DBConfig, error) {
	// Parse integers
	port, err := strconv.Atoi(getEnv("DB_PORT", "5439"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	maxConns, err := strconv.Atoi(getEnv("DB_MAX_CONNECTIONS", "25"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_CONNECTIONS: %w", err)
	}

	minConns, err := strconv.Atoi(getEnv("DB_MIN_CONNECTIONS", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MIN_CONNECTIONS: %w", err)
	}

	maxRetries, err := strconv.Atoi(getEnv("DB_MAX_RETRIES", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_RETRIES: %w", err)
	}

	// Parse durations
	maxConnLifetime, err := time.ParseDuration(getEnv("DB_MAX_CONN_LIFETIME", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_CONN_LIFETIME: %w", err)
	}

	maxConnIdleTime, err := time.ParseDuration(getEnv("DB_MAX_CONN_IDLE_TIME", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_CONN_IDLE_TIME: %w", err)
	}

	healthCheckPeriod, err := time.ParseDuration(getEnv("DB_HEALTH_CHECK_PERIOD", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_HEALTH_CHECK_PERIOD: %w", err)
	}

	retryDelay, err := time.ParseDuration(getEnv("DB_RETRY_DELAY", "1s"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_RETRY_DELAY: %w", err)
	}

	connectTimeout, err := time.ParseDuration(getEnv("DB_CONNECT_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_CONNECT_TIMEOUT: %w", err)
	}

	return &database.DBConfig{
		Host:              getEnv("DB_HOST", "localhost"),
		Port:              port,
		Username:          getEnv("DB_USER", "bookstore"),
		Password:          getEnv("DB_PASSWORD", "secret"),
		DBName:            getEnv("DB_NAME", "bookstore_dev"),
		MaxConns:          int32(maxConns),
		MinConns:          int32(minConns),
		MaxConnLifetime:   maxConnLifetime,
		MaxConnIdleTime:   maxConnIdleTime,
		HealthCheckPeriod: healthCheckPeriod,
		MaxRetries:        maxRetries,
		RetryDelay:        retryDelay,
		ConnectTimeout:    connectTimeout,
	}, nil
}
