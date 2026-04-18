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

	msg := fmt.Sprintf(
		"🐋 *WHALE*\nType: %s\nAmount: %.2f\nUSD: $%.2f",
		e.Type,
		e.TotalAmount,
		e.TotalUSD,
	)

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
