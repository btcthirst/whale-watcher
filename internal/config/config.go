package config

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
)

type Config struct {
	RPC             RPCConfig        `mapstructure:"rpc"`
	WhaleThresholds WhaleThresholds  `mapstructure:"whale_thresholds"`
	Alerts          AlertsConfig     `mapstructure:"alerts"`
	Storage         StorageConfig    `mapstructure:"storage"`
	Monitoring      MonitoringConfig `mapstructure:"monitoring"`
}

type RPCConfig struct {
	Endpoint   string          `mapstructure:"endpoint"`
	APIKey     string          `mapstructure:"api_key"`
	Commitment string          `mapstructure:"commitment"`
	RateLimit  RateLimitConfig `mapstructure:"rate_limit"`
}

type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	Burst             int `mapstructure:"burst"`
}

type WhaleThresholds struct {
	SOLLamports    uint64   `mapstructure:"sol_lamports"`
	USDValue       float64  `mapstructure:"usd_value"`
	TokenTransfers bool     `mapstructure:"token_transfers"`
	KnownWhales    []string `mapstructure:"known_whales"`
}

type AlertsConfig struct {
	Console     bool          `mapstructure:"console"`
	Webhook     WebhookConfig `mapstructure:"webhook"`
	LogFile     string        `mapstructure:"log_file"`
	LogRotation LogRotation   `mapstructure:"log_rotation"`
}

type WebhookConfig struct {
	Enabled bool              `mapstructure:"enabled"`
	URL     string            `mapstructure:"url"`
	Headers map[string]string `mapstructure:"headers"`
}

type LogRotation struct {
	MaxSizeMB  int  `mapstructure:"max_size_mb"`
	MaxBackups int  `mapstructure:"max_backups"`
	Compress   bool `mapstructure:"compress"`
}

type StorageConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	SQLitePath    string `mapstructure:"sqlite_path"`
	RetentionDays int    `mapstructure:"retention_days"`
}

type MonitoringConfig struct {
	Accounts        []string `mapstructure:"accounts"`
	Programs        []string `mapstructure:"programs"`
	ExcludePrograms []string `mapstructure:"exclude_programs"`
}

// ToDomain converts config to domain types
func (c *WhaleThresholds) ToDomain() (*WhaleDomain, error) {
	whaleMap := make(map[solana.PublicKey]bool)
	for _, addr := range c.KnownWhales {
		pk, err := solana.PublicKeyFromBase58(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid known whale %q: %w", addr, err)
		}
		whaleMap[pk] = true
	}

	excludeMap := make(map[solana.PublicKey]bool)
	for _, addr := range c.ExcludePrograms {
		pk, err := solana.PublicKeyFromBase58(addr)
		if err == nil {
			excludeMap[pk] = true
		}
	}

	return &WhaleDomain{
		SOLThresholdLamports: c.SOLLamports,
		USDThreshold:         c.USDValue,
		TrackTokenTransfers:  c.TokenTransfers,
		KnownWhaleAddresses:  whaleMap,
		ExcludedPrograms:     excludeMap,
	}, nil
}

type WhaleDomain struct {
	SOLThresholdLamports uint64
	USDThreshold         float64
	TrackTokenTransfers  bool
	KnownWhaleAddresses  map[solana.PublicKey]bool
	ExcludedPrograms     map[solana.PublicKey]bool
}
