package domain

import "context"

type WSClient interface {
	Connect(ctx context.Context) error
	SubscribeLogs(ctx context.Context) error
	ReadMessage(ctx context.Context) ([]byte, error)
	Close() error
}
