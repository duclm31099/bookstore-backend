package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

type AppConfig struct {
	Name        string
	Environment string
	Port        string
	Version     string
	URL         string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host        string
	Password    string
	DB          int
	MaxRetries  int
	PoolSize    int
	DialTimeout time.Duration
}

type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	RefreshExpiration time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "Bookstore Backend"),
			Environment: getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			URL:         getEnv("APP_URL", "http://localhost:8080"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "bookstore"),
			Password:        getEnv("DB_PASSWORD", "secret"),
			Name:            getEnv("DB_NAME", "bookstore_dev"),
			MaxConnections:  getEnvInt("DB_MAX_CONNECTIONS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNECTIONS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONNECTION_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:        getEnv("REDIS_HOST", "localhost:6379"),
			Password:    getEnv("REDIS_PASSWORD", ""),
			DB:          getEnvInt("REDIS_DB", 0),
			MaxRetries:  getEnvInt("REDIS_MAX_RETRIES", 3),
			PoolSize:    getEnvInt("REDIS_POOL_SIZE", 10),
			DialTimeout: 5 * time.Second,
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "change-this-secret"),
			Expiration:        getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			RefreshExpiration: getEnvDuration("JWT_REFRESH_EXPIRATION", 168*time.Hour),
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.JWT.Secret == "change-this-secret" && c.App.Environment == "production" {
		return fmt.Errorf("JWT_SECRET must be set in production")
	}
	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}
