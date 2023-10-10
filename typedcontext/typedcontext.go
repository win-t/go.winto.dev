// Package typedcontext provides utility to inject singleton value into the context.
package typedcontext

import (
	"context"
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
