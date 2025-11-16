package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config chứa toàn bộ application configuration
// Struct này được populate từ environment variables
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Email    EmailConfig
	VNPay    VNPayConfig
	Momo     MomoConfig
	MinIO    MinIOConfig
}
type VNPayConfig struct {
	TmnCode    string // Merchant Code (e.g., "DEMOV01")
	HashSecret string // Secret key for HMAC-SHA512
	APIURL     string // VNPay API base URL
	ReturnURL  string // Frontend callback URL
	IPNURL     string // Backend webhook URL
}
type MinIOConfig struct {
	Endpoint  string // localhost:9000
	AccessKey string // minioadmin
	SecretKey string // minioadmin
	Bucket    string // bookstore
	UseSSL    bool   // false for local
}

// =====================================================
// MOMO CONFIGURATION
// =====================================================

type MomoConfig struct {
	PartnerCode string // Partner Code
	AccessKey   string // Access Key
	SecretKey   string // Secret Key for HMAC-SHA256
	APIURL      string // Momo API base URL
	ReturnURL   string // Frontend callback URL
	IPNURL      string // Backend webhook URL
}
type AppConfig struct {
	Name        string
	Environment string // development, staging, production
	Port        string
	Version     string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int
	MinConns int
}

type RedisConfig struct {
	Host     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  int // minutes
	RefreshTokenExpiry int // hours
}

type EmailConfig struct {
	Provider string // ses, sendgrid
	APIKey   string
	From     string
}

// Load đọc config từ environment variables
func Load() (*Config, error) {
	cfg := &Config{
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("MINIO_BUCKET", "bookstore"),
			UseSSL:    false,
		},
		App: AppConfig{
			Name:        getEnv("APP_NAME", "Bookstore API"),
			Environment: getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "bookstore"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 25),
			MinConns: getEnvInt("DB_MIN_CONNS", 5),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:             getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessTokenExpiry:  getEnvInt("JWT_ACCESS_EXPIRY", 15),  // 15 minutes
			RefreshTokenExpiry: getEnvInt("JWT_REFRESH_EXPIRY", 72), // 3 days
		},
		Email: EmailConfig{
			Provider: getEnv("EMAIL_PROVIDER", "ses"),
			APIKey:   getEnv("EMAIL_API_KEY", ""),
			From:     getEnv("EMAIL_FROM", "noreply@bookstore.com"),
		},
		VNPay: VNPayConfig{
			TmnCode:    getEnv("VNPAY_TMN_CODE", ""),
			HashSecret: getEnv("VNPAY_HASH_SECRET", ""),
			APIURL:     getEnv("VNPAY_API_URL", "https://sandbox.vnpayment.vn"),
			ReturnURL:  getEnv("VNPAY_RETURN_URL", "http://localhost:3000/payment/callback"),
			IPNURL:     getEnv("VNPAY_IPN_URL", "http://localhost:8080/api/v1/webhooks/vnpay"),
		},

		// ========================================
		// MOMO CONFIGURATION
		// ========================================
		Momo: MomoConfig{
			PartnerCode: getEnv("MOMO_PARTNER_CODE", ""),
			AccessKey:   getEnv("MOMO_ACCESS_KEY", ""),
			SecretKey:   getEnv("MOMO_SECRET_KEY", ""),
			APIURL:      getEnv("MOMO_API_URL", "https://test-payment.momo.vn"),
			ReturnURL:   getEnv("MOMO_RETURN_URL", "http://localhost:3000/payment/callback"),
			IPNURL:      getEnv("MOMO_IPN_URL", "http://localhost:8080/api/v1/webhooks/momo"),
		},
	}

	// Validate critical config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate kiểm tra config có hợp lệ không
func (c *Config) Validate() error {
	// Production environment phải có JWT secret
	if c.App.Environment == "production" {
		if c.JWT.Secret == "your-secret-key-change-in-production" {
			return fmt.Errorf("JWT_SECRET must be set in production")
		}
		if c.Database.Password == "" {
			return fmt.Errorf("DB_PASSWORD must be set in production")
		}

		// Payment gateway validation (optional - only warn if not set)
		if c.VNPay.TmnCode == "" {
			fmt.Println("WARNING: VNPay TmnCode not set - VNPay payment will not work")
		}
		if c.Momo.PartnerCode == "" {
			fmt.Println("WARNING: Momo PartnerCode not set - Momo payment will not work")
		}
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
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
