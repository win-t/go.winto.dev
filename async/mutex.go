package async

import "sync"

type Sem struct{ ch chan struct{} }

// NewSem creates a new semaphore with the specified size.
func NewSem(size int) Sem {
	return Sem{make(chan struct{}, size)}
}

// Run runs a function with semaphore control, analogous to [Run].
func (s Sem) Run(f func() error) error {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	return f()
}

// SemRun0 similar to [Sem.Run], analogous to [Run0].
func (s Sem) Run0(f func()) {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	f()
}

// SemRun2 similar to [Sem.Run], analogous to [Run2].
func SemRun2[R any](s Sem, f func() (R, error)) (R, error) {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	return f()
}

type Mutex struct{ sync.Mutex }

// Run runs a function with mutex control, analogous to [Run].
func (m *Mutex) Run(f func() error) error {
	m.Lock()
	defer m.Unlock()
	return f()
}

// Run0 similar to [Mutex.Run], analogous to [Run0].
func (m *Mutex) Run0(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

// MutexRun2 similar to [Mutex.Run], analogous to [Run2].
func MutexRun2[R any](m *Mutex, f func() (R, error)) (R, error) {
	m.Lock()
	defer m.Unlock()
	return f()
}
