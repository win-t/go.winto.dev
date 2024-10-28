package errors

import (
	stderrors "errors"
	"fmt"
	"syscall"
)

// see [stdlib errors.As].
//
// [stdlib errors.As]: https://pkg.go.dev/errors/#As
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// see [stdlib errors.Is].
//
// [stdlib errors.Is]: https://pkg.go.dev/errors/#Is
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// see [stdlib errors.Unwrap].
//
// [stdlib errors.Unwrap]: https://pkg.go.dev/errors/#Unwrap
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// see [stdlib errors.New].
//
// [stdlib errors.New]: https://pkg.go.dev/errors/#New
func New(text string) error {
	return newTracedErr(stderrors.New(text), 1)
}

// see [stdlib fmt.Errorf].
//
// Note: the returned error doesn't implement Unwrap() []error method like [stdlib fmt.Errorf] does, but you can use [Unwrap] it first to get the underlying fmt error.
//
// [stdlib fmt.Errorf]: https://pkg.go.dev/fmt/#Errorf
func Errorf(format string, a ...any) error {
	return newTracedErr(fmt.Errorf(format, a...), 1)
}

// see [stdlib errors.Join].
//
// [stdlib errors.Join]: https://pkg.go.dev/errors/#As
func Join(err ...error) error {
	return stderrors.Join(err...)
}

// will panic if err is not nil.
//
// usage of this function is discouraged.
func Check(err error) {
	if err != nil && err != syscall.Errno(0) {
		panic(traceIfNeeded(err, 1))
	}
}

// Expect will panic with message if fact is false.
//
// usage of this function is discouraged.
func Expect(fact bool, message string) {
	if !fact {
		if message == "" {
			message = "expectation failed"
		}
		panic(newTracedErr(stderrors.New(message), 1))
	}
}

type unwrapslice interface{ Unwrap() []error }

// see [stdlib errors.Unwrap], but for unwraping slices.
//
// [stdlib errors.Unwrap]: https://pkg.go.dev/errors/#Unwrap
func UnwrapSlice(err error) []error {
	if u, ok := err.(unwrapslice); ok {
		return u.Unwrap()
	}
	if err, ok := err.(*TracedErr); ok {
		if u, ok := err.Original.(unwrapslice); ok {
			return u.Unwrap()
		}
	}
	return nil
}

// see [stdlib errors.ErrUnsupported].
//
// [stdlib errors.ErrUnsupported]: https://pkg.go.dev/errors#pkg-variables
var ErrUnsupported = stderrors.ErrUnsupported
