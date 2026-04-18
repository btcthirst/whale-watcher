package parser

import (
	"strconv"

	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
)

// parseInstructions — універсальний парсер (outer + inner)
func parseInstructions(signature string, slot uint64, instructions []rpc.Instruction) []domain.Transfer {
	var result []domain.Transfer

	for _, ix := range instructions {
		if ix.Parsed == nil {
			continue
		}

		info := ix.Parsed.Info

		// SOL transfer
		if ix.Program == "system" && ix.Parsed.Type == "transfer" {
			result = append(result, domain.Transfer{
				Type:      domain.TransferSOL,
				Signature: signature,
				Slot:      slot,
				From:      info.Source,
				To:        info.Destination,
				Amount:    float64(info.Lamports) / 1e9,
			})
		}

		// SPL transfer
		if ix.Program == "spl-token" &&
			(ix.Parsed.Type == "transfer" || ix.Parsed.Type == "transferChecked") {

			amount, _ := strconv.ParseFloat(info.Amount, 64)
			decimals := pow10(int(info.Decimals))

			result = append(result, domain.Transfer{
				Type:      domain.TransferToken,
				Signature: signature,
				Slot:      slot,
				From:      info.Source,
				To:        info.Destination,
				Amount:    amount / decimals,
				Mint:      info.Mint,
				Decimals:  info.Decimals,
			})
		}
	}

	return result
}
