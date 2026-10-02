// Package typedcontext provides utility to inject singleton value into the context.
package typedcontext

import (
	"context"
	"errors"
	"os"
	"os/signal"
)

type key[T any] struct{ _ [0]*T }

// Create new context that have singleton value of the val's type.
func New[T any](ctx context.Context, val T) context.Context {
	return context.WithValue(ctx, key[T]{}, val)
}

// Get the singleton value from the context.
func Get[T any](ctx context.Context) (T, bool) {
	v, ok := ctx.Value(key[T]{}).(T)
	return v, ok
}

// like [Get] but panic if the value is not in the context.
func MustGet[T any](ctx context.Context) T {
	return ctx.Value(key[T]{}).(T)
}

func WithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	ctx = New(ctx, cancel)
	return ctx, cancel
}

func Cancel(ctx context.Context) bool {
	cancel, ok := Get[context.CancelFunc](ctx)
	if ok {
		cancel()
	}
	return ok
}

func WithCancelCause(ctx context.Context) (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(ctx)
	ctx = New(ctx, cancel)
	return ctx, cancel
}

func CancelCause(ctx context.Context, err error) bool {
	cancel, ok := Get[context.CancelCauseFunc](ctx)
	if ok {
		cancel(err)
	}
	return ok
}

type CauseBySignal struct {
	os.Signal
}

func (e CauseBySignal) Error() string {
	return "context canceled by signal " + e.Signal.String()
}

func WithCancelSignal(ctx context.Context, sig os.Signal, sigs ...os.Signal) (context.Context, context.CancelCauseFunc) {
	ctx, cancel := WithCancelCause(ctx)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, append([]os.Signal{sig}, sigs...)...)
	go func() {
		select {
		case s := <-sigCh:
			cancel(CauseBySignal{s})
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()
	return ctx, cancel
}

func CauseSignal(ctx context.Context) os.Signal {
	cause, ok := errors.AsType[CauseBySignal](context.Cause(ctx))
	if !ok {
		return nil
	}
	return cause.Signal
}
