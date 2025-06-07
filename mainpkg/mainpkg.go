// Package mainpkg.
package mainpkg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.winto.dev/async"
	"go.winto.dev/errors"
)

var (
	mu          async.Mutex
	called      bool
	sig         os.Signal
	wg          async.WaitGroup
	tracePkgs   []string
	errLogger   func(error)
	waitOnPanic bool
)

// just a marker type to avoid Opt being called with outside this package
type optParam struct{ a struct{} }

type Opt func(optParam)

func TracePkgs(pkgs ...string) Opt {
	return func(optParam) {
		tracePkgs = pkgs
	}
}

func ErrorLogger(logger func(error)) Opt {
	return func(optParam) {
		errLogger = logger
	}
}

// WaitOnPanic will wait for all goroutines registered to [WaitGroup] to finish
func WaitOnPanic() Opt {
	return func(optParam) {
		waitOnPanic = true
	}
}

// Execute f, this function call os.Exit() after f returned or panic
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
	gotPanic := false
	exitCode := 0

	defer func() {
		if !gotPanic || waitOnPanic {
			wg.Wait()
		}
		os.Exit(exitCode)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

		select {
		case <-ctx.Done():
			signal.Stop(c)
		case s := <-c:
			signal.Stop(c)
			mu.RunFast(func() { sig = s })
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

	gotPanic = true
	if exitCodeErr := ExitCode(0); errors.As(err, &exitCodeErr) {
		exitCode = int(exitCodeErr)
		return
	}

	exitCode = 1
	var msg string
	if errLogger != nil {
		errLogger(err)
	} else {
		if len(tracePkgs) > 0 {
			msg = errors.FormatWithFilterPkgs(err, tracePkgs...)
		} else {
			msg = errors.FormatWithFilter(err, func(l errors.Location) bool { return !l.InPkg("go.winto.dev/mainpkg") })
		}
		fmt.Fprintln(os.Stderr, strings.TrimSuffix(msg, "\n"))
	}
}

type ExitCode int

func (e ExitCode) Error() string {
	return fmt.Sprintf("mainpkg.ExitCode (%d)", int(e))
}

// Return nil if graceful shutdown is not requested yet, otherwise return the signal
func Interrupted() os.Signal {
	var ret os.Signal
	mu.RunFast(func() { ret = sig })
	return ret
}

// Return WaitGroup that will be waited after f passed to [Exec] return normally
func WaitGroup() *async.WaitGroup { return &wg }
