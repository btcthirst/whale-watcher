// Package domain — агрегація трансферів в єдиний івент
package domain

// WhaleEvent — агрегована подія
type WhaleEvent struct {
	Signature string
	Slot      uint64

	Type string // SOL / TOKEN

	TotalAmount float64
	TotalUSD    float64

	Token string
}
