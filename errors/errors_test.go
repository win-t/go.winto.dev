package errors_test

import (
	stderrors "errors"
	"strings"
	"testing"

	"go.winto.dev/errors"
)

func funcAA(f func()) { f() }

func funcBB(f func()) { f() }

func haveTrace(ls []errors.Location, what string) bool {
	for _, l := range ls {
		if strings.Contains(l.Func(), what) {
			return true
		}
	}
	return false
}

func TestCheck(t *testing.T) {
	err := errors.Catch(func() error {
		funcAA(func() {
			errors.Check(stderrors.New("testerr"))
		})
		return nil
	})

	if !haveTrace(errors.StackTrace(err), "funcAA") {
		t.Fatalf("should contain funcAA")
	}
}

func TestAssert(t *testing.T) {
	err := errors.Catch(func() error {
		funcAA(func() {
			errors.Expect(false, "")
		})
		return nil
	})

	if !haveTrace(errors.StackTrace(err), "funcAA") {
		t.Fatalf("should contain funcAA")
	}
}

func TestJoin(t *testing.T) {
	a := errors.New("a")
	b := errors.New("b")
	c := errors.Join(a, b)

	errs := c.(interface{ Unwrap() []error }).Unwrap()
	if len(errs) != 2 {
		t.Fatalf("invalid Unwrap")
	}

	if !errors.Is(c, b) {
		t.Fatalf("invalid Is")
	}

	if !errors.Is(c, a) {
		t.Fatalf("invalid Is")
	}

	if errors.Unwrap(c) != nil {
		t.Fatalf("invalid unwrap")
	}

	if errors.Trace(errors.Trace(c)) != c {
		t.Fatalf("invalid Trace")
	}
}

func TestJoinErrorf(t *testing.T) {
	a := errors.New("a")
	b := errors.New("b")
	c := errors.Errorf("hai %w %w", a, b)

	errs := c.(interface{ Unwrap() []error }).Unwrap()
	if len(errs) != 2 {
		t.Fatalf("invalid Unwrap")
	}

	if !errors.Is(c, b) {
		t.Fatalf("invalid Is")
	}

	if !errors.Is(c, a) {
		t.Fatalf("invalid Is")
	}

	if errors.Unwrap(c) != nil {
		t.Fatalf("invalid unwrap")
	}

	if errors.Trace(errors.Trace(c)) != c {
		t.Fatalf("invalid Trace")
	}
}
