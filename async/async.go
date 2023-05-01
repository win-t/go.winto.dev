package async

import "go.winto.dev/errors"

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
	ch := make(chan Result[R])
	go func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	}()
	return ch
}
