package domain

import (
	"time"

	"github.com/gagliardetto/solana-go"
)

// WhaleAlert represents a detected large transaction
type WhaleAlert struct {
	ID             string            `json:"id"`
	Signature      string            `json:"signature"`
	Timestamp      time.Time         `json:"timestamp"`
	From           solana.PublicKey  `json:"from"`
	To             solana.PublicKey  `json:"to"`
	AmountLamports uint64            `json:"amount_lamports"`
	AmountSOL      float64           `json:"amount_sol"`
	USDValue       float64           `json:"usd_value,omitempty"`
	TokenMint      *solana.PublicKey `json:"token_mint,omitempty"`
	TokenSymbol    string            `json:"token_symbol,omitempty"`
	ProgramID      solana.PublicKey  `json:"program_id"`
	Slot           uint64            `json:"slot"`
	Commitment     string            `json:"commitment"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
}
