package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	alertapp "github.com/btcthirst/whale-watcher/internal/application/alert"
	"github.com/btcthirst/whale-watcher/internal/application/dedup"
	"github.com/btcthirst/whale-watcher/internal/application/parser"
	"github.com/btcthirst/whale-watcher/internal/application/pricer"
	"github.com/btcthirst/whale-watcher/internal/application/whale"
	"github.com/btcthirst/whale-watcher/internal/config"
	alertinfra "github.com/btcthirst/whale-watcher/internal/infrastructure/alert"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
	"github.com/btcthirst/whale-watcher/internal/storage/sqlite"
)

func main() {
	// logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// graceful shutdown context: cancelled on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "error", err)
		os.Exit(1)
	}

	logger.Info("whale watcher starting",
		"sol_threshold", cfg.SolThreshold,
		"usd_threshold", cfg.USDThreshold,
		"token_amount_threshold", cfg.TokenAmountThreshold,
	)

	// DB
	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		logger.Error("db init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("db close failed", "error", err)
		}
	}()

	if err := sqlite.Migrate(db); err != nil {
		logger.Error("db migrate failed", "error", err)
		os.Exit(1)
	}

	repo := sqlite.NewRepository(db)

	// core services
	wsClient := rpc.NewLogsWSClient(cfg.SolanaWSEndpoint(), logger)
	helius := rpc.NewHeliusClient(cfg.HeliusAPIKey)
	enhancedParser := parser.NewEnhancedParser()
	deduper := dedup.NewDeduper(2 * time.Minute)
	pr := pricer.NewPricer(30 * time.Second)
	detector := whale.NewDetector(pr, cfg.SolThreshold, cfg.USDThreshold, cfg.TokenAmountThreshold)

	// alerts
	var senders []alertapp.Sender

	if cfg.TelegramToken != "" && cfg.TelegramChatID != "" {
		senders = append(senders,
			alertinfra.NewTelegramSender(cfg.TelegramToken, cfg.TelegramChatID),
		)
	}

	if cfg.WebhookURL != "" {
		senders = append(senders,
			alertinfra.NewWebhookSender(cfg.WebhookURL),
		)
	}

	alerts := alertapp.NewService(1000, senders...)
	alerts.Start(ctx, 5)

	// handler: отримує raw WS-повідомлення logsNotification
	handler := func(msg []byte) {
		// parse logsNotification → signature
		notification, err := rpc.ParseLogs(msg)
		if err != nil || notification == nil {
			return // не logsNotification або помилка парсингу
		}

		sig := notification.Params.Result.Value.Signature
		if sig == "" {
			return
		}

		// збагачуємо через Helius Enhanced API
		txs, err := helius.GetTransactions(ctx, []string{sig})
		if err != nil {
			logger.Error("helius fetch failed", "signature", sig, "error", err)
			return
		}

		for _, tx := range txs {
			transfers := enhancedParser.Parse(tx)
			if len(transfers) == 0 {
				continue
			}

			transfers = deduper.DedupTransfers(transfers)

			// storage: transfers
			if err := repo.InsertTransfers(transfers); err != nil {
				logger.Error("insert transfers failed",
					"signature", sig,
					"error", err,
				)
			}

			events, err := detector.Detect(ctx, transfers)
			if err != nil {
				logger.Error("whale detect failed",
					"signature", sig,
					"error", err,
				)
				continue
			}

			if len(events) == 0 {
				continue
			}

			// storage: whale events
			if err := repo.InsertWhaleEvents(events); err != nil {
				logger.Error("insert whale events failed",
					"signature", sig,
					"error", err,
				)
			}

			// alerts
			for _, e := range events {
				alerts.Notify(e)

				logger.Info("🐋 whale event detected",
					"type", e.Type,
					"amount", e.TotalAmount,
					"usd", e.TotalUSD,
					"signature", e.Signature,
				)
			}
		}
	}

	// run WS client (blocks until ctx is cancelled or fatal error)
	if err := wsClient.Run(ctx, handler); err != nil {
		logger.Error("ws client stopped", "error", err)
	}

	logger.Info("shutdown complete")
}
