// Package notifier provides notifier implementations for the whale watcher.
package notifier

import (
	"fmt"
	"time"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type ConsoleAlert struct{}

func NewConsoleAlert() *ConsoleAlert { return &ConsoleAlert{} }

func (c *ConsoleAlert) Send(alert *domain.WhaleAlert) error {
	fmt.Println("\n🐋 WHALE ALERT DETECTED!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📝 Signature:  %s\n", alert.Signature)
	fmt.Printf("⏱ Time:       %s\n", alert.Timestamp.Format(time.RFC3339))
	fmt.Printf("📤 From:       %s\n", alert.From.String())
	fmt.Printf("📥 To:         %s\n", alert.To.String())

	if alert.TokenMint != nil {
		fmt.Printf("🪙 Token:      %.4f %s ($%.2f)\n",
			float64(alert.AmountLamports)/1e9, alert.TokenSymbol, alert.USDValue)
	} else {
		fmt.Printf("💰 Amount:     %.4f SOL ($%.2f)\n", alert.AmountSOL, alert.USDValue)
	}
	fmt.Printf("⚙ Program:    %s\n", alert.ProgramID.String())
	fmt.Printf("📦 Slot:       %d\n", alert.Slot)
	fmt.Printf("🔒 Commitment: %s\n", alert.Commitment)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	return nil
}

func (c *ConsoleAlert) Close() error { return nil }
