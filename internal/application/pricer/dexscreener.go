// Package pricer — DexScreener API
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
	dexscreenerURL     = "https://api.dexscreener.com/latest/dex/tokens"
	dexscreenerTimeout = 5 * time.Second

	// DexScreener приймає до 30 адрес через кому в одному запиті
	dexscreenerBatchSize = 30
)

// DexScreenerPricer реалізує TokenPricer через DexScreener API.
// Покриття: майже всі токени що мають хоч одну торгову пару на DEX.
// Rate limit: 300 req/min, без API ключа.
type DexScreenerPricer struct {
	client *http.Client
}

func NewDexScreenerPricer() *DexScreenerPricer {
	return &DexScreenerPricer{
		client: &http.Client{Timeout: dexscreenerTimeout},
	}
}

func (d *DexScreenerPricer) Name() string { return "dexscreener" }

func (d *DexScreenerPricer) GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, error) {
	if len(mints) == 0 {
		return nil, nil
	}

	result := make(map[string]float64, len(mints))

	for i := 0; i < len(mints); i += dexscreenerBatchSize {
		end := i + dexscreenerBatchSize
		if end > len(mints) {
			end = len(mints)
		}

		batch, err := d.fetchOnce(ctx, mints[i:end])
		if err != nil {
			return result, err
		}

		for mint, price := range batch {
			result[mint] = price
		}
	}

	return result, nil
}

// dexscreenerPair — одна торгова пара з відповіді DexScreener
type dexscreenerPair struct {
	ChainID   string `json:"chainId"`
	BaseToken struct {
		Address string `json:"address"`
	} `json:"baseToken"`
	PriceUsd  string `json:"priceUsd"` // рядок, може бути порожнім
	Liquidity struct {
		USD float64 `json:"usd"`
	} `json:"liquidity"`
}

func (d *DexScreenerPricer) fetchOnce(ctx context.Context, mints []string) (map[string]float64, error) {
	url := fmt.Sprintf("%s/%s", dexscreenerURL, strings.Join(mints, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dexscreener status %d", resp.StatusCode)
	}

	var body struct {
		Pairs []dexscreenerPair `json:"pairs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("dexscreener decode: %w", err)
	}

	// DexScreener повертає список пар для кожного токена.
	// Беремо пару з найбільшою ліквідністю на Solana — вона найточніша.
	// bestPair[mint] = пара з max Liquidity.USD серед solana пар
	bestLiq := make(map[string]float64)
	result := make(map[string]float64, len(mints))

	for _, pair := range body.Pairs {
		if pair.ChainID != "solana" {
			continue
		}
		if pair.PriceUsd == "" {
			continue
		}

		price, err := strconv.ParseFloat(pair.PriceUsd, 64)
		if err != nil || price <= 0 {
			continue
		}

		mint := strings.ToLower(pair.BaseToken.Address)
		if pair.Liquidity.USD > bestLiq[mint] {
			bestLiq[mint] = pair.Liquidity.USD
			result[mint] = price
		}
	}

	// DexScreener повертає адреси в нижньому регістрі,
	// але вхідні mints можуть бути в будь-якому — нормалізуємо назад
	normalized := make(map[string]float64, len(result))
	for _, mint := range mints {
		if price, ok := result[strings.ToLower(mint)]; ok {
			normalized[mint] = price
		}
	}

	return normalized, nil
}
