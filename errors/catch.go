package errors

import "fmt"

// run f, if f panic or returned, that value will be returned by this function.
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

		if len(StackTrace(recErr)) > 0 {
			err = recErr
			return
		}

		// must have stack trace
		err = Errorf("panic: %w", recErr)
	}()

	return f()
}

// like [Catch] but suitable for function that return 2 values.
func Catch2[A any](f func() (A, error)) (A, error) {
	var a A
	return a, Catch(func() error {
		var err error
		a, err = f()
		return err
	})
}
