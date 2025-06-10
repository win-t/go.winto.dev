package async

import (
	"context"
	"sync/atomic"

	"go.winto.dev/errors"
)

// Run the f function in new go routine, and return chan to get the value returned by f or the panic value if f panic.
func Run(f func() error) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- errors.Catch(f) }()
	return ch
}

type Result[R any] struct {
	Result R
	Error  error
}

// Run2 similar with [Run] but also returning other value not just error.
func Run2[R any](f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	go func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	}()
	return ch
}

// Run f in new goroutine, return a function that can be used concurrently to get the result, if f panic, the error value pass to panic will be returned.
func Promise[R any](f func() (R, error)) func(context.Context) (R, error) {
	type wait struct {
		data <-chan Result[R]
		lock chan struct{}
	}

	var data Result[R]
	var waitPtr atomic.Pointer[wait]
	{
		w := &wait{
			data: Run2(f),
			lock: make(chan struct{}, 1),
		}
		w.lock <- struct{}{}
		waitPtr.Store(w)
	}

	return func(ctx context.Context) (ret R, err error) {
		// fast path check
		wait := waitPtr.Load()
		if wait == nil {
			return data.Result, data.Error
		}

		// acquire lock
		select {
		case <-ctx.Done():
			return ret, ctx.Err()
		case <-wait.lock:
		}

		// check again
		wait = waitPtr.Load()
		if wait == nil {
			return data.Result, data.Error
		}

		// retrieve data
		select {
		case <-ctx.Done():
			// this goroutine failed, put back the lock
			wait.lock <- struct{}{}
			return ret, ctx.Err()
		case data = <-wait.data:
			// wake up all waiting goroutine after setting waitPtr to nil to indicate the promise is fulfilled
			waitPtr.Store(nil)
			close(wait.lock)
			return data.Result, data.Error
		}
	}
}
