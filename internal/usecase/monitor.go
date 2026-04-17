package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"go.uber.org/zap"

	"github.com/btcthirst/whale-watcher/internal/application"
	"github.com/btcthirst/whale-watcher/internal/config"
	infraRPC "github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
)

type Monitor struct {
	whaleSvc *application.WhaleService
	alertSvc *application.AlertService
	wsClient *infraRPC.WSClient
	logger   *zap.Logger
	wg       sync.WaitGroup
}

func NewMonitor(whaleSvc *application.WhaleService, alertSvc *application.AlertService, ws *infraRPC.WSClient, logger *zap.Logger) *Monitor {
	return &Monitor{
		whaleSvc: whaleSvc,
		alertSvc: alertSvc,
		wsClient: ws,
		logger:   logger,
	}
}

func (m *Monitor) Start(ctx context.Context, cfg config.MonitoringConfig) error {
	m.logger.Info("initializing monitor...")

	var mentions []solana.PublicKey
	for _, addr := range cfg.Accounts {
		if pk, err := solana.PublicKeyFromBase58(addr); err == nil {
			mentions = append(mentions, pk)
		}
	}

	_, err := m.wsClient.SubscribeLogs(mentions, func(sigStr string, err error) {
		if err != nil {
			m.logger.Error("log stream error", zap.Error(err))
			return
		}

		sig, err := solana.SignatureFromBase58(sigStr)
		if err != nil {
			m.logger.Warn("invalid signature from ws", zap.String("sig", sigStr))
			return
		}

		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.processLog(ctx, sig)
		}()
	})
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}

	m.logger.Info("monitor started, listening for transactions")
	return nil
}

func (m *Monitor) processLog(ctx context.Context, sig solana.Signature) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	alert, err := m.whaleSvc.ProcessTransaction(ctx, sig)
	if err != nil {
		m.logger.Debug("transaction skipped", zap.String("sig", sig.String()), zap.Error(err))
		return
	}
	if alert != nil {
		m.logger.Info("whale detected",
			zap.String("sig", sig.String()),
			zap.Float64("sol", alert.AmountSOL),
			zap.Float64("usd", alert.USDValue),
		)
		m.alertSvc.Process(ctx, alert)
	}
}

func (m *Monitor) Shutdown() {
	m.logger.Info("monitor shutting down, waiting for in-flight tasks...")
	m.wg.Wait()
}
