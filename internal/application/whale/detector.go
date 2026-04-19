// Package whale — детектор китів
package whale

import (
	"context"

	"github.com/btcthirst/whale-watcher/internal/application/pricer"
	"github.com/btcthirst/whale-watcher/internal/domain"
)

type Detector struct {
	pricer *pricer.Pricer

	solThreshold         float64 // мін. кількість SOL для тригера
	usdThreshold         float64 // мін. USD вартість SOL-трансферу для тригера
	tokenAmountThreshold float64 // мін. кількість токен-одиниць для тригера
}

func NewDetector(p *pricer.Pricer, solThreshold, usdThreshold, tokenAmountThreshold float64) *Detector {
	return &Detector{
		pricer:               p,
		solThreshold:         solThreshold,
		usdThreshold:         usdThreshold,
		tokenAmountThreshold: tokenAmountThreshold,
	}
}

// Detect — повертає whale events, розрізняючи SOL і TOKEN пороги
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

	var result []domain.WhaleEvent

	for _, e := range agg {
		switch e.Type {

		case "SOL":
			// USD оцінка на основі поточної ціни SOL
			e.TotalUSD = e.TotalAmount * solPrice

			// тригер: достатньо SOL по кількості АБО по USD вартості
			if e.TotalAmount >= d.solThreshold || e.TotalUSD >= d.usdThreshold {
				result = append(result, *e)
			}

		case "TOKEN":
			// USD оцінка токенів наразі недоступна — TotalUSD = 0
			// тригер: тільки за кількістю токен-одиниць
			if e.TotalAmount >= d.tokenAmountThreshold {
				result = append(result, *e)
			}
		}
	}

	return result, nil
}
