package async

import (
	"context"
	"sync"

	"go.winto.dev/errors"
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

func mutexRun[M sync.Locker](m M, f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

func mutexRun1[M sync.Locker, R any](m M, f func() R) R {
	m.Lock()
	defer m.Unlock()
	return f()
}

func mutexRun2[M sync.Locker, R any](m M, f func() (R, error)) (ret R, err error) {
	m.Lock()
	defer m.Unlock()
	return f()
}

func mutexRunNoPanic[M sync.Locker](m M, f func()) {
	m.Lock()
	f()
	m.Unlock()
}

func mutexRun1NoPanic[M sync.Locker, R any](m M, f func() R) (ret R) {
	m.Lock()
	ret = f()
	m.Unlock()
	return ret
}

func mutexRun2NoPanic[M sync.Locker, R any](m M, f func() (R, error)) (ret R, err error) {
	m.Lock()
	ret, err = f()
	m.Unlock()
	return ret, err
}

type rlockWrapper struct{ inner *sync.RWMutex }

func (m rlockWrapper) Lock()   { m.inner.RLock() }
func (m rlockWrapper) Unlock() { m.inner.RUnlock() }

type Mutex struct{ sync.Mutex }

func (m *Mutex) Run(f func()) { mutexRun(m, f) }

func (m *Mutex) Run1[R any](f func() R) R { return mutexRun1(m, f) }

func (m *Mutex) Run2[R any](f func() (R, error)) (R, error) { return mutexRun2(m, f) }

func (m *Mutex) RunNoPanic(f func()) { mutexRunNoPanic(m, f) }

func (m *Mutex) Run1NoPanic[R any](f func() R) R { return mutexRun1NoPanic(m, f) }

func (m *Mutex) Run2NoPanic[R any](f func() (R, error)) (R, error) { return mutexRun2NoPanic(m, f) }

type RWMutex struct{ sync.RWMutex }

func (m *RWMutex) Run(f func()) { mutexRun(m, f) }

func (m *RWMutex) Run1[R any](f func() R) R { return mutexRun1(m, f) }

func (m *RWMutex) Run2[R any](f func() (R, error)) (R, error) { return mutexRun2(m, f) }

func (m *RWMutex) RunNoPanic(f func()) { mutexRunNoPanic(m, f) }

func (m *RWMutex) Run1NoPanic[R any](f func() R) R { return mutexRun1NoPanic(m, f) }

func (m *RWMutex) Run2NoPanic[R any](f func() (R, error)) (R, error) { return mutexRun2NoPanic(m, f) }

func (m *RWMutex) RunRead(f func()) {
	mutexRun(rlockWrapper{inner: &m.RWMutex}, f)
}

func (m *RWMutex) Run1Read[R any](f func() R) R {
	return mutexRun1(rlockWrapper{inner: &m.RWMutex}, f)
}

func (m *RWMutex) Run2Read[R any](f func() (R, error)) (R, error) {
	return mutexRun2(rlockWrapper{inner: &m.RWMutex}, f)
}

func (m *RWMutex) RunReadNoPanic(f func()) {
	mutexRunNoPanic(rlockWrapper{inner: &m.RWMutex}, f)
}

func (m *RWMutex) Run1ReadNoPanic[R any](f func() R) R {
	return mutexRun1NoPanic(rlockWrapper{inner: &m.RWMutex}, f)
}

func (m *RWMutex) Run2ReadNoPanic[R any](f func() (R, error)) (R, error) {
	return mutexRun2NoPanic(rlockWrapper{inner: &m.RWMutex}, f)
}

type WaitGroup struct{ sync.WaitGroup }

// Run f in new goroutine, and register it into the waitgroup, and return chan to get the value returned by f or the panic value if f panic.
func (wg *WaitGroup) Run(f func() error) <-chan error {
	ch := make(chan error, 1)
	wg.Go(func() { ch <- errors.Catch(f) })
	return ch
}

// WaitGroupRun2 similar to [WaitGroup.Run] but also returning other value not just error.
func (wg *WaitGroup) Run2[R any](f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	wg.Go(func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	})
	return ch
}
