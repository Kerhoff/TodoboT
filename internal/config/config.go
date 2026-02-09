package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application
type Config struct {
	TelegramToken string
	DatabaseURL   string
	LogLevel      string
	Port          string
	WebhookURL    string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
		Port:     getEnvOrDefault("PORT", "8080"),
	}

	if cfg.TelegramToken = os.Getenv("TELEGRAM_TOKEN"); cfg.TelegramToken == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN environment variable is required")
	}

	if cfg.DatabaseURL = os.Getenv("DATABASE_URL"); cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	cfg.WebhookURL = os.Getenv("WEBHOOK_URL")

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
