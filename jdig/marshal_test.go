package jdig

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNum(t *testing.T) {
	v := MustUnmarshal(`{"a": 1, "b": 1.25}`, UseNumber())

	switch Any(v, "a").(type) {
	case json.Number:
	default:
		t.Fatal()
	}

	switch Any(v, "b").(type) {
	case json.Number:
	default:
		t.Fatal()
	}

	if Float(v, "a") != 1 {
		t.Fatal()
	}

	if Float(v, "b") != 1.25 {
		t.Fatal()
	}

	if Int(v, "a") != 1 {
		t.Fatal()
	}

	if Int(v, "b") != 1 {
		t.Fatal()
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	src := JObj{
		"A": "hello",
		"B": 12,
	}
	type dstT struct {
		A string
		B int
	}
	var dst dstT
	MustMarshalUnmarshal(&dst, src)
	if !reflect.DeepEqual(dst, dstT{
		A: "hello",
		B: 12,
	}) {
		t.Fatal()
	}
}
