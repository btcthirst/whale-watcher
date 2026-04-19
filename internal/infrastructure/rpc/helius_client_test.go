package rpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeliusClient_GetTransactions(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			fmt.Fprint(w, `[
				{
					"signature": "sig1",
					"slot": 100,
					"nativeTransfers": [{"fromUserAccount": "A", "toUserAccount": "B", "amount": 1000000000}],
					"tokenTransfers": []
				}
			]`)
		}))
		defer ts.Close()

		h := NewHeliusClient("test-key")
		h.client.Transport = &mockTransport{url: ts.URL}

		txs, err := h.GetTransactions(context.Background(), []string{"sig1"})
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, "sig1", txs[0].Signature)
	})

	t.Run("filters out failed transactions", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[
				{
					"signature": "sig-failed",
					"transactionError": {"InstructionError": [0, "Custom"]}
				},
				{
					"signature": "sig-ok",
					"transactionError": null
				}
			]`)
		}))
		defer ts.Close()

		h := NewHeliusClient("test-key")
		h.client.Transport = &mockTransport{url: ts.URL}

		txs, err := h.GetTransactions(context.Background(), []string{"sig-failed", "sig-ok"})
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, "sig-ok", txs[0].Signature)
	})

	t.Run("retry logic on 5xx", func(t *testing.T) {
		attempts := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, `[]`)
		}))
		defer ts.Close()

		h := NewHeliusClient("test-key")
		h.client.Transport = &mockTransport{url: ts.URL}

		txs, err := h.GetTransactions(context.Background(), []string{"sig1"})
		assert.NoError(t, err)
		assert.Empty(t, txs)
		assert.Equal(t, 2, attempts)
	})
}

type mockTransport struct {
	url string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq, _ := http.NewRequest(req.Method, m.url, req.Body)
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}
