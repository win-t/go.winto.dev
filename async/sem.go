package async

import (
	"context"
)

type Sem struct{ ch chan struct{} }

func NewSem(size int) Sem {
	return Sem{make(chan struct{}, size)}
}

// Run runs a function with semaphore control.
func (s Sem) Run(ctx context.Context, f func(ctx context.Context) error) error {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.ch }()
	return f(ctx)
}

func (s Sem) Run2[R any](ctx context.Context, f func(ctx context.Context) (R, error)) (ret R, err error) {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		return ret, ctx.Err()
	}
	defer func() { <-s.ch }()
	return f(ctx)
}

// RunNoPanic is similar to [Sem.Run] but assuming f will not panic.
//
// if f panic, the semaphore count will not be restored.
func (s Sem) RunNoPanic(ctx context.Context, f func(ctx context.Context) error) error {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	err := f(ctx)
	<-s.ch
	return err
}

func (s Sem) Run2NoPanic[R any](ctx context.Context, f func(ctx context.Context) (R, error)) (ret R, err error) {
	select {
	case s.ch <- struct{}{}:
	case <-ctx.Done():
		return ret, ctx.Err()
	}
	ret, err = f(ctx)
	<-s.ch
	return ret, err
}
