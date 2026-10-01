package errorsmain

import (
	"os"
	"strconv"
	"sync/atomic"

	"go.winto.dev/errors"
)

type ExitCode int

func (e ExitCode) Error() string { return "exit code: " + strconv.Itoa(int(e)) }

func Exit(code int) {
	panic(ExitCode(code))
}

func Exec(f func()) {
	ExecErrCallback(f, func(err error) { os.Stderr.WriteString(errors.Format(err)) })
}

var execCalled atomic.Bool

func ExecErrCallback(f func(), errFn func(error)) {
	if !execCalled.CompareAndSwap(false, true) {
		panic("errorsmain: Exec/ExecErrCallback only allowed to be called once")
	}

	if err := errors.Catch0(f); err != nil {
		if code, ok := errors.AsType[ExitCode](err); ok {
			os.Exit(int(code))
		}
		if code, ok := errors.AsType[ExitCode](errors.Catch0(func() { errFn(err) })); ok {
			os.Exit(int(code))
		}
		os.Exit(1)
	}
	os.Exit(0)
}
