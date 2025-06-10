// Package mainpkg.
package mainpkg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"go.winto.dev/errors"
)

var (
	mu        sync.RWMutex
	called    bool
	sig       os.Signal
	errLogger func(error)
)

// just a marker type to avoid Opt being called with outside this package
type optParam struct{ _ struct{} }

type Opt func(optParam)

func ErrorLogger(logger func(error)) Opt {
	return func(optParam) {
		errLogger = logger
	}
}

// Execute f with ctx that will be cancelled by SIGINT or SIGTERM, this function call os.Exit() after f returned or panic
//
// if the panic value throw by f is [ExitCode], it will be used as exit code,
// otherwise it will print stack trace and exit with code 1.
//
// Exec cannot be called twice.
func Exec(f func(ctx context.Context), opts ...Opt) {
	mu.Lock()
	if called {
		fmt.Fprintln(os.Stderr, "FATAL: mainpkg.Exec called twice")
		os.Exit(1)
	}

	for _, o := range opts {
		o(optParam{})
	}

	ctx, done := context.WithCancel(context.Background())

	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

		select {
		case <-ctx.Done():
			signal.Stop(c)
		case s := <-c:
			signal.Stop(c)
			mu.Lock()
			sig = s
			mu.Unlock()
			done()
		}
	}()

	called = true
	mu.Unlock()
	err := errors.Catch0(func() { f(ctx) })
	done()
	mu.Lock()

	if err == nil {
		return
	}

	if exitCodeErr := ExitCode(0); errors.As(err, &exitCodeErr) {
		exitCode = int(exitCodeErr)
		return
	}

	exitCode = 1
	if errLogger != nil {
		errLogger(err)
	} else {
		fmt.Fprintln(os.Stderr, strings.TrimSuffix(errors.Format(err), "\n"))
	}
}

type ExitCode int

func (e ExitCode) Error() string {
	return fmt.Sprintf("mainpkg.ExitCode (%d)", int(e))
}

// Return nil if graceful shutdown is not requested yet, otherwise return the signal
//
// possible signals are SIGINT or SIGTERM
func Interrupted() os.Signal {
	mu.RLock()
	ret := sig
	mu.RUnlock()
	return ret
}
