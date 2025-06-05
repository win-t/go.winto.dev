package async

import (
	"context"
	"runtime"

	"go.winto.dev/errors"
)

type poolBase[I any] struct {
	group *WaitGroup
	done  context.CancelFunc
	input chan I
}

func (p *poolBase[I]) init(ctx context.Context, size int) (context.Context, chan I) {
	p.group = &WaitGroup{}
	ctx, p.done = context.WithCancel(ctx)
	p.input = make(chan I, size)
	return ctx, p.input
}

// Close closes the pool and waits for all goroutines to finish.
func (p *poolBase[I]) Close() error {
	p.done()
	p.group.Wait()
	return nil
}

type (
	Pool struct {
		poolBase[poolInput]
	}
	poolInput struct {
		f     func() error
		errCh chan error
	}
)

// NewPool creates a new goroutine pool with the given size.
func NewPool(ctx context.Context, size int) *Pool {
	pool := &Pool{}
	ctx, input := pool.init(ctx, size)
	for range size {
		pool.group.Go(func() {
			for in := range ChanCtx(ctx, input) {
				err := errors.Catch(in.f)
				if in.errCh != nil {
					in.errCh <- err
				}
			}
		})
	}

	runtime.SetFinalizer(pool, (*Pool).Close) // fallback to close the pool if not closed explicitly
	return pool
}

// Submit submits a function to the pool for execution.
func (p *Pool) Submit(f func() error) <-chan error {
	cErr := make(chan error, 1)
	p.input <- poolInput{f, cErr}
	return cErr
}

// Submit similar to [Submit] but ignores the error returned by the function.
func (p *Pool) Submit0(f func()) {
	p.input <- poolInput{f: func() error { f(); return nil }, errCh: nil}
}

type (
	Pool2[R any] struct {
		poolBase[pool2Input[R]]
	}
	pool2Input[R any] struct {
		f     func() (R, error)
		resCh chan Result[R]
	}
)

// NewPool2 similar to [NewPool] but for functions that return a value and an error.
func NewPool2[R any](ctx context.Context, size int) *Pool2[R] {
	pool := &Pool2[R]{}
	ctx, input := pool.init(ctx, size)
	for range size {
		pool.group.Go(func() {
			for in := range ChanCtx(ctx, input) {
				res, err := errors.Catch2(in.f)
				in.resCh <- Result[R]{res, err}
			}
		})
	}

	runtime.SetFinalizer(pool, (*Pool2[R]).Close) // fallback to close the pool if not closed explicitly
	return pool
}

func (p *Pool2[R]) Submit(f func() (R, error)) <-chan Result[R] {
	cRes := make(chan Result[R], 1)
	p.input <- pool2Input[R]{f, cRes}
	return cRes
}
