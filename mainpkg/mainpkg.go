// Package mainpkg.
package mainpkg

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"syscall"

	"go.winto.dev/envparser"
	"go.winto.dev/errors"
	"go.winto.dev/typedcontext"
)

var execCalled atomic.Bool

func Exec(f func(ctx context.Context), envStore any, module string, errFormatFilterPkg ...string) {
	if !execCalled.CompareAndSwap(false, true) {
		panic("mainpkg: Exec only allowed to be called once")
	}

	main := func() {
		errors.SetFormatFilterPkgs(append([]string{"main", module}, errFormatFilterPkg...)...)
		if envStore != nil {
			err := envparser.Unmarshal(envStore)
			if err != nil {
				slog.Error(err.Error())
				Exit(1)
			}
		}
		ctx, cancel := typedcontext.WithCancelSignal(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel(nil)
		f(ctx)
		if err := context.Cause(ctx); err != nil {
			if _, graceful := errors.AsType[typedcontext.CauseBySignal](err); !graceful {
				errors.Check(err)
			}
		}
	}

	if err := errors.Catch0(main); err != nil {
		if code, ok := errors.AsType[ExitCode](err); ok {
			os.Exit(int(code))
		}
		slog.Error(errors.Format(err))
		os.Exit(1)
	}

	os.Exit(0)
}

type ExitCode int

func (e ExitCode) Error() string { return "exit code: " + strconv.Itoa(int(e)) }

func Exit(code int) {
	panic(ExitCode(code))
}
