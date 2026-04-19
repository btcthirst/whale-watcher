// Package whale — детектор китів
package whale

import (
	"context"

	"github.com/btcthirst/whale-watcher/internal/application/pricer"
	"github.com/btcthirst/whale-watcher/internal/domain"
)

type Detector struct {
	solPricer   *pricer.Pricer
	tokenPricer *pricer.MultiPricer

	solThreshold         float64
	usdThreshold         float64
	tokenAmountThreshold float64 // fallback якщо жодне джерело не знає ціну
}

func NewDetector(
	sp *pricer.Pricer,
	tp *pricer.MultiPricer,
	solThreshold, usdThreshold, tokenAmountThreshold float64,
) *Detector {
	return &Detector{
		solPricer:            sp,
		tokenPricer:          tp,
		solThreshold:         solThreshold,
		usdThreshold:         usdThreshold,
		tokenAmountThreshold: tokenAmountThreshold,
	}
}

// Detect — повертає whale events.
// SOL: порівнює з solThreshold (кількість) або usdThreshold (вартість).
// TOKEN: якщо MultiPricer знає ціну → usdThreshold; інакше → tokenAmountThreshold.
func (d *Detector) Detect(ctx context.Context, transfers []domain.Transfer) ([]domain.WhaleEvent, error) {
	if len(transfers) == 0 {
		return nil, nil
	}

	solPrice, err := d.solPricer.GetSOLPriceUSD(ctx)
	if err != nil {
		return nil, err
	}

	// агрегація по signature + mint + type
	agg := make(map[string]*domain.WhaleEvent)
	for _, t := range transfers {
		key := t.Signature + "|" + t.Mint + "|" + string(t.Type)
		if _, ok := agg[key]; !ok {
			agg[key] = &domain.WhaleEvent{
				Signature: t.Signature,
				Slot:      t.Slot,
				Type:      string(t.Type),
				Token:     t.Mint,
			}
		}
		agg[key].TotalAmount += t.Amount
	}

	// збираємо унікальні mint для TOKEN-подій
	var tokenMints []string
	seen := make(map[string]bool)
	for _, e := range agg {
		if e.Type == "TOKEN" && e.Token != "" && !seen[e.Token] {
			tokenMints = append(tokenMints, e.Token)
			seen[e.Token] = true
		}
	}

	// батч-запит до MultiPricer (Jupiter → DexScreener → GeckoTerminal)
	tokenPrices := make(map[string]float64)
	tokenSources := make(map[string]string)
	if len(tokenMints) > 0 {
		tokenPrices, tokenSources = d.tokenPricer.GetTokenPricesUSD(ctx, tokenMints)
		// помилки не фатальні — деградуємо до fallback
	}

	var result []domain.WhaleEvent

	for _, e := range agg {
		switch e.Type {

		case "SOL":
			e.TotalUSD = e.TotalAmount * solPrice
			e.PriceSource = "coingecko"

			if e.TotalAmount >= d.solThreshold || e.TotalUSD >= d.usdThreshold {
				result = append(result, *e)
			}

		case "TOKEN":
			if price, ok := tokenPrices[e.Token]; ok && price > 0 {
				e.TotalUSD = e.TotalAmount * price
				e.PriceSource = tokenSources[e.Token]

				if e.TotalUSD >= d.usdThreshold {
					result = append(result, *e)
				}
			} else {
				// жодне джерело не знає ціну → fallback за кількістю
				e.PriceSource = "fallback"

				if e.TotalAmount >= d.tokenAmountThreshold {
					result = append(result, *e)
				}
			}
		}
	}

	return result, nil
}
