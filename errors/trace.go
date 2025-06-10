package errors

import "fmt"

type traced[E any] struct {
	locs []Location
	e    E
}

type unwrapslice interface {
	Unwrap() []error
}
type stacktracer interface {
	StackTrace() []Location
}

func (e *traced[E]) StackTrace() []Location { return e.locs }
func (e *traced[E]) Unwrap() E              { return e.e }

func (e *traced[E]) Error() string {
	if err, ok := any(e.e).(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("any error: %v", e.e)
}

//go:noinline
func findTracedErr(err error, digErrSlices bool) stacktracer {
	for err != nil {
		switch v := err.(type) {
		case stacktracer:
			return v
		case unwrapslice: // see comment on traceIfNeeded function
			if !digErrSlices {
				return &traced[error]{nil, err}
			}
			slices := v.Unwrap()
			if len(slices) == 0 {
				return nil
			}
			err = slices[0]
		default:
			err = Unwrap(err)
		}

	}
	return nil
}

func traceIfNeeded(err error, skip int) error {
	// assuming unwrapslice as already traced but without locations
	// as the individual errors in the slice might have locations
	if findTracedErr(err, false) != nil {
		return err
	}

	return &traced[error]{getLocs(skip + 1), err}
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
func Trace2[Ret any](ret Ret, err error) (Ret, error) {
	if err == nil {
		return ret, nil
	}

	return ret, traceIfNeeded(err, 1)
}

// Get stack trace of err
//
// return nil if err doesn't have stack trace
func StackTrace(err error) []Location {
	if traced := findTracedErr(err, true); traced != nil {
		return traced.StackTrace()
	}
	return nil
}
