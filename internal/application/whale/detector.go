// Package whale — детектор китів
package whale

import (
	"context"

	"github.com/btcthirst/whale-watcher/internal/application/pricer"
	"github.com/btcthirst/whale-watcher/internal/domain"
)

type Detector struct {
	pricer *pricer.Pricer

	solThreshold float64
	usdThreshold float64
}

func NewDetector(p *pricer.Pricer, solThreshold, usdThreshold float64) *Detector {
	return &Detector{
		pricer:       p,
		solThreshold: solThreshold,
		usdThreshold: usdThreshold,
	}
}

// Detect — повертає whale events
func (d *Detector) Detect(ctx context.Context, transfers []domain.Transfer) ([]domain.WhaleEvent, error) {
	if len(transfers) == 0 {
		return nil, nil
	}

	solPrice, err := d.pricer.GetSOLPriceUSD(ctx)
	if err != nil {
		return nil, err
	}

	// агрегація по signature + token
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
		// USD оцінка
		if e.Type == "SOL" {
			e.TotalUSD = e.TotalAmount * solPrice
		} else {
			// поки тільки SOL має USD
			e.TotalUSD = 0
		}

		// threshold check
		if e.TotalAmount >= d.solThreshold || e.TotalUSD >= d.usdThreshold {
			result = append(result, *e)
		}
	}

	return result, nil
}
