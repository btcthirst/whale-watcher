package domain

// TransferType — тип трансферу
type TransferType string

const (
	TransferSOL   TransferType = "SOL"
	TransferToken TransferType = "TOKEN"
)

// Transfer — уніфікована модель
type Transfer struct {
	Type      TransferType
	Signature string
	Slot      uint64

	From   string
	To     string
	Amount float64

	// token
	Mint     string
	Decimals uint8
}
