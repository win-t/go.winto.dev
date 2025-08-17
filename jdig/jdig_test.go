package jdig

import (
	"errors"
	"testing"
)

func catch(f func()) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = rec.(error)
		}
	}()
	f()
	return nil
}

func TestUnmarshalErr(t *testing.T) {
	err := catch(func() {
		Unmarshal(``)
	})
	if err == nil {
		t.Fatal()
	}
	if errors.Unwrap(err) == nil {
		t.Fatal()
	}
}

func TestMarshal(t *testing.T) {
	if Marshal(123) != "123" {
		t.Fatal()
	}
}

func TestMain(t *testing.T) {
	v := Unmarshal(`{"a": 1, "b": [{"c": 12, "n": null}, {"c": 13, "d": true}]}`)
	if a := Float(v, "a"); a != 1 {
		t.Fatal()
	}
	if a := Float(v, "a"); a != 1 {
		t.Fatal()
	}
	if a := String(v, "a"); a != "" {
		t.Fatal()
	}
	if a := Float(v, "b", "y", "z"); a != 0 {
		t.Fatal()
	}
	if !IsNull(v, "zz") {
		t.Fatal()
	}
	if Float(v, "b", 0, "c") != 12 {
		t.Fatal()
	}
	if !IsNull(v, "b", 0, "n") {
		t.Fatal()
	}
	if !Bool(v, "b", 1, "d") {
		t.Fatal()
	}
	if Arr(v, 1, 2, 3) != nil {
		t.Fatal()
	}
	if Obj(v, "b", 1) == nil {
		t.Fatal()
	}
}

func TestRecursiveDeleteKeyIfEmpty(t *testing.T) {
	v := map[string]any{
		"a": nil,
		"b": 0,
		"c": 0.0,
		"d": []any{},
		"e": map[string]any{},
		"e2": map[string]any{
			"a": nil,
		},
		"f": "",
		"g": []any{
			1,
			map[string]any{},
			map[string]any{
				"a": 0,
				"b": "something",
			},
		},
	}
	RecursiveDeleteKeyIfEmpty(v)
	if len(v) != 1 {
		t.Fatal()
	}
	if len(Arr(v, "g")) != 3 {
		t.Fatal()
	}
	if len(Obj(v, "g", 2)) != 1 {
		t.Fatal()
	}
}
