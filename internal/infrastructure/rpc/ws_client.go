// Package rpc — realisation of RPC client
package rpc

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient — WebSocket клієнт для Solana
type WSClient struct {
	endpoint string

	conn   *websocket.Conn
	connMu sync.RWMutex

	writeMu sync.Mutex
}

// NewWSClient створює клієнт
func NewWSClient(endpoint string) *WSClient {
	return &WSClient{
		endpoint: endpoint,
	}
}

// Run — запускає connect + read loop + reconnect
func (c *WSClient) Run(ctx context.Context, handler func([]byte)) error {
	for {
		if err := c.connect(ctx); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		// підписка після конекту
		if err := c.subscribeLogs(); err != nil {
			c.close()
			continue
		}

		// читаємо повідомлення
		if err := c.readLoop(ctx, handler); err != nil {
			c.close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

// connect встановлює WS з'єднання
func (c *WSClient) connect(ctx context.Context) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return err
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.setupHandlers(conn)

	return nil
}

// subscribeLogs — підписка на всі логи
func (c *WSClient) subscribeLogs() error {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "logsSubscribe",
		"params":  []any{"all"},
	}

	return c.sendJSON(req)
}

// readLoop читає повідомлення
func (c *WSClient) readLoop(ctx context.Context, handler func([]byte)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn := c.getConn()
		if conn == nil {
			return errors.New("no connection")
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		handler(msg)
	}
}

// sendJSON — thread-safe send
func (c *WSClient) sendJSON(v any) error {
	conn := c.getConn()
	if conn == nil {
		return errors.New("no connection")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

// setupHandlers — ping/pong + deadlines
func (c *WSClient) setupHandlers(conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	go c.pingLoop(conn)
}

// pingLoop підтримує connection живим
func (c *WSClient) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.writeMu.Lock()
		err := conn.WriteMessage(websocket.PingMessage, nil)
		c.writeMu.Unlock()

		if err != nil {
			return
		}
	}
}

// helpers

func (c *WSClient) getConn() *websocket.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

func (c *WSClient) close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}
