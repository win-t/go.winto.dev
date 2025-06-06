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
func (wg *WaitGroup) Run(f func() error) <-chan error {
	ch := make(chan error, 1)
	wg.Go(func() { ch <- errors.Catch(f) })
	return ch
}

// Run0 similar to [WaitGroup.Run], analogous to [Run0].
//
// it differs from [WaitGroup.Go] in that it ignore panic
func (wg *WaitGroup) Run0(f func()) {
	wg.Go(func() { errors.Catch0(f) })
}

// WgRun2 similar to [WaitGroup.Run], analogous to [Run2].
func WgRun2[R any](wg *WaitGroup, f func() (R, error)) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	wg.Go(func() {
		r, err := errors.Catch2(f)
		ch <- Result[R]{r, err}
	})
	return ch
}
