package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/btcthirst/whale-watcher/internal/application"
	"github.com/btcthirst/whale-watcher/internal/config"
	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/notifier"
	rpcinfra "github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/storage"
	"github.com/btcthirst/whale-watcher/internal/usecase"
)

func main() {
	viper.SetConfigFile("configs/default.yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("config load failed: %w", err))
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("config unmarshal failed: %w", err))
	}

	logger := setupLogger(cfg.Alerts.LogFile, cfg.Alerts.LogRotation)
	defer logger.Sync()

	logger.Info("🐋 Whale-Watcher starting...",
		zap.String("rpc", cfg.RPC.Endpoint),
		zap.Uint64("sol_threshold", cfg.WhaleThresholds.SOLLamports),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Domain config
	whaleCfg, err := cfg.WhaleThresholds.ToDomain()
	if err != nil {
		logger.Fatal("invalid thresholds", zap.Error(err))
	}

	// 2. RPC Clients (використовуємо офіційний rpc.CommitmentType)
	commitment := rpc.CommitmentType(cfg.RPC.Commitment)

	httpClient := rpcinfra.NewHTTPClient(
		cfg.RPC.Endpoint, cfg.RPC.APIKey, commitment,
		cfg.RPC.RateLimit, logger,
	)

	wsClient := rpcinfra.NewWSClient(cfg.RPC.Endpoint, cfg.RPC.APIKey, commitment, logger)
	if err := wsClient.Connect(); err != nil {
		logger.Fatal("ws connect failed", zap.Error(err))
	}

	// 3. Price Oracle
	pricer := &SimplePriceOracle{logger: logger}

	// 4. Alert Channels
	var alertChannels []domain.AlertChannel
	if cfg.Alerts.Console {
		alertChannels = append(alertChannels, notifier.NewConsoleAlert())
	}
	if cfg.Alerts.Webhook.Enabled {
		alertChannels = append(alertChannels, notifier.NewWebhookAlert(
			os.ExpandEnv(cfg.Alerts.Webhook.URL),
			cfg.Alerts.Webhook.Headers,
			logger,
		))
	}

	// 5. Storage
	var storageImpl domain.Storage
	if cfg.Storage.Enabled {
		var err error
		storageImpl, err = storage.NewSQLite(cfg.Storage.SQLitePath, cfg.Storage.RetentionDays)
		if err != nil {
			logger.Warn("storage init failed, running without persistence", zap.Error(err))
		}
	}

	// 6. Alert Service
	alertSvc := application.NewAlertService(alertChannels, storageImpl, 5*time.Minute, logger)
	alertSvc.StartCleanup(ctx)
	defer alertSvc.Close()

	// 7. Business Logic & Monitor
	whaleSvc := application.NewWhaleService(whaleCfg, httpClient, pricer, logger)
	monitor := usecase.NewMonitor(whaleSvc, alertSvc, wsClient, logger)

	if err := monitor.Start(ctx, cfg.Monitoring); err != nil {
		logger.Fatal("monitor start failed", zap.Error(err))
	}

	logger.Info("✅ Whale-Watcher is running. Press Ctrl+C to stop.")

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("🛑 Shutting down...")
	cancel()
	monitor.Shutdown()
	wsClient.Close()
	logger.Info("👋 Stopped")
}

// setupLogger коректно використовує lumberjack через zapcore.Tee
func setupLogger(path string, rot config.LogRotation) *zap.Logger {
	lj := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    rot.MaxSizeMB,
		MaxBackups: rot.MaxBackups,
		Compress:   rot.Compress,
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	level := zap.NewAtomicLevelAt(zap.InfoLevel)

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		level,
	)

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(lj), // ✅ lj тепер використовується
		level,
	)

	return zap.New(zapcore.NewTee(stdoutCore, fileCore))
}

// SimplePriceOracle заглушка для PriceOracle
type SimplePriceOracle struct {
	logger *zap.Logger
}

func (p *SimplePriceOracle) GetSOLPriceUSD(ctx context.Context) (float64, error) {
	// У продакшені тут запит до CoinGecko / Pyth / Jupiter
	p.logger.Debug("fetching mock SOL price")
	return 175.0, nil
}

func (p *SimplePriceOracle) GetTokenPriceUSD(ctx context.Context, mint solana.PublicKey) (float64, error) {
	return 0, nil
}
