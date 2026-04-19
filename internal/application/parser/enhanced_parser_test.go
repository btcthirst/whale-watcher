package parser

import (
	"reflect"
	"testing"

	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/btcthirst/whale-watcher/internal/infrastructure/rpc"
)

func TestEnhancedParser_Parse(t *testing.T) {
	parser := NewEnhancedParser()

	tx := rpc.HeliusTransaction{
		Signature: "sig123",
		Slot:      12345,
		NativeTransfers: []rpc.NativeTransfer{
			{
				FromUserAccount: "Alice",
				ToUserAccount:   "Bob",
				Amount:          1500000000, // 1.5 SOL
			},
			{
				FromUserAccount: "Charlie",
				ToUserAccount:   "Dave",
				Amount:          0, // Should be ignored
			},
		},
		TokenTransfers: []rpc.TokenTransfer{
			{
				FromUserAccount: "Eve",
				ToUserAccount:   "Frank",
				TokenAmount:     500.5,
				Mint:            "USDC111111111111111111111111111111111111111",
			},
			{
				FromUserAccount: "Grace",
				ToUserAccount:   "Heidi",
				TokenAmount:     0, // Should be ignored
				Mint:            "BONK111111111111111111111111111111111111111",
			},
		},
	}

	expected := []domain.Transfer{
		{
			Type:      domain.TransferSOL,
			Signature: "sig123",
			Slot:      12345,
			From:      "Alice",
			To:        "Bob",
			Amount:    1.5,
		},
		{
			Type:      domain.TransferToken,
			Signature: "sig123",
			Slot:      12345,
			From:      "Eve",
			To:        "Frank",
			Amount:    500.5,
			Mint:      "USDC111111111111111111111111111111111111111",
		},
	}

	result := parser.Parse(tx)

	if len(result) != len(expected) {
		t.Fatalf("expected %d transfers, got %d", len(expected), len(result))
	}

	for i := range expected {
		if !reflect.DeepEqual(result[i], expected[i]) {
			t.Errorf("mismatch at index %d:\nExpected: %+v\nGot:      %+v", i, expected[i], result[i])
		}
	}
}
