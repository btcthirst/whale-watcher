package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
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
	ctx := context.Background()

	// logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "error", err)
		os.Exit(1)
	}

	// DB
	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		logger.Error("db init failed", "error", err)
		os.Exit(1)
	}

	if err := sqlite.Migrate(db); err != nil {
		logger.Error("db migrate failed", "error", err)
		os.Exit(1)
	}

	repo := sqlite.NewRepository(db)

	// core services
	wsClient := rpc.NewBlockWSClient(cfg.SolanaWSEndpoint())
	blockParser := parser.NewBlockParser()
	deduper := dedup.NewDeduper(2 * time.Minute)
	pr := pricer.NewPricer(30 * time.Second)
	detector := whale.NewDetector(pr, 100, 50000)

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

	// handler
	handler := func(msg []byte) {
		var block rpc.BlockNotification
		fmt.Println("RAW:", string(msg))

		if err := json.Unmarshal(msg, &block); err != nil {
			logger.Error("json parse error", "error", err)
			return
		}

		slot := block.Params.Result.Context.Slot

		for _, tx := range block.Params.Result.Value.Block.Transactions {

			sig := getSignature(tx)

			transfers := blockParser.ParseBlock(tx, slot)
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

				fmt.Printf("🐋 %s %.2f $%.2f\n",
					e.Type,
					e.TotalAmount,
					e.TotalUSD,
				)
			}
		}
	}

	// run WS client
	if err := wsClient.Run(ctx, handler); err != nil {
		logger.Error("ws client stopped", "error", err)
		os.Exit(1)
	}
}

// helper
func getSignature(tx rpc.BlockTransaction) string {
	if len(tx.Transaction.Signatures) > 0 {
		return tx.Transaction.Signatures[0]
	}
	return "unknown"
}
