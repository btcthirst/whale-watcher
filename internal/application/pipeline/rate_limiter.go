package pipeline

import "time"

// TokenBucket — простий rate limiter
type TokenBucket struct {
	tokens chan struct{}
}

func NewTokenBucket(rate int, interval time.Duration) *TokenBucket {
	tb := &TokenBucket{
		tokens: make(chan struct{}, rate),
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			for i := 0; i < rate; i++ {
				select {
				case tb.tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	return tb
}

func (t *TokenBucket) Take() {
	<-t.tokens
}
