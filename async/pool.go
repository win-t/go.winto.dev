package async

import (
	"context"
	"runtime"

	"go.winto.dev/errors"
)

type Pool struct {
	group *WaitGroup
	done  context.CancelFunc
	input chan func()
}

// NewPool creates a new goroutine pool with the specified size.
func NewPool(ctx context.Context, size int) *Pool {
	var group WaitGroup
	ctx, done := context.WithCancel(ctx)
	input := make(chan func(), 64*size)
	for range size {
		group.Go(func() {
			for f := range ChanCtx(ctx, input) {
				f()
			}
		})
	}

	pool := &Pool{&group, done, input}
	runtime.AddCleanup(pool, func(struct{}) { done() }, struct{}{})
	return pool
}

// Close closes the pool and waits for all goroutines to finish.
func (p *Pool) Close() {
	p.done()
	p.group.Wait()
}

// Run submits a function to the pool, and return chan to get the value returned by f or the panic value if f panic.
func (p *Pool) Run(f func() error) <-chan error {
	ch := make(chan error, 1)
	p.input <- func() { ch <- errors.Catch(f) }
	return ch
}

// PoolRun2 similar to [Pool.Run] but also returning other value not just error.
func PoolRun2[R any](p *Pool, f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	p.input <- func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	}
	return ch
}
