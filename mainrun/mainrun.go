// Package mainrun.
package mainrun

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"go.winto.dev/errors"
	"go.winto.dev/typedcontext"
)

// Run f, this function never return.
//
// ctx passed to f will be canceled when graceful shutdown is requested,
// if f returned error or panic, then log it and run os.Exit(1), otherwise run os.Exit(0).
//
// if err returned by f implement HasExitHandler, that handler will be used.
func Exec(f func(ctx context.Context) error) {
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	var sigCtx osSignal
	ctx, cancel := context.WithCancel(typedcontext.New(context.Background(), &sigCtx))
	defer cancel()

	go func() {
		defer cancel()

		c := make(chan os.Signal, 1)
		signal.Notify(c, getInterruptSigs()...)
		sig := <-c
		signal.Stop(c)

		sigCtx.Lock()
		sigCtx.Signal = sig
		sigCtx.Unlock()
	}()

	err := errors.Catch(func() error { return f(ctx) })
	if err == nil {
		exitCode = 0
		return
	}

	if h := (HasExitHandler)(nil); errors.As(err, &h) {
		errors.Catch(func() error { exitCode = h.ExitHandler()(); return nil })
		return
	}

	fmt.Fprintln(os.Stderr,
		errors.FormatWithFilter(
			err,
			func(l errors.Location) bool { return !l.InPkg("go.winto.dev/mainrun") },
		),
	)
}

type ExitHandler func() (exitCode int)
type HasExitHandler interface{ ExitHandler() ExitHandler }

type osSignal struct {
	sync.Mutex
	os.Signal
}

func (f ExitHandler) Error() string            { return "program executed unsuccessfully" }
func (f ExitHandler) ExitHandler() ExitHandler { return f }

// Return nil if graceful shutdown is not requested yet, otherwise return the signal
func Interrupted(ctx context.Context) os.Signal {
	if s, ok := typedcontext.Get[*osSignal](ctx); ok {
		s.Lock()
		defer s.Unlock()
		return s.Signal
	}
	return nil
}
