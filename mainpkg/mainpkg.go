// Package mainpkg.
package mainpkg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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

	var eDetail ExitDetail
	var errHappen bool
	defer func() {
		lock.Unlock()
		if !errHappen {
			wg.Wait()
		}
		os.Exit(eDetail.Code)
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
	cancelCtx()
	lock.Lock()
	if err == nil {
		return
	}
	errHappen = true
	eDetail.Code = 1

	if d := (HasExitDetail)(nil); errors.As(err, &d) {
		eDetail = d.ExitDetail()
	}

	if eDetail.Message == "" {
		eDetail.Message = errorFormatter(err)
	}

	if len(eDetail.Message) > 0 {
		if eDetail.Message[len(eDetail.Message)-1] == '\n' {
			fmt.Fprint(os.Stderr, eDetail.Message)
		} else {
			fmt.Fprintln(os.Stderr, eDetail.Message)
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

// Return WaitGroup that will be waited after f passed to [Exec] return normally
func WaitGroup() *async.WaitGroup {
	return &wg
}

var errorFormatter = func(err error) string {
	return errors.FormatWithFilter(
		err,
		func(l errors.Location) bool { return !l.InPkg("go.winto.dev/mainpkg") },
	)
}

func SetErrorFormatter(f func(error) string) {
	lock.Lock()
	errorFormatter = f
	lock.Unlock()
}
