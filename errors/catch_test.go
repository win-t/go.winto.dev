package errors_test

import (
	"fmt"
	"testing"

	"go.winto.dev/errors"
)

func TestCatch(t *testing.T) {
	check := func(shouldNil bool, shouldHaveTrace bool, f func() error) {
		err := errors.Catch(func() error {
			var err error
			funcAA(func() {
				err = f()
			})
			return err
		})

		if shouldNil {
			if err != nil {
				t.Errorf("errors.Catch should return nil when f returning nil")
			}
		} else if err == nil {
			t.Errorf("errors.Catch should return non-nil when f returning non-nil or panic")
		}

		if shouldHaveTrace && !haveTrace(errors.StackTrace(err), "funcAA") {
			t.Errorf("errors.Catch trace should contains funcAA")
		}
	}

	check(true, false, func() error {
		return nil
	})

	check(false, true, func() error {
		return errors.New("testerr")
	})

	check(false, false, func() error {
		return fmt.Errorf("testerr")
	})

	check(false, true, func() error {
		panic(errors.New("testerr"))
	})

	check(false, true, func() error {
		panic(fmt.Errorf("testerr"))
	})

	check(false, true, func() error {
		var something interface{ something() }
		// this trigger nil pointer exception
		something.something()
		return nil
	})

	check(false, true, func() error {
		panic("a test string")
	})
}

func TestCatchMultipleReturn(t *testing.T) {
	a, err := errors.Catch2(func() (int, error) { return 10, nil })
	if a != 10 || err != nil {
		t.FailNow()
	}

	orierr := errors.New("orierr")

	a, err = errors.Catch2(func() (int, error) { return 10, orierr })
	if a != 10 || !errors.Is(err, orierr) {
		t.FailNow()
	}
}

func TestCatchMultiErr(t *testing.T) {
	err := errors.Catch(func() error {
		a := errors.New("a")
		b := errors.New("b")
		funcAA(func() {
			panic(errors.Join(a, b))
		})
		return nil
	})
	if !haveTrace(errors.StackTrace(err), "funcAA") {
		t.Errorf("errors.Catch trace should contains funcAA")
	}
}

func TestCatchDeepTracedErrro(t *testing.T) {
	err := errors.Catch(func() error {
		funcAA(func() {
			panic(fmt.Errorf("test wrapper: %w", errors.New("test err")))
		})
		return nil
	})
	if !haveTrace(errors.StackTrace(err), "funcAA") {
		t.FailNow()
	}
}
