package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type HTTPClient struct {
	endpoint string
	client   *http.Client
}

func NewHTTPClient(endpoint string) *HTTPClient {
	return &HTTPClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetTransaction RPC
func (c *HTTPClient) GetTransaction(ctx context.Context, sig string) (*GetTransactionResponse, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []any{
			sig,
			map[string]any{
				"encoding": "jsonParsed",
			},
		},
	}

	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GetTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Result == nil {
		return nil, errors.New("transaction not found")
	}

	return &result, nil
}
