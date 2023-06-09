// Package mainpkg.
package mainpkg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
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
)

// Execute f, this function call os.Exit() after f returned or panic
//
// if the panic value throw by f implement [HasExitDetail], it will be used as exit detail.
// otherwise it will print stack trace and exit with code 1.
//
// Exec cannot be called twice.
func Exec(f func()) {
	lock.Lock()
	if called {
		lock.Unlock()
		panic("cannot call mainpkg.Exec twice")
	}

	var eCode int
	defer func() { os.Exit(eCode) }()

	var isRuntimeError bool
	defer func() {
		lock.Unlock()
		if !isRuntimeError {
			wg.Wait()
		}
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
	err := errors.Catch(func() error { f(); return nil })
	lock.Lock()
	cancelCtx()
	if err == nil {
		return
	}
	eCode = 1

	var runtimeError runtime.Error
	isRuntimeError = errors.As(err, &runtimeError)

	var eMsg string
	var hasExitDetail bool

	if d := (HasExitDetail)(nil); errors.As(err, &d) {
		hasExitDetail = errors.Catch(func() error {
			d := d.ExitDetail()
			eCode = d.Code
			eMsg = d.Message
			return nil
		}) == nil
	}

	if !hasExitDetail {
		var hasErrMsg bool
		if errorFormatter != nil {
			hasErrMsg = errors.Catch(func() error { eMsg = errorFormatter(err); return nil }) == nil
		}
		if !hasErrMsg {
			eMsg = errors.FormatWithFilter(
				err,
				func(l errors.Location) bool { return !l.InPkg("go.winto.dev/mainpkg") },
			)
		}
	}

	if len(eMsg) > 0 {
		if eMsg[len(eMsg)-1] == '\n' {
			fmt.Fprint(os.Stderr, eMsg)
		} else {
			fmt.Fprintln(os.Stderr, eMsg)
		}
	}
}

type ExitDetail struct {
	Code    int
	Message string
}

type HasExitDetail interface {
	ExitDetail() ExitDetail
}

func (e ExitDetail) Error() string {
	return fmt.Sprintf("exit (%d): %s", e.Code, e.Message)
}

func (e ExitDetail) ExitDetail() ExitDetail {
	return e
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

// Return WaitGroup that will be waited before Exec terminated
func WaitGroup() *async.WaitGroup {
	return &wg
}

var errorFormatter func(error) string

func SetErrorFormatter(f func(error) string) {
	lock.Lock()
	errorFormatter = f
	lock.Unlock()
}
