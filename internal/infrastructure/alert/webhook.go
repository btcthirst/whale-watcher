package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type WebhookSender struct {
	url string
}

func NewWebhookSender(url string) *WebhookSender {
	return &WebhookSender{url: url}
}

func (w *WebhookSender) Send(ctx context.Context, e domain.WhaleEvent) error {
	b, _ := json.Marshal(e)

	req, _ := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return err
	}

	return nil
}
