package dedup

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

// Deduper — основний сервіс дедуплікації
type Deduper struct {
	mu sync.Mutex

	seen map[string]time.Time

	ttl time.Duration
}

// NewDeduper створює dedup engine
func NewDeduper(ttl time.Duration) *Deduper {
	d := &Deduper{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}

	go d.cleanupLoop()

	return d
}

// DedupTransfers — повертає унікальні трансфери
func (d *Deduper) DedupTransfers(transfers []domain.Transfer) []domain.Transfer {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	var result []domain.Transfer

	for _, t := range transfers {
		key := buildKey(t)

		if ts, ok := d.seen[key]; ok {
			// ще в TTL → пропускаємо
			if now.Sub(ts) < d.ttl {
				continue
			}
		}

		d.seen[key] = now
		result = append(result, t)
	}

	return result
}

// cleanupLoop — очищає старі записи
func (d *Deduper) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.cleanup()
	}
}

func (d *Deduper) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	for k, ts := range d.seen {
		if now.Sub(ts) > d.ttl {
			delete(d.seen, k)
		}
	}
}

// buildKey — ключ унікальності трансферу
func buildKey(t domain.Transfer) string {
	extra := ""

	if t.To == "" {
		extra = "fallback"
	}

	return strings.Join([]string{
		t.Signature,
		string(t.Type),
		normalize(t.From),
		normalize(t.To),
		normalize(t.Mint),
		formatAmount(t.Amount),
		extra,
	}, "|")
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// щоб уникнути float noise
func formatAmount(v float64) string {
	return strconv.FormatFloat(v, 'f', 9, 64)
}
