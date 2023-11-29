package errorstestutil

import (
	"testing"

	"go.winto.dev/errors"
)

func TestWrapper(t *testing.T, f func()) {
	if err := errors.Catch(func() error { f(); return nil }); err != nil {
		t.Fatal(errors.Format(err))
		panic("unreachable")
	}
}
