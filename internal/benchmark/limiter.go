package benchmark

import (
	"context"
	"errors"
)

// Limiter bounds aggregate model calls across every journey sharing the same
// local inference server.
type Limiter struct {
	slots chan struct{}
}

func NewLimiter(limit int) (*Limiter, error) {
	if limit < 1 {
		return nil, errors.New("completion concurrency limit must be positive")
	}
	return &Limiter{slots: make(chan struct{}, limit)}, nil
}

func (limiter *Limiter) acquire(ctx context.Context) error {
	if limiter == nil {
		return nil
	}
	select {
	case limiter.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (limiter *Limiter) release() {
	if limiter != nil {
		<-limiter.slots
	}
}
