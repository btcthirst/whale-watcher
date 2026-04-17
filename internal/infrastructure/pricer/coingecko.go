// Package pricer provides price oracle implementations for the whale watcher.
package pricer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"go.uber.org/zap"
)

type CoinGeckoPricer struct {
	client *http.Client
	logger *zap.Logger
}

func NewCoinGeckoPricer(logger *zap.Logger) *CoinGeckoPricer {
	return &CoinGeckoPricer{
		client: &http.Client{Timeout: 5 * time.Second},
		logger: logger,
	}
}

func (p *CoinGeckoPricer) GetSOLPriceUSD(ctx context.Context) (float64, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		"https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd", nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return data["solana"]["usd"], nil
}

func (p *CoinGeckoPricer) GetTokenPriceUSD(ctx context.Context, mint solana.PublicKey) (float64, error) {
	// У продакшені тут краще використовувати Pyth Network або Jupiter Price API
	// CoinGecko підтримує лише основні токени
	p.logger.Warn("Token price lookup not fully implemented for arbitrary mints")
	return 0, fmt.Errorf("unsupported token")
}
