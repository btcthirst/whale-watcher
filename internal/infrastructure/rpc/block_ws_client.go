package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type BlockWSClient struct {
	endpoint string

	conn   *websocket.Conn
	connMu sync.RWMutex

	writeMu sync.Mutex
}

func NewBlockWSClient(endpoint string) *BlockWSClient {
	return &BlockWSClient{endpoint: endpoint}
}

func (c *BlockWSClient) Run(ctx context.Context, handler func([]byte)) error {
	for {
		if err := c.connect(ctx); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if err := c.subscribeBlocks(); err != nil {
			c.close()
			continue
		}

		if err := c.readLoop(ctx, handler); err != nil {
			c.close()
		}
	}
}

func (c *BlockWSClient) connect(ctx context.Context) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}

	fmt.Println("CONNECTING:", u.String())

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		fmt.Println("CONNECT ERROR:", err)
		return err
	}

	fmt.Println("CONNECTED")

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.setupHandlers(conn)

	return nil
}

func (c *BlockWSClient) subscribeBlocks() error {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "blockSubscribe",
		"params": []any{
			map[string]any{},
			map[string]any{
				"encoding":           "jsonParsed",
				"transactionDetails": "full",
			},
		},
	}

	fmt.Println("SUBSCRIBING:", req)

	return c.sendJSON(req)
}

func (c *BlockWSClient) readLoop(ctx context.Context, handler func([]byte)) error {
	for {
		conn := c.getConn()
		if conn == nil {
			return errors.New("no connection")
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("READ ERROR:", err)
			return err
		}

		fmt.Println("RAW MSG:", string(msg))

		handler(msg)
	}
}

func (c *BlockWSClient) sendJSON(v any) error {
	conn := c.getConn()
	if conn == nil {
		return errors.New("no connection")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	fmt.Println("SENDING WS REQUEST")

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

func (c *BlockWSClient) setupHandlers(conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	go c.pingLoop(conn)
}

func (c *BlockWSClient) pingLoop(conn *websocket.Conn) {
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

func (c *BlockWSClient) getConn() *websocket.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

func (c *BlockWSClient) close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}
