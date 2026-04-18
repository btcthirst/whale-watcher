// Package alert provides alerting logic for whale events.
package alert

import (
	"context"
	"sync"
	"time"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type Sender interface {
	Send(ctx context.Context, e domain.WhaleEvent) error
}

type Service struct {
	queue chan domain.WhaleEvent

	senders []Sender

	wg sync.WaitGroup
}

func NewService(buffer int, senders ...Sender) *Service {
	return &Service{
		queue:   make(chan domain.WhaleEvent, buffer),
		senders: senders,
	}
}

func (s *Service) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		s.wg.Add(1)

		go func() {
			defer s.wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-s.queue:
					s.process(ctx, e)
				}
			}
		}()
	}
}

func (s *Service) process(ctx context.Context, e domain.WhaleEvent) {
	for _, sender := range s.senders {
		retry(ctx, func() error {
			return sender.Send(ctx, e)
		})
	}
}

func (s *Service) Notify(e domain.WhaleEvent) {
	select {
	case s.queue <- e:
	default:
		// drop якщо queue переповнена
	}
}

// simple retry with backoff
func retry(ctx context.Context, fn func() error) {
	backoff := 200 * time.Millisecond

	for i := 0; i < 3; i++ {
		if err := fn(); err == nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff *= 2
		}
	}
}
