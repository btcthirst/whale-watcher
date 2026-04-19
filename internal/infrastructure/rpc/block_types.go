// Package rpc Solana RPC types
package rpc

type BlockNotification struct {
	Method string `json:"method"`
	Params struct {
		Result struct {
			Context struct {
				Slot uint64 `json:"slot"`
			} `json:"context"`
			Value struct {
				Block struct {
					Transactions []BlockTransaction `json:"transactions"`
				} `json:"block"`
			} `json:"value"`
		} `json:"result"`
	} `json:"params"`
}

type BlockTransaction struct {
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys  []AccountKey  `json:"accountKeys"`
			Instructions []Instruction `json:"instructions"`
		} `json:"message"`
	} `json:"transaction"`

	Meta struct {
		PreBalances  []uint64 `json:"preBalances"`
		PostBalances []uint64 `json:"postBalances"`

		InnerInstructions []InnerInstruction `json:"innerInstructions"`
	} `json:"meta"`
}
