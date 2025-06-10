package errors

var (
	_ error       = (*traced[error])(nil)
	_ stacktracer = (*traced[error])(nil)
	_ unwrap      = (*traced[error])(nil)

	_ error       = (*traced[[]error])(nil)
	_ stacktracer = (*traced[[]error])(nil)
	_ unwrapslice = (*traced[[]error])(nil)

	_ error       = (*traced[any])(nil)
	_ stacktracer = (*traced[any])(nil)
)
