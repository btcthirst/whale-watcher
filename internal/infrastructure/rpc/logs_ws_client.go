package rpc

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsReadDeadline  = 60 * time.Second
	wsPingInterval  = 15 * time.Second
	wsWriteDeadline = 10 * time.Second
	wsReconnectWait = 2 * time.Second
)

// LogsWSClient — WebSocket клієнт для logsSubscribe
type LogsWSClient struct {
	endpoint string
	logger   *slog.Logger

	conn   *websocket.Conn
	connMu sync.RWMutex

	writeMu sync.Mutex
}

// NewLogsWSClient створює клієнт підписки на логи
func NewLogsWSClient(endpoint string, logger *slog.Logger) *LogsWSClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogsWSClient{endpoint: endpoint, logger: logger}
}

// Run — нескінченний цикл з авто-реконнектом; зупиняється при ctx.Done()
func (c *LogsWSClient) Run(ctx context.Context, handler func([]byte)) error {
	for {
		select {
		case <-ctx.Done():
			c.close()
			return ctx.Err()
		default:
		}

		if err := c.connect(ctx); err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wsReconnectWait):
			}
			continue
		}

		if err := c.subscribeLogs(); err != nil {
			c.close()
			continue
		}

		if err := c.readLoop(ctx, handler); err != nil {
			c.close()
		}
	}
}

func (c *LogsWSClient) connect(ctx context.Context) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}

	c.logger.Info("connecting to Solana WS", "endpoint", u.String())

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		c.logger.Error("WS connection failed", "error", err)
		return err
	}

	c.logger.Info("WS connection established")

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.setupHandlers(conn)
	return nil
}

// subscribeLogs підписується на всі лог-нотифікації
func (c *LogsWSClient) subscribeLogs() error {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "logsSubscribe",
		"params": []any{
			"all",
			map[string]any{"commitment": "confirmed"},
		},
	}

	c.logger.Info("subscribing to logs notifications")
	return c.sendJSON(req)
}

func (c *LogsWSClient) readLoop(ctx context.Context, handler func([]byte)) error {
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
			c.logger.Error("WS read error", "error", err)
			return err
		}

		// скидаємо deadline після кожного успішно прочитаного повідомлення.
		// без цього deadline спрацьовував через 60s незалежно від активності стріму,
		// бо Helius WS proxy може не відповідати на ping.
		conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

		c.logger.Debug("logs message received", "bytes", len(msg))
		handler(msg)
	}
}

func (c *LogsWSClient) sendJSON(v any) error {
	conn := c.getConn()
	if conn == nil {
		return errors.New("no connection")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))
	return conn.WriteJSON(v)
}

func (c *LogsWSClient) setupHandlers(conn *websocket.Conn) {
	// початковий deadline до першого повідомлення або pong
	conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
		return nil
	})

	go c.pingLoop(conn)
}

func (c *LogsWSClient) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(wsPingInterval)
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

func (c *LogsWSClient) getConn() *websocket.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

func (c *LogsWSClient) close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}
