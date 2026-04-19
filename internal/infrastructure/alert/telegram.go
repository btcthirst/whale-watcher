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

	msg := formatWhaleMessage(e)

	body := map[string]any{
		"chat_id":    t.chatID,
		"text":       msg,
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

// formatWhaleMessage формує текст повідомлення залежно від типу події і джерела ціни
func formatWhaleMessage(e domain.WhaleEvent) string {
	switch e.Type {
	case "SOL":
		return fmt.Sprintf(
			"🐋 *WHALE — SOL*\nAmount: `%.2f SOL`\nUSD: `$%s`\nSig: `%s`",
			e.TotalAmount,
			formatUSD(e.TotalUSD),
			shortSig(e.Signature),
		)

	case "TOKEN":
		switch e.PriceSource {
		case domain.PriceSourceJupiter:
			// відома ціна — показуємо USD і скорочений mint
			return fmt.Sprintf(
				"🐋 *WHALE — TOKEN*\nMint: `%s`\nAmount: `%.2f`\nUSD: `$%s`\nSig: `%s`",
				shortMint(e.Token),
				e.TotalAmount,
				formatUSD(e.TotalUSD),
				shortSig(e.Signature),
			)
		default:
			// PriceSourceFallback — ціна невідома, не показуємо $0
			return fmt.Sprintf(
				"🐋 *WHALE — TOKEN* _(unverified price)_\nMint: `%s`\nAmount: `%.2f`\nSig: `%s`",
				shortMint(e.Token),
				e.TotalAmount,
				shortSig(e.Signature),
			)
		}
	}

	return fmt.Sprintf("🐋 *WHALE*\nType: %s\nAmount: %.2f", e.Type, e.TotalAmount)
}

// formatUSD форматує число в читабельний USD рядок: 1234567 → "1,234,567.00"
func formatUSD(v float64) string {
	if v >= 1_000_000 {
		return fmt.Sprintf("%.2fM", v/1_000_000)
	}
	if v >= 1_000 {
		return fmt.Sprintf("%.2fK", v/1_000)
	}
	return fmt.Sprintf("%.2f", v)
}

// shortMint скорочує mint адресу: перші 4 + "..." + останні 4 символи
func shortMint(mint string) string {
	if len(mint) <= 12 {
		return mint
	}
	return mint[:6] + "..." + mint[len(mint)-6:]
}

// shortSig скорочує signature для читабельності в Telegram
func shortSig(sig string) string {
	if len(sig) <= 16 {
		return sig
	}
	return sig[:8] + "..." + sig[len(sig)-8:]
}
