// Package pricer — отримання ціни SPL-токенів через ланцюг джерел
package pricer

import (
	"context"
	"sync"
	"time"
)

// TokenPricer — інтерфейс джерела ціни токенів
type TokenPricer interface {
	// Name повертає назву джерела для логування
	Name() string
	// GetTokenPricesUSD повертає map[mint]priceUSD для переданих mint-адрес.
	// Токени з невідомою ціною не потрапляють у результат.
	GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, error)
}

// cacheEntry — один запис TTL-кешу
type cacheEntry struct {
	priceUSD  float64
	source    string
	fetchedAt time.Time
}

// MultiPricer — обходить джерела по черзі для кожного mint.
// Якщо перше джерело знає ціну — використовує її, не питає наступне.
// Результати кешуються з TTL щоб не бити API на кожній транзакції.
type MultiPricer struct {
	sources []TokenPricer
	ttl     time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry // mint → entry
}

// NewMultiPricer створює ланцюг джерел з заданим TTL кешу.
// Порядок sources визначає пріоритет: перший знайдений результат виграє.
func NewMultiPricer(ttl time.Duration, sources ...TokenPricer) *MultiPricer {
	return &MultiPricer{
		sources: sources,
		ttl:     ttl,
		cache:   make(map[string]cacheEntry),
	}
}

// GetTokenPricesUSD повертає ціни для всіх mint.
// Для кожного mint обходить sources по черзі до першого успіху.
// Повертає також map[mint]string з назвою джерела (для логування).
func (m *MultiPricer) GetTokenPricesUSD(ctx context.Context, mints []string) (prices map[string]float64, sources map[string]string) {
	prices = make(map[string]float64, len(mints))
	sources = make(map[string]string, len(mints))

	var stale []string
	now := time.Now()

	// fast path: беремо з кешу
	m.mu.RLock()
	for _, mint := range mints {
		if e, ok := m.cache[mint]; ok && now.Sub(e.fetchedAt) < m.ttl {
			if e.priceUSD > 0 {
				prices[mint] = e.priceUSD
				sources[mint] = e.source
			}
			// priceUSD == 0 → "всі джерела не знають", теж кешуємо
		} else {
			stale = append(stale, mint)
		}
	}
	m.mu.RUnlock()

	if len(stale) == 0 {
		return prices, sources
	}

	// slow path: для кожного stale mint обходимо sources
	fetched := make(map[string]cacheEntry, len(stale))
	remaining := make([]string, len(stale))
	copy(remaining, stale)

	for _, src := range m.sources {
		if len(remaining) == 0 || ctx.Err() != nil {
			break
		}

		got, err := src.GetTokenPricesUSD(ctx, remaining)
		if err != nil {
			// джерело недоступне — пробуємо наступне
			continue
		}

		var stillMissing []string
		for _, mint := range remaining {
			if p, ok := got[mint]; ok && p > 0 {
				fetched[mint] = cacheEntry{
					priceUSD:  p,
					source:    src.Name(),
					fetchedAt: time.Now(),
				}
				prices[mint] = p
				sources[mint] = src.Name()
			} else {
				stillMissing = append(stillMissing, mint)
			}
		}
		remaining = stillMissing
	}

	// токени яких не знає жодне джерело — кешуємо як 0
	for _, mint := range remaining {
		fetched[mint] = cacheEntry{priceUSD: 0, source: "none", fetchedAt: time.Now()}
	}

	// оновлюємо кеш
	m.mu.Lock()
	for mint, e := range fetched {
		m.cache[mint] = e
	}
	m.mu.Unlock()

	return prices, sources
}
