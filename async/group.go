package async

import (
	"sync"

	"go.winto.dev/errors"
)

type WaitGroup struct{ sync.WaitGroup }

// Run f in new goroutine, and register it into the waitgroup
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

// Run f in new goroutine, and register it into the waitgroup and return a channel to receive the error. analogous to [Run].
func WgRun(wg *WaitGroup, f func() error) <-chan error {
	ch := make(chan error, 1)
	wg.Go(func() { ch <- errors.Catch(f) })
	return ch
}

// WgRun0 similar to [WgRun], analogous to [Run0].
func WgRun0(wg *WaitGroup, f func()) {
	wg.Go(func() { errors.Catch0(f) })
}

// WgRun2 similar to [WgRun], analogous to [Run2].
func WgRun2[R any](wg *WaitGroup, f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	wg.Go(func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	})
	return ch
}
