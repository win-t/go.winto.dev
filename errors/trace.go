package errors

type tracedErr struct {
	ori  error
	locs []Location
}

func (e *tracedErr) Error() string { return e.ori.Error() }
func (e *tracedErr) Unwrap() error { return e.ori }

func findTracedErr(err error) *tracedErr {
	for err != nil {
		switch err := err.(type) {
		case *tracedErr:
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
	if traced := findTracedErr(err); traced != nil {
		return traced.locs
	}
	return nil
}
