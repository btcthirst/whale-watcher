package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	heliusMaxAttempts    = 3
	heliusRetryBaseDelay = 500 * time.Millisecond
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
// При 5xx відповіді виконує до heliusMaxAttempts спроб з exponential backoff.
func (h *HeliusClient) GetTransactions(ctx context.Context, signatures []string) ([]HeliusTransaction, error) {
	if len(signatures) == 0 {
		return nil, nil
	}

	body := map[string]any{"transactions": signatures}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("helius marshal: %w", err)
	}

	url := fmt.Sprintf("https://api.helius.xyz/v0/transactions?api-key=%s", h.apiKey)

	var lastErr error
	delay := heliusRetryBaseDelay

	for attempt := 1; attempt <= heliusMaxAttempts; attempt++ {
		txs, err := h.doRequest(ctx, url, b)
		if err == nil {
			return txs, nil
		}

		lastErr = err

		// не ретраємо: помилки контексту, мережеві помилки, 4xx
		if ctx.Err() != nil {
			return nil, fmt.Errorf("helius http: %w", ctx.Err())
		}
		if !isRetryable(err) {
			return nil, err
		}

		if attempt == heliusMaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("helius http: %w", ctx.Err())
		case <-time.After(delay):
			delay *= 2
		}
	}

	return nil, fmt.Errorf("helius: %d attempts failed, last error: %w", heliusMaxAttempts, lastErr)
}

// doRequest виконує один HTTP-запит і повертає транзакції або помилку.
func (h *HeliusClient) doRequest(ctx context.Context, url string, body []byte) ([]HeliusTransaction, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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
		return nil, &heliusStatusError{code: resp.StatusCode}
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

// heliusStatusError — помилка HTTP-статусу, дозволяє перевіряти код у isRetryable.
type heliusStatusError struct {
	code int
}

func (e *heliusStatusError) Error() string {
	return fmt.Sprintf("helius status %d", e.code)
}

// isRetryable — true для 5xx (сервер тимчасово недоступний).
// 4xx не ретраємо: невалідний запит повторно не допоможе.
func isRetryable(err error) bool {
	var se *heliusStatusError
	if ok := isStatusError(err, &se); ok {
		return se.code >= 500
	}
	return false
}

func isStatusError(err error, target **heliusStatusError) bool {
	e, ok := err.(*heliusStatusError)
	if ok {
		*target = e
	}
	return ok
}
