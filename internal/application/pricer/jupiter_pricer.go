// Package pricer — отримання ціни SPL-токенів через Jupiter Price API v2
package pricer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	jupiterPriceURL = "https://lite-api.jup.ag/price/v2"
	jupiterTimeout  = 5 * time.Second

	// maxMintsPerRequest — Jupiter приймає до 100 mint-адрес в одному запиті
	maxMintsPerRequest = 100
)

// tokenCacheEntry — один запис кешу
type tokenCacheEntry struct {
	priceUSD  float64
	fetchedAt time.Time
}

// JupiterPricer — кеш цін SPL-токенів з TTL
type JupiterPricer struct {
	client *http.Client
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]tokenCacheEntry // mint → entry
}

// NewJupiterPricer створює клієнт з заданим TTL кешу
func NewJupiterPricer(ttl time.Duration) *JupiterPricer {
	return &JupiterPricer{
		client: &http.Client{Timeout: jupiterTimeout},
		ttl:    ttl,
		cache:  make(map[string]tokenCacheEntry),
	}
}

// GetTokenPricesUSD повертає map[mint]priceUSD для переданих mint-адрес.
// Токени з невідомою ціною (не лістяться на Jupiter) не потрапляють у результат.
// Використовує кеш — реальний HTTP робиться тільки для mint з простроченим TTL.
func (j *JupiterPricer) GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, error) {
	if len(mints) == 0 {
		return nil, nil
	}

	result := make(map[string]float64, len(mints))
	var stale []string

	// fast path: беремо з кешу все що ще свіже
	now := time.Now()
	j.mu.RLock()
	for _, mint := range mints {
		if e, ok := j.cache[mint]; ok && now.Sub(e.fetchedAt) < j.ttl {
			if e.priceUSD > 0 {
				result[mint] = e.priceUSD
			}
			// priceUSD == 0 означає "невідомий токен" — теж кешуємо щоб не бити API
		} else {
			stale = append(stale, mint)
		}
	}
	j.mu.RUnlock()

	if len(stale) == 0 {
		return result, nil
	}

	// slow path: запитуємо Jupiter батчами
	fetched, err := j.fetchBatched(ctx, stale)
	if err != nil {
		// при помилці повертаємо те що є з кешу — часткова деградація краща за нуль
		return result, fmt.Errorf("jupiter fetch: %w", err)
	}

	// оновлюємо кеш і результат
	j.mu.Lock()
	fetchedAt := time.Now()
	for _, mint := range stale {
		price := fetched[mint] // 0 якщо не знайдено
		j.cache[mint] = tokenCacheEntry{priceUSD: price, fetchedAt: fetchedAt}
		if price > 0 {
			result[mint] = price
		}
	}
	j.mu.Unlock()

	return result, nil
}

// fetchBatched розбиває mints на батчі і збирає результати
func (j *JupiterPricer) fetchBatched(ctx context.Context, mints []string) (map[string]float64, error) {
	result := make(map[string]float64, len(mints))

	for i := 0; i < len(mints); i += maxMintsPerRequest {
		end := i + maxMintsPerRequest
		if end > len(mints) {
			end = len(mints)
		}
		batch := mints[i:end]

		prices, err := j.fetchOnce(ctx, batch)
		if err != nil {
			return result, err
		}

		for mint, price := range prices {
			result[mint] = price
		}
	}

	return result, nil
}

// jupiterPriceItem — один елемент відповіді Jupiter Price API v2.
// Поле Price — float64 (не string, як у деяких старих версіях документації).
//
// Реальна відповідь API:
//
//	{
//	  "data": {
//	    "<mint>": { "id": "...", "mintSymbol": "USDC", "price": 1.0 }
//	  }
//	}
type jupiterPriceItem struct {
	ID         string  `json:"id"`
	MintSymbol string  `json:"mintSymbol"`
	Price      float64 `json:"price"` // USD ціна за одну одиницю токена
}

// fetchOnce — один HTTP запит до Jupiter Price API v2
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
