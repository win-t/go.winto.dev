package errorsmain

import (
	"os"
	"strconv"

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

func ExecErrCallback(f func(), errFn func(error)) {
	if err := errors.Catch0(f); err != nil {
		if code, ok := errors.AsType[ExitCode](err); ok {
			os.Exit(int(code))
		}
		errFn(err)
		os.Exit(1)
	}
	os.Exit(0)
}
