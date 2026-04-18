package parser

import (
	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
)

type BlockParser struct{}

func NewBlockParser() *BlockParser {
	return &BlockParser{}
}

func (p *BlockParser) ParseBlock(tx rpc.BlockTransaction, slot uint64) []domain.Transfer {
	signature := ""
	if len(tx.Transaction.Signatures) > 0 {
		signature = tx.Transaction.Signatures[0]
	}

	var transfers []domain.Transfer

	transfers = append(transfers,
		parseInstructions(signature, slot, tx.Transaction.Message.Instructions)...,
	)

	for _, inner := range tx.Meta.InnerInstructions {
		transfers = append(transfers,
			parseInstructions(signature, slot, inner.Instructions)...,
		)
	}

	return transfers
}
