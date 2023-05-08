package errors

import (
	stderrors "errors"
	"fmt"
)

// see [stdlib errors.As]
//
// [stdlib errors.As]: https://pkg.go.dev/errors/#As
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// see [stdlib errors.Is]
//
// [stdlib errors.Is]: https://pkg.go.dev/errors/#Is
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// see [stdlib errors.Unwrap]
//
// [stdlib errors.Unwrap]: https://pkg.go.dev/errors/#Unwrap
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// see [stdlib errors.New]
//
// [stdlib errors.New]: https://pkg.go.dev/errors/#New
func New(text string) error {
	return newWrappedErr(stderrors.New(text), 1)
}

// see [stdlib fmt.Errorf]
//
// [stdlib fmt.Errorf]: https://pkg.go.dev/fmt/#Errorf
func Errorf(format string, a ...any) error {
	return newWrappedErr(fmt.Errorf(format, a...), 1)
}

// see [stdlib errors.Join]
//
// [stdlib errors.Join]: https://pkg.go.dev/errors/#As
func Join(err ...error) error {
	return newWrappedErr(stderrors.Join(err...), 1)
}

// will panic if err is not nil
func Check(err error) {
	if err != nil {
		panic(traceIfNeeded(err, 1))
	}
}

// Expect will panic with message if fact is false
func Expect(fact bool, message string) {
	if !fact {
		panic(newWrappedErr(stderrors.New(message), 1))
	}
}
