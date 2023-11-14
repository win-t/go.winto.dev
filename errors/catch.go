package errors

import (
	"fmt"
)

// run f, if f panic or returned, that value will be returned by this function
func Catch(f func() error) (err error) {
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}

		recErr, ok := rec.(error)
		if !ok {
			err = newTracedErr(fmt.Errorf("%v", rec), 1)
			return
		}

		if findTracedErr(recErr) == nil {
			err = newTracedErr(recErr, 1)
			return
		}

		err = recErr
		if _, ok := recErr.(*tracedSliceErr); ok {
			// we need to have stack trace here
			err = newTracedErr(fmt.Errorf("%w", recErr), 1)
		}
	}()

	return f()
}

// like [Catch] but suitable for function that return 2 values
func Catch2[A any](f func() (A, error)) (A, error) {
	var a A
	return a, Catch(func() error {
		var err error
		a, err = f()
		return err
	})
}
