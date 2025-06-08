package async

import (
	"sync"

	"go.winto.dev/errors"
)

type Sem struct{ ch chan struct{} }

func NewSem(size int) Sem {
	return Sem{make(chan struct{}, size)}
}

// Run runs a function with semaphore control.
func (s Sem) Run(f func()) {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	f()
}

// RunNoPanic is similar to [Sem.Run] but assuming f will not panic.
//
// if f panic, the semapore count will not be restored.
func (s Sem) RunNoPanic(f func()) {
	s.ch <- struct{}{}
	f()
	<-s.ch
}

type Mutex struct{ sync.Mutex }

// Run runs a function with mutex control.
func (m *Mutex) Run(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

// RunNoPanic is similar to [Mutex.Run] but assuming f will not panic.
//
// if f panic, the mutex will not be unlocked.
func (m *Mutex) RunNoPanic(f func()) {
	m.Lock()
	f()
	m.Unlock()
}

type RWMutex struct{ sync.RWMutex }

// Run runs a function with mutex control.
func (m *RWMutex) Run(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

// RunNoPanic is similar to [RWMutex.Run] but assuming f will not panic.
//
// if f panic, the mutex will not be unlocked.
func (m *RWMutex) RunNoPanic(f func()) {
	m.Lock()
	f()
	m.Unlock()
}

// RunRead runs a function with mutex control for read-only data.
func (m *RWMutex) RunRead(f func()) {
	m.RLock()
	defer m.RUnlock()
	f()
}

// RunReadNoPanic is similar to [RWMutex.RunRead] but assuming f will not panic.
//
// if f panic, the mutex will not be unlocked.
func (m *RWMutex) RunReadNoPanic(f func()) {
	m.RLock()
	f()
	m.RUnlock()
}

type WaitGroup struct{ sync.WaitGroup }

// Go is similar to normal go keyword, but it registers to the waitgroup.
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Add(-1)
		f()
	}()
}

// Run0 similar to [WaitGroup.Go], but ignoring panic so that panic will not crash the program.
func (wg *WaitGroup) Run0(f func()) {
	wg.Go(func() { errors.Catch0(f) })
}

// Run f in new goroutine, and register it into the waitgroup, and return chan to get the value returned by f or the panic value if f panic.
func (wg *WaitGroup) Run(f func() error) <-chan error {
	ch := make(chan error, 1)
	wg.Go(func() { ch <- errors.Catch(f) })
	return ch
}

// WaitGroupRun2 similar to [WaitGroup.Run] but also returning other value not just error.
func WaitGroupRun2[R any](wg *WaitGroup, f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	wg.Go(func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	})
	return ch
}
