// Package config application configuration
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	SolanaWS string
	DBPath   string

	HeliusAPIKey string

	TelegramToken  string
	TelegramChatID string

	WebhookURL string

	// Whale detection thresholds
	SolThreshold         float64 // minimum SOL amount to trigger alert  (env: SOL_THRESHOLD)
	USDThreshold         float64 // minimum USD value to trigger SOL alert (env: USD_THRESHOLD)
	TokenAmountThreshold float64 // minimum token units to trigger TOKEN alert (env: TOKEN_AMOUNT_THRESHOLD)
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		SolanaWS: getEnv("SOLANA_WS_URL", ""),
		DBPath:   getEnv("DB_PATH", "whale.db"),

		HeliusAPIKey: getEnv("HELIUS_API_KEY", ""),

		TelegramToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID: getEnv("CHAT_ID", ""),

		WebhookURL: getEnv("WEBHOOK_URL", ""),

		SolThreshold:         getEnvFloat("SOL_THRESHOLD", 100),
		USDThreshold:         getEnvFloat("USD_THRESHOLD", 50000),
		TokenAmountThreshold: getEnvFloat("TOKEN_AMOUNT_THRESHOLD", 50000),
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

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
