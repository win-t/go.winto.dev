// Package typedcontext provides utility to inject singleton value into the context.
package typedcontext

import (
	"context"
	"reflect"
)

type key struct{ t reflect.Type }

// Create new context that have singleton value of the val's type.
func New[T any](ctx context.Context, val T) context.Context {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return context.WithValue(ctx, key{t}, val)
}

// Get the singleton value from the context.
func Get[T any](ctx context.Context) (T, bool) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	v, ok := ctx.Value(key{t}).(T)
	return v, ok
}

// like [Get] but panic if the value is not in the context.
func MustGet[T any](ctx context.Context) T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	v, ok := ctx.Value(key{t}).(T)
	if !ok {
		panic("context doesn't have the singleton value of type " + t.String())
	}
	return v
}
