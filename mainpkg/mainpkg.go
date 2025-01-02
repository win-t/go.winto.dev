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

	"go.winto.dev/async"
	"go.winto.dev/errors"
)

var (
	lock sync.Mutex

	called bool
	ctx    context.Context
	sig    os.Signal
	wg     async.WaitGroup

	errLogger func(error)
)

// Execute f, this function call os.Exit() after f returned or panic
//
// if the panic value throw by f is [ExitCode], it will be used as exit code,
// otherwise it will print stack trace and exit with code 1.
//
// Exec cannot be called twice.
func Exec(f func(), tracePkg ...string) {
	lock.Lock()
	if called {
		lock.Unlock()
		panic("cannot call mainpkg.Exec twice")
	}

	ecode := ExitCode(0)
	dontWaitWg := false
	defer func() {
		lock.Unlock()
		if !dontWaitWg {
			wg.Wait()
		}
		os.Exit(int(ecode))
	}()

	var cancelCtx context.CancelFunc
	ctx, cancelCtx = context.WithCancel(context.Background())

	wg.Go(func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case s := <-c:
			lock.Lock()
			sig = s
			lock.Unlock()
			cancelCtx()
		}
	})

	called = true

	lock.Unlock()
	err := errors.Catch0(f)
	cancelCtx()
	lock.Lock()

	if err == nil {
		return
	}

	dontWaitWg = true
	if newecode := ExitCode(0); errors.As(err, &newecode) {
		ecode = newecode
	} else {
		ecode = 1
		var msg string
		if errLogger != nil {
			errLogger(err)
		} else {
			if len(tracePkg) > 0 {
				msg = errors.FormatWithFilterPkgs(err, tracePkg...)
			} else {
				msg = errors.FormatWithFilter(err, func(l errors.Location) bool { return !l.InPkg("go.winto.dev/mainpkg") })
			}
			fmt.Fprintln(os.Stderr, strings.TrimSuffix(msg, "\n"))
		}
	}
}

type ExitCode int

func (e ExitCode) Error() string {
	return fmt.Sprintf("exit (%d)", int(e))
}

// this Context is cancelled when graceful shutdown is requested (SIGTERM or SIGINT)
func Context() context.Context {
	return ctx
}

// Return nil if graceful shutdown is not requested yet, otherwise return the signal
func Interrupted() os.Signal {
	lock.Lock()
	ret := sig
	lock.Unlock()
	return ret
}

// Return WaitGroup that will be waited after f passed to [Exec] return normally
func WaitGroup() *async.WaitGroup {
	return &wg
}

// SetErrLogger set the error logger, default is the error will be printed to os.Stderr
func SetErrLogger(f func(error)) {
	lock.Lock()
	errLogger = f
	lock.Unlock()
}
