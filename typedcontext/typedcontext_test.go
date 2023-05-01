package typedcontext

import (
	"context"
	"reflect"
	"testing"
)

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
