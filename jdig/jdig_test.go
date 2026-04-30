package jdig

import (
	"testing"
)

func TestQuery(t *testing.T) {
	v := MustUnmarshal(`{"a": 1, "b": [{"c": 12, "n": null}, {"c": 13, "d": true}]}`)
	if a := Float(v, "a"); a != 1 {
		t.Fatal()
	}
	if a := Int(v, "a"); a != 1 {
		t.Fatal()
	}
	if a := String(v, "a"); a != "" {
		t.Fatal()
	}
	if a := Float(v, "b", "y", "z"); a != 0 {
		t.Fatal()
	}
	if Any(v, "zz") != nil {
		t.Fatal()
	}
	if Float(v, "b", 0, "c") != 12 {
		t.Fatal()
	}
	if Any(v, "b", 0, "n") != nil {
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

func TestDeppCopy(t *testing.T) {
	a := JObj{
		"a": JArr{
			JObj{
				"b": 1,
			},
		},
	}
	b := DeepCopy(a)
	Obj(a, "a", 0)["b"] = 100
	if Int(b, "a", 0, "b") != 1 {
		t.Fatal()
	}
}
