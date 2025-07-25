package async

import (
	"context"
	"iter"
)

// ChanCtx return iterator function that will yield values in the ch or until ctx is done.
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
func ChanCtx[T any](ctx context.Context, ch <-chan T) iter.Seq[T] {
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

// RecvCtx waits for a value from the channel or until the context is done.
func RecvCtx[T any](ctx context.Context, c <-chan T) (value T, ok bool) {
	select {
	case <-ctx.Done():
		return value, false
	case value, ok := <-c:
		return value, ok
	}
}

// SendCtx sends a value to the channel or returns false if the context is done.
func SendCtx[T any](ctx context.Context, c chan<- T, value T) bool {
	select {
	case <-ctx.Done():
		return false
	case c <- value:
		return true
	}
}
