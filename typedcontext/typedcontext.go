// Package typedcontext provides utility to inject singleton value into the context.
package typedcontext

import (
	"context"
	"reflect"
)

type key[T any] struct{}

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
	v, ok := ctx.Value(key[T]{}).(T)
	if !ok {
		panicWrongType[T]()
	}
	return v
}

// panicWrongType is separate function to make MustGet got inlined in the caller
func panicWrongType[T any]() {
	panic("context doesn't have the singleton value of type " + reflect.TypeOf((*T)(nil)).Elem().String())
}
