package async

import (
	"go.winto.dev/errors"
)

// Run0 is similar to normal go keyword, but ignoring panic so that panic will not crash the program.
func Run0(f func()) {
	go func() { errors.Catch0(f) }()
}

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

type Cache[T any] struct {
	C        <-chan T
	HasValue bool
	Value    T
}

func NewCache[T any](ch <-chan T) *Cache[T] {
	return &Cache[T]{C: ch}
}

// Get returns the cached value if it exists, otherwise it waits for the first value from the channel and caches it.
func (c *Cache[T]) Get() T {
	if c.HasValue {
		return c.Value
	}
	c.Value = <-c.C
	c.HasValue = true
	return c.Value
}
