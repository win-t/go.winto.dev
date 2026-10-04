package errors

// this function only allowed to be called as deferred function, it will catch error and store it to err pointer.
//
// if this function is not called as deferred function, it will not work since `recover()` will return nil.
func Recover(err *error) {
	rec := recover()
	if rec == nil {
		return
	}

	recErr, ok := rec.(error)
	if !ok {
		*err = &traced[any]{getLocs(1), rec}
		return
	}

	if len(StackTrace(recErr)) > 0 {
		*err = recErr
		return
	}

	// error from recovered panic must have stack trace
	*err = Errorf("panic: %w", recErr)
}

// like [Catch] but suitable for function doesn't expect to return error
func Catch0(f func()) (err error) {
	defer Recover(&err)
	f()
	return nil
}

// run f, if f panic or returned, that value will be returned by this function.
func Catch(f func() error) (err error) {
	defer Recover(&err)
	return f()
}

// like [Catch] but suitable for function expect to return single value
func Catch1[R any](f func() R) (ret R, err error) {
	defer Recover(&err)
	return f(), err
}

func Catch1Args1[R any, A1 any](f func(A1) R, a1 A1) (ret R, err error) {
	defer Recover(&err)
	return f(a1), err
}

func Catch1Args2[R any, A1 any, A2 any](f func(A1, A2) R, a1 A1, a2 A2) (ret R, err error) {
	defer Recover(&err)
	return f(a1, a2), err
}

func Catch1Args3[R any, A1 any, A2 any, A3 any](f func(A1, A2, A3) R, a1 A1, a2 A2, a3 A3) (ret R, err error) {
	defer Recover(&err)
	return f(a1, a2, a3), err
}

func Catch1Args4[R any, A1 any, A2 any, A3 any, A4 any](f func(A1, A2, A3, A4) R, a1 A1, a2 A2, a3 A3, a4 A4) (ret R, err error) {
	defer Recover(&err)
	return f(a1, a2, a3, a4), err
}

// like [Catch] but suitable for function that return value and error.
func Catch2[R any](f func() (R, error)) (ret R, err error) {
	defer Recover(&err)
	return f()
}

func Catch2Args1[R any, A1 any](f func(A1) (R, error), a1 A1) (ret R, err error) {
	defer Recover(&err)
	return f(a1)
}

func Catch2Args2[R any, A1 any, A2 any](f func(A1, A2) (R, error), a1 A1, a2 A2) (ret R, err error) {
	defer Recover(&err)
	return f(a1, a2)
}

func Catch2Args3[R any, A1 any, A2 any, A3 any](f func(A1, A2, A3) (R, error), a1 A1, a2 A2, a3 A3) (ret R, err error) {
	defer Recover(&err)
	return f(a1, a2, a3)
}

func Catch2Args4[R any, A1 any, A2 any, A3 any, A4 any](f func(A1, A2, A3, A4) (R, error), a1 A1, a2 A2, a3 A3, a4 A4) (ret R, err error) {
	defer Recover(&err)
	return f(a1, a2, a3, a4)
}
