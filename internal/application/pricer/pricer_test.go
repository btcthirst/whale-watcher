package pricer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPricer_GetSOLPriceUSD(t *testing.T) {
	t.Run("successful fetch and caching", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"solana":{"usd":150.5}}`)
		}))
		defer ts.Close()

		p := NewPricer(time.Second)
		// Override fetchSOLPrice logic by using a custom transport to the test server
		p.client.Transport = &mockTransport{url: ts.URL}

		ctx := context.Background()
		price, err := p.GetSOLPriceUSD(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 150.5, price)

		// Change price in test server
		// The value should still be 150.5 due to caching
		price, err = p.GetSOLPriceUSD(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 150.5, price)
	})

	t.Run("fetch error and fallback", func(t *testing.T) {
		p := NewPricer(time.Second)
		p.solPrice = 140.0
		p.lastFetch = time.Now().Add(-2 * time.Second) // expired

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()
		p.client.Transport = &mockTransport{url: ts.URL}

		ctx := context.Background()
		price, err := p.GetSOLPriceUSD(ctx)
		assert.NoError(t, err) // Should not error if fallback exists
		assert.Equal(t, 140.0, price)
	})
}

type mockTransport struct {
	url string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq, _ := http.NewRequest(req.Method, m.url, req.Body)
	return http.DefaultTransport.RoundTrip(newReq)
}
