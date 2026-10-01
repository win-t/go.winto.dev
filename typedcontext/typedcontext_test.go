package typedcontext

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"unsafe"
)

func TestTypeSizeMust0(t *testing.T) {
	if unsafe.Sizeof(key[struct {
		a string
		b int
		c float64
	}]{}) != 0 {
		t.FailNow()
	}
	if unsafe.Sizeof(key[interface {
		a() string
		b() int
		c() float64
	}]{}) != 0 {
		t.FailNow()
	}
}

func TestNormalOperation(t *testing.T) {
	ctx := context.Background()
	ctx = New(ctx, 10)
	if MustGet[int](ctx) != 10 {
		t.FailNow()
	}
	if _, ok := Get[float64](ctx); ok {
		t.FailNow()
	}
}

func TestIsolatedFromExplicitTypeReflection(t *testing.T) {
	ctx := context.Background()
	ctx = New(ctx, 10)
	ctx = context.WithValue(ctx, reflect.TypeOf(20), 20)
	if MustGet[int](ctx) != 10 {
		t.FailNow()
	}
}

func TestPanicIfNoValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.FailNow()
		}
	}()
	MustGet[int](context.Background())
}

type x interface {
	a()
}

type y struct{ v int }

func (y) a() {}

type z struct{ f func() }

func (z z) a() { z.f() }

func TestShouldWorkOnInterface(t *testing.T) {
	var a x = y{10}
	ctx := context.Background()
	ctx = New(ctx, a)
	b := MustGet[x](ctx)
	if b.(y).v != 10 {
		t.FailNow()
	}

	r := ""
	a = z{func() { r = "hello" }}
	ctx = New(ctx, a)
	MustGet[x](ctx).a()
	if r != "hello" {
		t.FailNow()
	}
}

func TestCancel(t *testing.T) {
	ctx, _ := WithCancel(context.Background())
	if !Cancel(ctx) {
		t.FailNow()
	}
	if ctx.Err() == nil {
		t.FailNow()
	}
}

func TestCancelCause(t *testing.T) {
	ctx, _ := WithCancelCause(context.Background())
	if !CancelCause(ctx, errors.New("test")) {
		t.FailNow()
	}
	if ctx.Err() == nil {
		t.FailNow()
	}
	if context.Cause(ctx).Error() != "test" {
		t.FailNow()
	}
}
