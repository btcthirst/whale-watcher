package parser

import (
	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
)

// EnhancedParser парсить відповідь Helius Enhanced Transactions API.
// nativeTransfers → TransferSOL, tokenTransfers → TransferToken
type EnhancedParser struct{}

func NewEnhancedParser() *EnhancedParser {
	return &EnhancedParser{}
}

// Parse перетворює одну Helius-транзакцію на []Transfer
func (p *EnhancedParser) Parse(tx rpc.HeliusTransaction) []domain.Transfer {
	var result []domain.Transfer

	// SOL transfers (amount in lamports → SOL)
	for _, n := range tx.NativeTransfers {
		if n.Amount == 0 {
			continue
		}
		result = append(result, domain.Transfer{
			Type:      domain.TransferSOL,
			Signature: tx.Signature,
			Slot:      tx.Slot,
			From:      n.FromUserAccount,
			To:        n.ToUserAccount,
			Amount:    float64(n.Amount) / 1e9,
		})
	}

	// SPL token transfers (amount already adjusted for decimals)
	for _, t := range tx.TokenTransfers {
		if t.TokenAmount == 0 {
			continue
		}
		result = append(result, domain.Transfer{
			Type:      domain.TransferToken,
			Signature: tx.Signature,
			Slot:      tx.Slot,
			From:      t.FromUserAccount,
			To:        t.ToUserAccount,
			Amount:    t.TokenAmount,
			Mint:      t.Mint,
		})
	}

	return result
}
