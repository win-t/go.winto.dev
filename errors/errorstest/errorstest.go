package errorstest

import (
	"go.winto.dev/errors"
)

// run f with [errors.Catch], if error happen pass it to t.Fatal.
//
// default errFormater is [errors.Format].
//
// [errors.Catch]: https://pkg.go.dev/go.winto.dev/errors#Catch
// [errors.Format]: https://pkg.go.dev/go.winto.dev/errors#Format
func Catch(t interface{ Fatal(...any) }, errFormater func(error) string, f func()) {
	if err := errors.Catch(func() error { f(); return nil }); err != nil {
		if errFormater == nil {
			errFormater = errors.Format
		}
		t.Fatal(errFormater(err))
		panic("unreachable")
	}
}
