package errors_test

import (
	"strings"
	"testing"

	"go.winto.dev/errors"
)

func TestFormat(t *testing.T) {
	var err error
	funcAA(func() {
		funcBB(func() {
			err = errors.New("err1")
			err = errors.Errorf("err2: %w", err)
		})
	})

	f := errors.Format(err)

	if !strings.Contains(f, "funcAA") ||
		!strings.Contains(f, "funcBB") ||
		!strings.Contains(f, "err1") ||
		!strings.Contains(f, "err2") {
		t.FailNow()
	}
}

func TestFormatFilter(t *testing.T) {
	var err error
	funcAA(func() {
		funcBB(func() {
			err = errors.New("err1")
			err = errors.Errorf("err2: %w", err)
		})
	})

	f := errors.FormatWithFilterPkgs(err)

	if strings.Contains(f, "funcAA") ||
		strings.Contains(f, "funcBB") ||
		!strings.Contains(f, "err1") ||
		!strings.Contains(f, "err2") {
		t.FailNow()
	}
}
