package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HeliusClient — клієнт Helius Enhanced Transactions API
type HeliusClient struct {
	apiKey string
	client *http.Client
}

// NewHeliusClient створює клієнт з API-ключем Helius
func NewHeliusClient(apiKey string) *HeliusClient {
	return &HeliusClient{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetTransactions — збагачує список сигнатур через Enhanced API.
// Повертає тільки успішні транзакції (TransactionError == nil).
func (h *HeliusClient) GetTransactions(ctx context.Context, signatures []string) ([]HeliusTransaction, error) {
	if len(signatures) == 0 {
		return nil, nil
	}

	url := fmt.Sprintf("https://api.helius.xyz/v0/transactions?api-key=%s", h.apiKey)

	body := map[string]any{"transactions": signatures}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("helius marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("helius request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("helius http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("helius status %d", resp.StatusCode)
	}

	var txs []HeliusTransaction
	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, fmt.Errorf("helius decode: %w", err)
	}

	// відфільтровуємо failed транзакції
	result := txs[:0]
	for _, tx := range txs {
		if tx.TransactionError == nil {
			result = append(result, tx)
		}
	}

	return result, nil
}
