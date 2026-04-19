// Package pricer — GeckoTerminal API
package pricer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	geckoTerminalURL     = "https://api.geckoterminal.com/api/v2/simple/networks/solana/token_price"
	geckoTerminalTimeout = 5 * time.Second

	// GeckoTerminal приймає до 30 адрес через кому
	geckoTerminalBatchSize = 30
)

// GeckoTerminalPricer реалізує TokenPricer через GeckoTerminal API.
// Покриття: токени з пулами на будь-якому DEX (Raydium, Orca, Meteora тощо).
// Rate limit: 30 req/min без ключа.
type GeckoTerminalPricer struct {
	client *http.Client
}

func NewGeckoTerminalPricer() *GeckoTerminalPricer {
	return &GeckoTerminalPricer{
		client: &http.Client{Timeout: geckoTerminalTimeout},
	}
}

func (g *GeckoTerminalPricer) Name() string { return "geckoterminal" }

func (g *GeckoTerminalPricer) GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, error) {
	if len(mints) == 0 {
		return nil, nil
	}

	result := make(map[string]float64, len(mints))

	for i := 0; i < len(mints); i += geckoTerminalBatchSize {
		end := i + geckoTerminalBatchSize
		if end > len(mints) {
			end = len(mints)
		}

		batch, err := g.fetchOnce(ctx, mints[i:end])
		if err != nil {
			return result, err
		}

		for mint, price := range batch {
			result[mint] = price
		}
	}

	return result, nil
}

func (g *GeckoTerminalPricer) fetchOnce(ctx context.Context, mints []string) (map[string]float64, error) {
	url := fmt.Sprintf("%s/%s", geckoTerminalURL, strings.Join(mints, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// GeckoTerminal вимагає заголовок Accept
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("geckoterminal status %d", resp.StatusCode)
	}

	// GeckoTerminal response:
	// { "data": { "attributes": { "token_prices": { "<mint>": "1.234" } } } }
	var body struct {
		Data struct {
			Attributes struct {
				TokenPrices map[string]string `json:"token_prices"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("geckoterminal decode: %w", err)
	}

	result := make(map[string]float64, len(body.Data.Attributes.TokenPrices))
	for mint, priceStr := range body.Data.Attributes.TokenPrices {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			continue
		}
		result[mint] = price
	}

	return result, nil
}
