package errors

import (
	stderrors "errors"
)

type tracedErr struct {
	err  error
	locs []Location
}

func (e *tracedErr) Error() string        { return e.err.Error() }
func (e *tracedErr) Unwrap() error        { return stderrors.Unwrap(e.err) }
func (e *tracedErr) As(target any) bool   { return stderrors.As(e.err, target) }
func (e *tracedErr) Is(target error) bool { return stderrors.Is(e.err, target) }

type tracedSliceErr struct{ tracedErr }

func (e *tracedSliceErr) Unwrap() []error { return e.err.(unwrapslice).Unwrap() }

func findTracedErr(err error) error {
	for err != nil {
		switch err.(type) {
		case *tracedErr, *tracedSliceErr:
			return err
		}
		err = stderrors.Unwrap(err)
	}
	return nil
}

func traceIfNeeded(err error, skip int) error {
	if findTracedErr(err) != nil {
		return err
	}

	return newTracedErr(err, skip+1)
}

func newTracedErr(err error, skip int) error {
	if _, ok := err.(unwrapslice); ok {
		// we only care about trace locs of individual error
		// so skipping locs for error slice
		// keep in sync with [Catch]
		return &tracedSliceErr{tracedErr{err, nil}}
	}

	return &tracedErr{err, getLocs(skip + 1)}
}

// Trace will return new error that have stack trace
//
// will return same err if err already have stack trace
// use [Is] function to compare the returned error with others, because equality (==) operator will fail
func Trace(err error) error {
	if err == nil {
		return nil
	}

	return traceIfNeeded(err, 1)
}

// like [Trace] but suitable for multiple return
func Trace2[A any](a A, err error) (A, error) {
	if err == nil {
		return a, nil
	}

	return a, traceIfNeeded(err, 1)
}

// Get stack trace of err
//
// return nil if err doesn't have stack trace
func StackTrace(err error) []Location {
	switch err := err.(type) {
	case *tracedErr:
		return err.locs
	case *tracedSliceErr:
		return err.tracedErr.locs
	}
	return nil
}
