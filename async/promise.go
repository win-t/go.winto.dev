package async

import (
	"context"
	"sync/atomic"
)

type promiseData[T any] struct {
	lockCh chan struct{}
	dataCh <-chan T
}

type Promise[T any] struct {
	innerData atomic.Pointer[promiseData[T]] // nil here means that the promise is already fulfilled
	value     T
}

// Create a promise for the first value that we got from the cannel.
func NewPromise[T any](ch <-chan T) *Promise[T] {
	var p Promise[T]
	p.innerData.Store(&promiseData[T]{
		lockCh: make(chan struct{}, 1),
		dataCh: ch,
	})
	return &p
}

// Create a promise that already fulfilled with a value.
func NewFulfilledPromise[T any](value T) *Promise[T] {
	return &Promise[T]{value: value}
}

// Get the value once from the channel, and return it. Subsequent calls will return the same value without blocking.
func (c *Promise[T]) Get() T {
	v, _ := c.GetCtx(context.Background())
	return v
}

// Similar to [Promise.Get] but with context cancelation support.
func (p *Promise[T]) GetCtx(ctx context.Context) (T, error) {
	// fast path
	state := p.innerData.Load()
	if state == nil {
		return p.value, nil
	}

	return p.slowGet(ctx, state.lockCh)
}

//go:noinline
func (p *Promise[T]) slowGet(ctx context.Context, lockCh chan struct{}) (ret T, err error) {
	// acquire the lock to ensure that only one goroutine can wait for the value at a time
	select {
	case <-ctx.Done():
		return ret, ctx.Err()
	case lockCh <- struct{}{}:
		defer func() { <-lockCh }()
	}

	// check again if the promise is already fulfilled by other goroutine after we acquired the lock
	state := p.innerData.Load()
	if state == nil {
		return p.value, nil
	}

	select {
	case <-ctx.Done():
		return ret, ctx.Err()
	case p.value = <-state.dataCh:
		p.innerData.Store(nil)
		return p.value, nil
	}
}
