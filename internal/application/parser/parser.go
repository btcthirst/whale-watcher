// Package parser — реалізація парсера транзакцій
package parser

import (
	"context"
	"strconv"

	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
)

type Parser struct {
	rpc *rpc.HTTPClient
}

func NewParser(rpcClient *rpc.HTTPClient) *Parser {
	return &Parser{rpc: rpcClient}
}

func (p *Parser) ParseTransaction(ctx context.Context, signature string) ([]domain.Transfer, error) {
	tx, err := p.rpc.GetTransaction(ctx, signature)
	if err != nil {
		return nil, err
	}

	var transfers []domain.Transfer

	// 1. outer instructions
	transfers = append(transfers,
		parseInstructions(signature, tx.Result.Slot, tx.Result.Transaction.Message.Instructions)...,
	)

	// 2. inner instructions
	for _, inner := range tx.Result.Meta.InnerInstructions {
		transfers = append(transfers,
			parseInstructions(signature, tx.Result.Slot, inner.Instructions)...,
		)
	}

	// 3. fallback: SOL balance diff
	transfers = append(transfers,
		parseSOLBalanceDiff(signature, tx)...,
	)

	// 4. fallback: token balance diff
	transfers = append(transfers,
		parseTokenBalanceDiff(signature, tx)...,
	)

	return transfers, nil
}

// fallback SOL (balance diff)
func parseSOLBalanceDiff(signature string, tx *rpc.GetTransactionResponse) []domain.Transfer {
	var result []domain.Transfer

	keys := tx.Result.Transaction.Message.AccountKeys

	for i := range tx.Result.Meta.PreBalances {
		pre := tx.Result.Meta.PreBalances[i]
		post := tx.Result.Meta.PostBalances[i]

		if pre > post {
			diff := float64(pre-post) / 1e9

			result = append(result, domain.Transfer{
				Type:      domain.TransferSOL,
				Signature: signature,
				Slot:      tx.Result.Slot,
				From:      keys[i].Pubkey,
				Amount:    diff,
			})
		}
	}

	return result
}

// fallback TOKEN (balance diff)
func parseTokenBalanceDiff(signature string, tx *rpc.GetTransactionResponse) []domain.Transfer {
	var result []domain.Transfer

	preMap := make(map[int]rpc.TokenBalance)
	for _, b := range tx.Result.Meta.PreTokenBalances {
		preMap[b.AccountIndex] = b
	}

	for _, post := range tx.Result.Meta.PostTokenBalances {
		pre, ok := preMap[post.AccountIndex]
		if !ok {
			continue
		}

		preAmount, _ := strconv.ParseFloat(pre.UITokenAmount.Amount, 64)
		postAmount, _ := strconv.ParseFloat(post.UITokenAmount.Amount, 64)

		if postAmount > preAmount {
			diff := (postAmount - preAmount) / pow10(int(post.UITokenAmount.Decimals))

			result = append(result, domain.Transfer{
				Type:      domain.TransferToken,
				Signature: signature,
				Slot:      tx.Result.Slot,
				Mint:      post.Mint,
				Amount:    diff,
				Decimals:  post.UITokenAmount.Decimals,
			})
		}
	}

	return result
}

// helpers

func pow10(n int) float64 {
	result := 1.0
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}
