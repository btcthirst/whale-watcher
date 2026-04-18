// Package pricer — отримання ціни SOL
package pricer

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Pricer — з cache
type Pricer struct {
	client *http.Client

	mu sync.RWMutex

	solPrice  float64
	lastFetch time.Time

	ttl time.Duration
}

func NewPricer(ttl time.Duration) *Pricer {
	return &Pricer{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		ttl: ttl,
	}
}

// GetSOLPriceUSD — повертає cached значення або оновлює
func (p *Pricer) GetSOLPriceUSD(ctx context.Context) (float64, error) {
	// fast path (read lock)
	p.mu.RLock()
	if time.Since(p.lastFetch) < p.ttl && p.solPrice > 0 {
		price := p.solPrice
		p.mu.RUnlock()
		return price, nil
	}
	p.mu.RUnlock()

	// slow path (write lock)
	p.mu.Lock()
	defer p.mu.Unlock()

	// double-check (інший goroutine міг вже оновити)
	if time.Since(p.lastFetch) < p.ttl && p.solPrice > 0 {
		return p.solPrice, nil
	}

	price, err := p.fetchSOLPrice(ctx)
	if err != nil {
		// fallback: повертаємо старе значення якщо є
		if p.solPrice > 0 {
			return p.solPrice, nil
		}
		return 0, err
	}

	p.solPrice = price
	p.lastFetch = time.Now()

	return price, nil
}

// fetchSOLPrice — реальний HTTP виклик
func (p *Pricer) fetchSOLPrice(ctx context.Context) (float64, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		"https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd",
		nil,
	)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data struct {
		Solana struct {
			USD float64 `json:"usd"`
		} `json:"solana"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Solana.USD, nil
}
