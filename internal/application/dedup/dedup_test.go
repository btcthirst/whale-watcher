package dedup

import (
	"testing"
	"time"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

func TestDeduper_DedupTransfers(t *testing.T) {
	d := NewDeduper(50 * time.Millisecond)

	transfers := []domain.Transfer{
		{
			Type:      domain.TransferSOL,
			Signature: "sig1",
			From:      "Alice",
			To:        "Bob",
			Amount:    10.5,
		},
		{
			Type:      domain.TransferToken,
			Signature: "sig1", // Same signature, different type -> different key
			From:      "Alice",
			To:        "Bob",
			Amount:    10.5,
			Mint:      "USDC",
		},
		{
			Type:      domain.TransferSOL,
			Signature: "sig1",
			From:      "Alice",
			To:        "Bob",
			Amount:    10.5, // Duplicate!
		},
		{
			Type:      domain.TransferSOL,
			Signature: "sig1",
			From:      "Alice",
			To:        "Charlie", // Different destination -> different key
			Amount:    10.5,
		},
		{
			Type:      domain.TransferSOL,
			Signature: "sig2",
			From:      "Alice",
			To:        "", // empty to (fallback case)
			Amount:    10.5,
		},
	}

	result := d.DedupTransfers(transfers)

	if len(result) != 4 {
		t.Fatalf("expected 4 transfers, got %d", len(result))
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Submit the exact same transfers again, should all pass because TTL expired
	resultAfterTTL := d.DedupTransfers(transfers)

	if len(resultAfterTTL) != 4 {
		t.Fatalf("expected 4 transfers after TTL, got %d", len(resultAfterTTL))
	}
}

func TestDeduper_Cleanup(t *testing.T) {
	d := NewDeduper(10 * time.Millisecond)

	t1 := domain.Transfer{Signature: "sig1", Type: domain.TransferSOL, From: "A", To: "B", Amount: 1}
	
	// Insert
	d.DedupTransfers([]domain.Transfer{t1})

	d.mu.Lock()
	if len(d.seen) != 1 {
		t.Fatalf("expected 1 item in seen, got %d", len(d.seen))
	}
	d.mu.Unlock()

	// Wait for TTL
	time.Sleep(20 * time.Millisecond)

	// Cleanup
	d.cleanup()

	d.mu.Lock()
	if len(d.seen) != 0 {
		t.Fatalf("expected 0 items in seen after cleanup, got %d", len(d.seen))
	}
	d.mu.Unlock()
}
