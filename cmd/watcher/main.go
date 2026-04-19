package main

import (
	"context"
	"errors"
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
	"github.com/btcthirst/whale-watcher/internal/domain"
	alertinfra "github.com/btcthirst/whale-watcher/internal/infrastructure/alert"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
	"github.com/btcthirst/whale-watcher/internal/storage/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	wsClient := rpc.NewLogsWSClient(cfg.SolanaWSEndpoint(), logger)
	helius := rpc.NewHeliusClient(cfg.HeliusAPIKey)
	enhancedParser := parser.NewEnhancedParser()
	deduper := dedup.NewDeduper(2 * time.Minute)

	solPricer := pricer.NewPricer(30 * time.Second)

	// ланцюг джерел ціни токенів: Jupiter → DexScreener → GeckoTerminal
	// перше джерело що знає ціну виграє; результати кешуються 30 секунд
	tokenPricer := pricer.NewMultiPricer(
		30*time.Second,
		pricer.NewJupiterPricer(),
		pricer.NewDexScreenerPricer(),
		pricer.NewGeckoTerminalPricer(),
	)

	detector := whale.NewDetector(solPricer, tokenPricer, cfg.SolThreshold, cfg.USDThreshold, cfg.TokenAmountThreshold)

	var senders []alertapp.Sender
	if cfg.TelegramToken != "" && cfg.TelegramChatID != "" {
		senders = append(senders, alertinfra.NewTelegramSender(cfg.TelegramToken, cfg.TelegramChatID))
	}
	if cfg.WebhookURL != "" {
		senders = append(senders, alertinfra.NewWebhookSender(cfg.WebhookURL))
	}

	alerts := alertapp.NewService(1000, senders...)
	alerts.Start(ctx, 5)

	handler := func(msg []byte) {
		notification, err := rpc.ParseLogs(msg)
		if err != nil || notification == nil {
			return
		}

		sig := notification.Params.Result.Value.Signature
		if sig == "" {
			return
		}

		txs, err := helius.GetTransactions(ctx, []string{sig})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Error("helius fetch failed", "signature", sig, "error", err)
			return
		}

		for _, tx := range txs {
			transfers := enhancedParser.Parse(tx)
			if len(transfers) == 0 {
				continue
			}

			transfers = deduper.DedupTransfers(transfers)

			if err := repo.InsertTransfers(transfers); err != nil {
				logger.Error("insert transfers failed", "signature", sig, "error", err)
			}

			events, err := detector.Detect(ctx, transfers)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				logger.Error("whale detect failed", "signature", sig, "error", err)
				continue
			}

			if len(events) == 0 {
				continue
			}

			if err := repo.InsertWhaleEvents(events); err != nil {
				logger.Error("insert whale events failed", "signature", sig, "error", err)
			}

			for _, e := range events {
				alerts.Notify(e)
				logWhaleEvent(logger, e)
			}
		}
	}

	if err := wsClient.Run(ctx, handler); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("ws client stopped", "error", err)
	}

	logger.Info("shutdown complete")
}

// logWhaleEvent логує подію з урахуванням джерела ціни.
// fallback-події — рівень WARN: токен є, але ціна невідома жодному джерелу.
func logWhaleEvent(logger *slog.Logger, e domain.WhaleEvent) {
	args := []any{
		"type", e.Type,
		"amount", e.TotalAmount,
		"usd", e.TotalUSD,
		"price_source", e.PriceSource,
		"signature", e.Signature,
	}

	if e.Type == "TOKEN" {
		args = append(args, "mint", e.Token)
	}

	if e.PriceSource == "fallback" {
		logger.Warn("🐋 whale event detected (price unknown)", args...)
		return
	}

	logger.Info("🐋 whale event detected", args...)
}
