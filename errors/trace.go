package errors

// TracedErr is an error wrapper that have stack trace
type TracedErr struct {
	Original error
	Trace    []Location
}

func (e *TracedErr) Error() string { return e.Original.Error() }
func (e *TracedErr) Unwrap() error { return e.Original }

func findTracedErr(err error) error {
	for err != nil {
		switch err := err.(type) {
		case *TracedErr:
			return err
		case unwrapslice: // don't deep dive into multi error
			return nil
		}
		err = Unwrap(err)
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
	return &TracedErr{err, getLocs(skip + 1)}
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
	if err, ok := findTracedErr(err).(*TracedErr); ok {
		return err.Trace
	}
	return nil
}
