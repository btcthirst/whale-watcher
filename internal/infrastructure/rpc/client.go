package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"whale-watcher/internal/domain"
)

type HTTPClient struct {
	client     *rpc.Client
	limiter    *rate.Limiter
	logger     *zap.Logger
	maxRetries int
}

func NewHTTPClient(endpoint, apiKey string, commitment rpc.CommitmentType, rlCfg RateLimitConfig, logger *zap.Logger) *HTTPClient {
	var limiter *rate.Limiter
	if rlCfg.RequestsPerSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(rlCfg.RequestsPerSecond), rlCfg.Burst)
	}

	rpcURL := endpoint
	if apiKey != "" {
		rpcURL = fmt.Sprintf("%s?api-key=%s", endpoint, apiKey)
	}

	return &HTTPClient{
		client:     rpc.New(rpcURL),
		limiter:    limiter,
		logger:     logger,
		maxRetries: 3,
	}
}

func (c *HTTPClient) GetTransaction(ctx context.Context, sig solana.Signature, opts *rpc.GetTransactionOpts) (*rpc.GetTransactionResult, error) {
	return c.executeWithRetry(ctx, func(ctx context.Context) (*rpc.GetTransactionResult, error) {
		return c.client.GetTransaction(ctx, sig, opts)
	})
}

func (c *HTTPClient) GetTokenAccountsByOwner(ctx context.Context, owner solana.PublicKey, opts *rpc.GetTokenAccountsByOwnerOpts) (*rpc.GetTokenAccountsByOwnerResult, error) {
	return c.executeWithRetry(ctx, func(ctx context.Context) (*rpc.GetTokenAccountsByOwnerResult, error) {
		return c.client.GetTokenAccountsByOwner(ctx, owner, opts)
	})
}

func (c *HTTPClient) GetLatestBlockhash(ctx context.Context, commitment rpc.CommitmentType) (*rpc.GetLatestBlockhashResult, error) {
	return c.executeWithRetry(ctx, func(ctx context.Context) (*rpc.GetLatestBlockhashResult, error) {
		return c.client.GetLatestBlockhash(ctx, commitment)
	})
}

func (c *HTTPClient) GetAccountInfo(ctx context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	return c.executeWithRetry(ctx, func(ctx context.Context) (*rpc.GetAccountInfoResult, error) {
		return c.client.GetAccountInfo(ctx, account)
	})
}

func (c *HTTPClient) executeWithRetry[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var zero T

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return zero, fmt.Errorf("rate limiter wait: %w", err)
			}
		}

		res, err := fn(ctx)
		if err == nil {
			return res, nil
		}

		c.logger.Warn("RPC request failed",
			zap.Int("attempt", attempt+1),
			zap.Int("max", c.maxRetries),
			zap.Error(err),
		)

		if attempt < c.maxRetries {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	return zero, fmt.Errorf("max retries exceeded for RPC call")
}