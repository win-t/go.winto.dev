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
func Catch2[Ret any](f func() (Ret, error)) (Ret, error) {
	var ret Ret
	return ret, Catch(func() error {
		var err error
		ret, err = f()
		return err
	})
}

// like [Catch] but suitable for function doesn't expect to return error
func Catch0(f func()) error {
	return Catch(func() error { f(); return nil })
}
