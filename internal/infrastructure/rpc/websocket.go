package rpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"go.uber.org/zap"
)

// WSClient manages WebSocket connection and subscriptions
type WSClient struct {
	rpcURL     string
	commitment rpc.CommitmentType
	logger     *zap.Logger
	client     *ws.Client
	subs       []*ws.LogSubscription
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWSClient creates a new WebSocket client
func NewWSClient(endpoint, apiKey string, commitment rpc.CommitmentType, logger *zap.Logger) *WSClient {
	ctx, cancel := context.WithCancel(context.Background())
	url := endpoint
	if apiKey != "" {
		url = fmt.Sprintf("%s?api-key=%s", endpoint, apiKey)
	}

	return &WSClient{
		rpcURL:     url,
		commitment: commitment,
		logger:     logger,
		subs:       make([]*ws.LogSubscription, 0),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Connect establishes the WebSocket connection
func (c *WSClient) Connect() error {
	client, err := ws.Connect(c.ctx, c.rpcURL)
	if err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}
	c.client = client
	c.logger.Info("WebSocket connected")
	return nil
}

// LogHandler callback signature
type LogHandler func(signature string, err error)

// SubscribeLogs subscribes to transaction logs
func (c *WSClient) SubscribeLogs(mentions []solana.PublicKey, handler LogHandler) error {
	if c.client == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	// ✅ FIX 1: Правильний тип фільтру в v1.17.0
	filter := rpc.Filter{
		Mentions: mentions,
	}

	sub, err := c.client.LogsSubscribe(filter, c.commitment)
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}

	c.mu.Lock()
	c.subs = append(c.subs, sub)
	c.mu.Unlock()

	go c.readLogs(sub, handler)
	return nil
}

// readLogs handles incoming log notifications
func (c *WSClient) readLogs(sub rpc.RPCSubscription, handler LogHandler) {
	for {
		// ✅ FIX 2: Recv() вимагає context у v1.17.0
		res, err := sub.Recv(c.ctx)
		if err != nil {
			// Graceful exit on context cancellation
			if c.ctx.Err() != nil {
				return
			}
			handler("", fmt.Errorf("recv error: %w", err))
			continue
		}

		// ✅ FIX 3: res — це struct, не pointer. Перевіряємо res.Err або використовуємо напряму.
		// У solana-go v1.17.0 LogsNotification має поля: Value (struct) та Err
		// Якщо res.Err != nil — це помилка на рівні ноди
		// Якщо res.Value.Signature не порожній — це валідна подія

		// Спроба отримати signature з Value
		// У v1.17.0 структура LogsNotification.Value містить Signature типу solana.Signature
		if res.Value.Signature != (solana.Signature{}) {
			handler(res.Value.Signature.String(), nil)
		}
	}
}

// Close gracefully shuts down the client
func (c *WSClient) Close() error {
	c.cancel()
	c.mu.Lock()
	defer c.mu.Unlock()

	// ✅ FIX 4: У v1.17.0 відписка через client.Unsubscribe(), не sub.Close()
	for _, sub := range c.subs {
		if sub != nil {
			// Ігноруємо помилки при закритті — головне звільнити ресурси
			_ = c.client.Unsubscribe(c.ctx, sub)
		}
	}
	c.subs = nil

	if c.client != nil {
		_ = c.client.Close()
	}
	return nil
}