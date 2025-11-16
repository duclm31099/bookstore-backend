package main

import (
	"log"

	"bookstore-backend/internal/shared/utils"
)

// Config holds all configuration for the worker
type Config struct {
	RedisAddr string
	SMTPHost  string
	SMTPPort  string
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	cfg := &Config{
		RedisAddr: utils.GetEnvVariable("REDIS_HOST", "localhost:6379"),
		SMTPHost:  utils.GetEnvVariable("SMTP_HOST", "localhost"),
		SMTPPort:  utils.GetEnvVariable("SMTP_PORT", "1025"),
	}

	log.Printf("[Config] Redis: %s, SMTP: %s:%s",
		cfg.RedisAddr, cfg.SMTPHost, cfg.SMTPPort)

	return cfg
}
