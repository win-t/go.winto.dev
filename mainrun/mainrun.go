// Package mainrun.
package mainrun

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.winto.dev/async"
	"go.winto.dev/errors"
	"go.winto.dev/typedcontext"
)

// Execute f.
//
// this function never return.
//
// ctx passed to f will be canceled when graceful shutdown is requested or f return
// if f panic, it will be printed to stderr with stack trace.returned
//
// if the panic value implement [HasExitDetail], it will be used.
func Exec(f func(ctx context.Context)) {
	var exit ExitDetail
	defer func() {
		if len(exit.Message) > 0 {
			if exit.Message[len(exit.Message)-1] == '\n' {
				fmt.Fprint(os.Stderr, exit.Message)
			} else {
				fmt.Fprintln(os.Stderr, exit.Message)
			}
		}
		os.Exit(exit.Code)
	}()

	var data contextData

	wg := data.getWg()
	defer wg.Wait()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	ctx = typedcontext.New(ctx, &data)

	wg.Add(1)
	go func() {
		defer wg.Done()

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case sig := <-c:
			data.setSignal(sig)
			cancelCtx()
		}
	}()

	err := errors.Catch(func() error { f(ctx); return nil })
	if err == nil {
		return
	}

	if d := (HasExitDetail)(nil); errors.As(err, &d) {
		exit = d.ExitDetail()
	} else {
		exit = ExitDetail{
			Code: 1,
			Message: errors.FormatWithFilter(
				err,
				func(l errors.Location) bool { return !l.InPkg("go.winto.dev/mainrun") },
			),
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

type contextData struct {
	sync.Mutex
	sig os.Signal
	wg  async.WaitGroup
}

func (d *contextData) setSignal(sig os.Signal) {
	d.Lock()
	d.sig = sig
	d.Unlock()
}

func (d *contextData) getSignal() os.Signal {
	if d == nil {
		return nil
	}
	var sig os.Signal
	d.Lock()
	sig = d.sig
	d.Unlock()
	return sig
}

func (d *contextData) getWg() *async.WaitGroup {
	if d == nil {
		return nil
	}
	return &d.wg
}

// Return nil if graceful shutdown is not requested yet, otherwise return the signal
func Interrupted(ctx context.Context) os.Signal {
	s, _ := typedcontext.Get[*contextData](ctx)
	return s.getSignal()
}

// Return WaitGroup that will be waited before Exec terminated
func WaitGroup(ctx context.Context) *async.WaitGroup {
	wg, _ := typedcontext.Get[*contextData](ctx)
	return wg.getWg()
}
