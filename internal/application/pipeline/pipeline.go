// Package pipeline — простий fan-out/fan-in конвеєр
package pipeline

import (
	"context"
	"sync"
)

// Pipeline — простий fan-out/fan-in конвеєр
type Pipeline[T any] struct {
	In  chan T
	Out chan T

	wg sync.WaitGroup
}

func New[T any](inBuf, outBuf int) *Pipeline[T] {
	return &Pipeline[T]{
		In:  make(chan T, inBuf),
		Out: make(chan T, outBuf),
	}
}

// StartWorkers запускає N воркерів
func (p *Pipeline[T]) StartWorkers(ctx context.Context, n int, fn func(context.Context, T) ([]T, error)) {
	p.wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer p.wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-p.In:
					if !ok {
						return
					}

					results, err := fn(ctx, item)
					if err != nil {
						continue
					}

					for _, r := range results {
						select {
						case p.Out <- r:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}()
	}
}

// Close закриває Out після завершення воркерів
func (p *Pipeline[T]) Close() {
	go func() {
		p.wg.Wait()
		close(p.Out)
	}()
}
