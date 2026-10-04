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

// run f, if f panic or returned, that value will be returned by this function.
func Catch(f func() error) (err error) {
	defer Recover(&err)
	return f()
}

// like [Catch] but suitable for function doesn't expect to return error
func Catch0(f func()) (err error) {
	defer Recover(&err)
	f()
	return nil
}

// like [Catch] but suitable for function expect to return single value
func Catch1[Ret any](f func() Ret) (ret Ret, err error) {
	defer Recover(&err)
	return ret, Catch0(func() { ret = f() })
}

// like [Catch] but suitable for function that return value and error.
func Catch2[Ret any](f func() (Ret, error)) (ret Ret, err error) {
	defer Recover(&err)
	return f()
}
