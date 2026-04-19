// Package pricer — Jupiter Price API v2
package pricer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	jupiterPriceURL    = "https://lite-api.jup.ag/price/v2"
	jupiterTimeout     = 5 * time.Second
	maxMintsPerRequest = 100
)

// JupiterPricer реалізує TokenPricer через Jupiter Price API v2.
// Покриття: ~90% ліквідних Solana токенів.
type JupiterPricer struct {
	client *http.Client
}

func NewJupiterPricer() *JupiterPricer {
	return &JupiterPricer{
		client: &http.Client{Timeout: jupiterTimeout},
	}
}

func (j *JupiterPricer) Name() string { return "jupiter" }

func (j *JupiterPricer) GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, error) {
	if len(mints) == 0 {
		return nil, nil
	}

	result := make(map[string]float64, len(mints))

	for i := 0; i < len(mints); i += maxMintsPerRequest {
		end := i + maxMintsPerRequest
		if end > len(mints) {
			end = len(mints)
		}

		batch, err := j.fetchOnce(ctx, mints[i:end])
		if err != nil {
			return result, err
		}

		for mint, price := range batch {
			result[mint] = price
		}
	}

	return result, nil
}

// jupiterPriceItem — один елемент відповіді Jupiter Price API v2.
// Price — float64 (не string, як у старій документації).
type jupiterPriceItem struct {
	ID         string  `json:"id"`
	MintSymbol string  `json:"mintSymbol"`
	Price      float64 `json:"price"`
}

func (j *JupiterPricer) fetchOnce(ctx context.Context, mints []string) (map[string]float64, error) {
	url := fmt.Sprintf("%s?ids=%s", jupiterPriceURL, strings.Join(mints, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jupiter status %d", resp.StatusCode)
	}

	var body struct {
		Data map[string]jupiterPriceItem `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("jupiter decode: %w", err)
	}

	result := make(map[string]float64, len(body.Data))
	for mint, item := range body.Data {
		if item.Price > 0 {
			result[mint] = item.Price
		}
	}

	return result, nil
}
