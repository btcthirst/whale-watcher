package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/btcthirst/whale-watcher/internal/domain"

	"go.uber.org/zap"
)

type WebhookAlert struct {
	url     string
	headers map[string]string
	client  *http.Client
	logger  *zap.Logger
}

func NewWebhookAlert(url string, headers map[string]string, logger *zap.Logger) *WebhookAlert {
	return &WebhookAlert{
		url:     url,
		headers: headers,
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  logger,
	}
}

func (w *WebhookAlert) Send(alert *domain.WhaleAlert) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook status: %d", resp.StatusCode)
	}
	return nil
}

func (w *WebhookAlert) Close() error { return nil }
