// Package config application configuration
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SolanaWS string
	DBPath   string

	TelegramToken  string
	TelegramChatID string

	WebhookURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		SolanaWS: getEnv("SOLANA_WS_URL", ""),
		DBPath:   getEnv("DB_PATH", "whale.db"),

		TelegramToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID: getEnv("CHAT_ID", ""),

		WebhookURL: getEnv("WEBHOOK_URL", ""),
	}

	if cfg.SolanaWS == "" {
		return nil, fmt.Errorf("SOLANA_WS_URL is required")
	}

	return cfg, nil
}

func (c *Config) SolanaWSEndpoint() string {
	return c.SolanaWS
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
