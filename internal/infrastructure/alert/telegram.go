// Package alert provides alerting logic for whale events.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type TelegramSender struct {
	token  string
	chatID string
	client *http.Client
}

func NewTelegramSender(token, chatID string) *TelegramSender {
	return &TelegramSender{
		token:  token,
		chatID: chatID,
		client: &http.Client{},
	}
}

func (t *TelegramSender) Send(ctx context.Context, e domain.WhaleEvent) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)

	body := map[string]any{
		"chat_id":    t.chatID,
		"text":       formatWhaleMessage(e),
		"parse_mode": "Markdown",
	}

	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram error: %d", resp.StatusCode)
	}

	return nil
}

func formatWhaleMessage(e domain.WhaleEvent) string {
	switch e.Type {
	case "SOL":
		return fmt.Sprintf(
			"🐋 *WHALE — SOL*\nAmount: `%.4f SOL`\nUSD: `$%s`\nSig: `%s`",
			e.TotalAmount,
			formatUSD(e.TotalUSD),
			shortSig(e.Signature),
		)

	case "TOKEN":
		if e.PriceSource == "fallback" {
			return fmt.Sprintf(
				"🐋 *WHALE — TOKEN* ⚠️ _price unknown_\nMint: `%s`\nAmount: `%s`\nSig: `%s`",
				e.Token,
				formatAmount(e.TotalAmount),
				shortSig(e.Signature),
			)
		}

		return fmt.Sprintf(
			"🐋 *WHALE — TOKEN*\nMint: `%s`\nAmount: `%s`\nUSD: `$%s` _(via %s)_\nSig: `%s`",
			e.Token,
			formatAmount(e.TotalAmount),
			formatUSD(e.TotalUSD),
			e.PriceSource,
			shortSig(e.Signature),
		)
	}

	return fmt.Sprintf("🐋 *WHALE*\nType: %s\nAmount: %.2f", e.Type, e.TotalAmount)
}

// formatUSD: 1_500_000 → "1.50M", 25_000 → "25.00K", 999 → "999.00"
func formatUSD(v float64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("%.2fM", v/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("%.2fK", v/1_000)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

// formatAmount: великі числа скорочуємо для читабельності
func formatAmount(v float64) string {
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", v/1_000_000_000)
	case v >= 1_000_000:
		return fmt.Sprintf("%.2fM", v/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("%.2fK", v/1_000)
	default:
		return fmt.Sprintf("%.4f", v)
	}
}

// shortSig: перші 8 + "…" + останні 8 символів
func shortSig(sig string) string {
	if len(sig) <= 20 {
		return sig
	}
	return sig[:8] + "…" + sig[len(sig)-8:]
}
