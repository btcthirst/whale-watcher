package rpc

// HeliusTransaction — відповідь Helius Enhanced Transactions API
type HeliusTransaction struct {
	Signature string `json:"signature"`
	Slot      uint64 `json:"slot"`
	Timestamp int64  `json:"timestamp"`

	NativeTransfers []NativeTransfer `json:"nativeTransfers"`
	TokenTransfers  []TokenTransfer  `json:"tokenTransfers"`

	// для фільтрації помилкових транзакцій
	TransactionError any `json:"transactionError"`
}

// NativeTransfer — переміщення SOL (в lamports)
type NativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          uint64 `json:"amount"` // lamports
}

// TokenTransfer — переміщення SPL-токену
type TokenTransfer struct {
	FromUserAccount  string  `json:"fromUserAccount"`
	ToUserAccount    string  `json:"toUserAccount"`
	FromTokenAccount string  `json:"fromTokenAccount"`
	ToTokenAccount   string  `json:"toTokenAccount"`
	TokenAmount      float64 `json:"tokenAmount"`
	Mint             string  `json:"mint"`
}
