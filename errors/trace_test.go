package errors_test

import (
	"fmt"
	"testing"

	"go.winto.dev/errors"
)

func TestTraceNil(t *testing.T) {
	err := errors.Trace(nil)
	if err != nil {
		t.Errorf("errors.Trace(nil) should be nil")
	}
}

func TestTraceMessage(t *testing.T) {
	err1 := errors.New("testerr")
	err2 := errors.Trace(err1)
	if err2.Error() != "testerr" {
		t.Errorf("errors.Trace should not change error message")
	}
}

func TestIndempotentTrace(t *testing.T) {
	err1 := errors.Errorf("testerr")
	err2 := errors.Trace(err1)
	err3 := errors.Trace(err2)

	if err1 != err2 || err2 != err3 {
		t.Errorf("traced error must be indempotent")
	}
}

func TestErrorf(t *testing.T) {
	err0 := fmt.Errorf("err1")
	err1 := errors.Trace(err0)
	err2 := errors.Errorf("err2: %w", err1)
	err3 := errors.Errorf("err3: %w", err2)

	if !errors.Is(err3, err2) {
		t.Errorf("invalid errors.Is")
	}

	if !errors.Is(err3, err1) {
		t.Errorf("invalid errors.Is")
	}

	if !errors.Is(err3, err0) {
		t.Errorf("invalid errors.Is")
	}
}

func TestTraceMessageErrorf(t *testing.T) {
	err1 := fmt.Errorf("testerr")
	err2 := errors.Errorf("testwrapper: %w", err1)
	if !errors.Is(errors.Unwrap(err2), err1) {
		t.Errorf("errors.Errorf should support %%w")
	}
}

func TestStackTrace(t *testing.T) {
	var err error
	funcAA(func() {
		funcBB(func() {
			err = errors.New("testerr")
		})
	})

	trace := errors.StackTrace(err)

	if !haveTrace(trace, "funcAA") {
		t.Errorf("errors.StackTrace should contains funcAA")
	}

	if !haveTrace(trace, "funcBB") {
		t.Errorf("errors.StackTrace should contains funcBB")
	}
}

func TestNonTraced(t *testing.T) {
	if errors.StackTrace(fmt.Errorf("testerror")) != nil {
		t.Errorf("errors.StackTrace on non traced error should return nil")
	}
}

type myErr struct{ msg string }

func (e *myErr) Error() string { return e.msg }

func TestErrorsAs(t *testing.T) {
	var target *myErr

	err := errors.Trace(&myErr{msg: "testerr"})

	if !errors.As(err, &target) {
		t.Errorf("invalid errors.As")
	}

	if err.Error() != target.msg {
		t.Errorf("invalid errors.As")
	}
}

func TestDeepTracedErrro(t *testing.T) {
	var err error
	funcAA(func() {
		err = fmt.Errorf("test wrapper: %w", errors.New("test err"))
	})
	if !haveTrace(errors.StackTrace(err), "funcAA") {
		t.FailNow()
	}
}

func TestStackTraceOutputShouldUseFirstOnUnwrapSlice(t *testing.T) {
	var errAa error
	funcAA(func() { errAa = errors.New("test1") })
	var errBb error
	funcBB(func() { errBb = errors.New("test2") })
	err := errors.Join(errAa, errBb)
	traced := errors.Trace(err)
	if err != traced {
		t.Errorf("errors.Trace should not wrap unwrapslice")
	}
	if !haveTrace(errors.StackTrace(traced), "funcAA") {
		t.Errorf("errors.StackTrace should use contains funcAA")
	}

	err = errors.Errorf("multierr: %w %w", errAa, errBb)
	traced = errors.Trace(err)
	if err != traced {
		t.Errorf("errors.Trace should not wrap unwrapslice")
	}
	if !haveTrace(errors.StackTrace(traced), "funcAA") {
		t.Errorf("errors.StackTrace should use contains funcAA")
	}
}
