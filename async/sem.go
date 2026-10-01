package async

import (
	"context"
)

type Sem struct{ ch chan struct{} }

func NewSem(size int) Sem {
	return Sem{make(chan struct{}, size)}
}

func (s Sem) Run0(ctx context.Context, f func()) error {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.ch }()
	f()
	return nil
}

// Run runs a function with semaphore limit size.
func (s Sem) Run(ctx context.Context, f func() error) error {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.ch }()
	return f()
}

func (s Sem) Run2[R any](ctx context.Context, f func() (R, error)) (R, error) {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		var ret R
		return ret, ctx.Err()
	}
	defer func() { <-s.ch }()
	return f()
}
