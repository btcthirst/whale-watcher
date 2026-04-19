// Package domain — агрегація трансферів в єдиний івент
package domain

// PriceSource — звідки прийшла USD оцінка події
type PriceSource string

const (
	PriceSourceSOL      PriceSource = "coingecko" // SOL/USD через CoinGecko
	PriceSourceJupiter  PriceSource = "jupiter"   // токен USD через Jupiter
	PriceSourceFallback PriceSource = "fallback"  // ціна невідома, спрацював amount threshold
)

// WhaleEvent — агрегована подія
type WhaleEvent struct {
	Signature string
	Slot      uint64

	Type string // SOL / TOKEN

	TotalAmount float64
	TotalUSD    float64

	Token       string      // mint адреса (для TOKEN)
	PriceSource PriceSource // звідки USD оцінка
}
