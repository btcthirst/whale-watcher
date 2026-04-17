package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type AlertService struct {
	logger   *zap.Logger
	channels []domain.AlertChannel
	storage  domain.Storage
	seen     map[string]time.Time
	seenMu   sync.RWMutex
	dedupTTL time.Duration
}

func NewAlertService(channels []domain.AlertChannel, storage domain.Storage, dedupTTL time.Duration, logger *zap.Logger) *AlertService {
	return &AlertService{
		logger:   logger,
		channels: channels,
		storage:  storage,
		seen:     make(map[string]time.Time),
		dedupTTL: dedupTTL,
	}
}

func (s *AlertService) Process(ctx context.Context, alert *domain.WhaleAlert) {
	if alert == nil || alert.Signature == "" {
		return
	}

	s.seenMu.RLock()
	if last, exists := s.seen[alert.Signature]; exists && time.Since(last) < s.dedupTTL {
		s.seenMu.RUnlock()
		s.logger.Debug("duplicate alert skipped", zap.String("sig", alert.Signature))
		return
	}
	s.seenMu.RUnlock()

	for _, ch := range s.channels {
		go func(ch domain.AlertChannel) {
			if err := ch.Send(alert); err != nil {
				s.logger.Error("alert delivery failed",
					zap.String("channel", fmt.Sprintf("%T", ch)),
					zap.Error(err),
				)
			}
		}(ch)
	}

	if s.storage != nil {
		if err := s.storage.Save(alert); err != nil {
			s.logger.Error("storage save failed", zap.Error(err))
		}
	}

	s.seenMu.Lock()
	s.seen[alert.Signature] = time.Now()
	s.seenMu.Unlock()
}

func (s *AlertService) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.dedupTTL / 2)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.purgeOld()
			}
		}
	}()
}

func (s *AlertService) purgeOld() {
	s.seenMu.Lock()
	defer s.seenMu.Unlock()
	now := time.Now()
	for sig, ts := range s.seen {
		if now.Sub(ts) > s.dedupTTL {
			delete(s.seen, sig)
		}
	}
}

func (s *AlertService) Close() error {
	var lastErr error
	for _, ch := range s.channels {
		if err := ch.Close(); err != nil {
			lastErr = err
		}
	}
	if s.storage != nil {
		if err := s.storage.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
