package rpc

type GetTransactionResponse struct {
	Result *TransactionResult `json:"result"`
}

type TransactionResult struct {
	Slot uint64 `json:"slot"`

	Meta struct {
		PreBalances  []uint64 `json:"preBalances"`
		PostBalances []uint64 `json:"postBalances"`

		PreTokenBalances  []TokenBalance `json:"preTokenBalances"`
		PostTokenBalances []TokenBalance `json:"postTokenBalances"`

		InnerInstructions []InnerInstruction `json:"innerInstructions"`
	} `json:"meta"`

	Transaction struct {
		Message struct {
			AccountKeys  []AccountKey  `json:"accountKeys"`
			Instructions []Instruction `json:"instructions"`
		} `json:"message"`
	} `json:"transaction"`
}

type AccountKey struct {
	Pubkey string `json:"pubkey"`
}

type Instruction struct {
	Program string  `json:"program"`
	Parsed  *Parsed `json:"parsed"`
}

type InnerInstruction struct {
	Index        int           `json:"index"`
	Instructions []Instruction `json:"instructions"`
}

type Parsed struct {
	Type string `json:"type"`
	Info struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`

		Lamports uint64 `json:"lamports"`

		Amount   string `json:"amount"`
		Mint     string `json:"mint"`
		Decimals uint8  `json:"decimals"`
	} `json:"info"`
}

type TokenBalance struct {
	AccountIndex  int    `json:"accountIndex"`
	Mint          string `json:"mint"`
	UITokenAmount struct {
		Amount   string `json:"amount"`
		Decimals uint8  `json:"decimals"`
	} `json:"uiTokenAmount"`
}
