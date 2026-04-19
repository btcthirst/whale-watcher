// Package whale — детектор китів
package whale

import (
	"context"

	"github.com/btcthirst/whale-watcher/internal/application/pricer"
	"github.com/btcthirst/whale-watcher/internal/domain"
)

type Detector struct {
	pricer        *pricer.Pricer
	jupiterPricer *pricer.JupiterPricer

	solThreshold         float64 // мін. кількість SOL для тригера
	usdThreshold         float64 // мін. USD вартість для тригера (SOL і TOKEN)
	tokenAmountThreshold float64 // fallback поріг за кількістю токенів якщо ціна невідома
}

func NewDetector(
	p *pricer.Pricer,
	jp *pricer.JupiterPricer,
	solThreshold, usdThreshold, tokenAmountThreshold float64,
) *Detector {
	return &Detector{
		pricer:               p,
		jupiterPricer:        jp,
		solThreshold:         solThreshold,
		usdThreshold:         usdThreshold,
		tokenAmountThreshold: tokenAmountThreshold,
	}
}

// Detect — повертає whale events, розрізняючи SOL і TOKEN пороги.
// Для TOKEN: якщо Jupiter знає ціну → перевіряємо USD поріг (PriceSourceJupiter);
// якщо ціна невідома → fallback на tokenAmountThreshold (PriceSourceFallback).
func (d *Detector) Detect(ctx context.Context, transfers []domain.Transfer) ([]domain.WhaleEvent, error) {
	if len(transfers) == 0 {
		return nil, nil
	}

	solPrice, err := d.pricer.GetSOLPriceUSD(ctx)
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

	// збираємо унікальні mint-адреси TOKEN-подій для батч-запиту до Jupiter
	var tokenMints []string
	seen := make(map[string]bool)
	for _, e := range agg {
		if e.Type == "TOKEN" && e.Token != "" && !seen[e.Token] {
			tokenMints = append(tokenMints, e.Token)
			seen[e.Token] = true
		}
	}

	// отримуємо ціни токенів одним батч-запитом
	tokenPrices := make(map[string]float64)
	if len(tokenMints) > 0 {
		prices, err := d.jupiterPricer.GetTokenPricesUSD(ctx, tokenMints)
		if err == nil {
			tokenPrices = prices
		}
		// при помилці Jupiter — деградуємо до fallback, не фатально
	}

	var result []domain.WhaleEvent

	for _, e := range agg {
		switch e.Type {

		case "SOL":
			e.TotalUSD = e.TotalAmount * solPrice
			e.PriceSource = domain.PriceSourceSOL

			if e.TotalAmount >= d.solThreshold || e.TotalUSD >= d.usdThreshold {
				result = append(result, *e)
			}

		case "TOKEN":
			if tokenPrice, ok := tokenPrices[e.Token]; ok {
				// Jupiter знає ціну → точна USD оцінка
				e.TotalUSD = e.TotalAmount * tokenPrice
				e.PriceSource = domain.PriceSourceJupiter

				if e.TotalUSD >= d.usdThreshold {
					result = append(result, *e)
				}
			} else {
				// невідомий токен → fallback за кількістю одиниць
				e.PriceSource = domain.PriceSourceFallback

				if e.TotalAmount >= d.tokenAmountThreshold {
					result = append(result, *e)
				}
			}
		}
	}

	return result, nil
}
