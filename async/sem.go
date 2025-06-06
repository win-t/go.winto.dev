package async

type Sem struct {
	ch chan struct{}
}

// NewSem creates a new semaphore with the specified size.
func NewSem(size int) Sem {
	return Sem{make(chan struct{}, size)}
}

// SemRun runs a function with semaphore control, analogous to [Run].
func SemRun(s Sem, f func() error) error {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	return f()
}

// SemRun0 similar to [SemRun], analogous to [Run0].
func SemRun0(s Sem, f func()) {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	f()
}

// SemRun2 similar to [SemRun], analogous to [Run2].
func SemRun2[R any](s Sem, f func() (R, error)) (R, error) {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()
	return f()
}
