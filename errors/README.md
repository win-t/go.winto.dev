# What is this?

[![GoReference](https://pkg.go.dev/badge/go.winto.dev/errors)](https://pkg.go.dev/go.winto.dev/errors)

This package can be used as drop-in replacement for standard errors package.

This package provide `func StackTrace(error) []Location` to get the stack trace.

Stack trace can be attached to any `error` by passing it to `func Trace(error) error`.

`New`, and `Errorf` function will return error that have stack trace.
