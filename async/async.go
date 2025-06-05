package async

import (
	"context"
	"sync"

	"go.winto.dev/errors"
)

// Run the f function in new go routine, and return chan to get the value returned by f
func Run(f func() error) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- errors.Catch(f) }()
	return ch
}

type Result[R any] struct {
	Result R
	Error  error
}

// similar with [Run] but returning some value instead of just error
func Run2[R any](f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	go func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	}()
	return ch
}

type WaitGroup struct{ sync.WaitGroup }

// Run f in new goroutine, and register it into the waitgroup
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

// ChanCtx return iterator function that will yield values in the ch or until ctx is done
//
// similarity can be seen in the following code, but the later can be canceled by the context
//
//	for value := range ch {
//		// ...
//	}
//
//	for value := range async.ChanCtx(ctx, ch) {
//		// ...
//	}
func ChanCtx[T any](ctx context.Context, ch <-chan T) func(func(T) bool) {
	return func(yield func(T) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-ch:
				if !ok {
					return
				}
				if !yield(value) {
					return
				}
			}
		}
	}
}
