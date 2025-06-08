package async

import (
	"context"
	"sync/atomic"
)

type Promise[T any] struct {
	semCh     chan struct{}
	dataCh    <-chan T
	fulfilled atomic.Bool
	value     T
}

// Create a promise for the first value that we got from the cannel.
func NewPromise[T any](ch <-chan T) *Promise[T] {
	return &Promise[T]{
		semCh:  make(chan struct{}, 1),
		dataCh: ch,
	}
}

// Create a promise that already fulfilled with a value.
func NewDonePromise[T any](value T) *Promise[T] {
	p := &Promise[T]{value: value}
	p.fulfilled.Store(true)
	return p
}

// Get the value once from the chanel, and return it. Subsequent calls will return the same value without blocking.
func (c *Promise[T]) Get() T {
	v, _ := c.GetCtx(context.Background())
	return v
}

// Similar to [Promise.Get] but with context cancelation support.
func (p *Promise[T]) GetCtx(ctx context.Context) (T, error) {
	// fast path
	if p.fulfilled.Load() {
		return p.value, nil
	}

	return p.slowGet(ctx)
}

func (p *Promise[T]) lock(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case p.semCh <- struct{}{}:
		return true
	}
}

func (p *Promise[T]) unlock() {
	<-p.semCh
}

//go:noinline
func (p *Promise[T]) slowGet(ctx context.Context) (ret T, err error) {
	if !p.lock(ctx) {
		return ret, ctx.Err()
	}

	if p.fulfilled.Load() {
		p.unlock()
		return p.value, nil
	}

	select {
	case <-ctx.Done():
		p.unlock()
		return ret, ctx.Err()
	case p.value = <-p.dataCh:
		p.fulfilled.Store(true)
		p.unlock()
		return p.value, nil
	}
}
