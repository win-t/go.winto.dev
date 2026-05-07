package jdig

import (
	"reflect"
	"testing"
)

func TestMergeObj(t *testing.T) {
	a := Merge(
		JObj{
			"a": 1,
			"b": JObj{
				"c": 2,
				"d": JObj{
					"e": 3,
				},
			},
		},
		JObj{
			"b": JObj{
				"c": 20,
			},
			"x": JObj{
				"y": JObj{
					"z": 4,
				},
			},
		},
	)
	aExpected := JObj{
		"a": 1,
		"b": JObj{
			"c": 20,
			"d": JObj{
				"e": 3,
			},
		},
		"x": JObj{
			"y": JObj{
				"z": 4,
			},
		},
	}
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}
}

func TestMergeArr(t *testing.T) {
	a := Merge(
		JObj{
			"b": JArr{1},
			"c": JArr{1, 2, 3},
		},
		JObj{
			"a": JArr{1, 2, 3},
			"b": JArr{2, 3},
			"c": JArr{4},
		},
	)
	aExpected := JObj{
		"a": JArr{1, 2, 3},
		"b": JArr{2, 3},
		"c": JArr{4, 2, 3},
	}
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}
}

func TestDiscardKey(t *testing.T) {
	a := Merge(
		JObj{
			"a": 1,
			"b": DiscardKey(),
		},
	)
	var aExpected any = JObj{
		"a": 1,
	}
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}

	a = Merge(DiscardKey())
	aExpected = nil
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}
}

func TestMerge(t *testing.T) {
	var a any = Merge(
		JObj{
			"a": 1,
			"b": JObj{
				"c": 12,
			},
			"c": JObj{
				"d": 1,
			},
		},
		JObj{
			"b": 1,
			"c": JObj{
				"d": 2,
				"e": 1,
			},
		},
		JObj{
			"f": JObj{
				"a": 1,
			},
		},
		JObj{
			"x": MergeCallback(
				func(dst any) any {
					a, _ := dst.(int)
					return a + 12
				},
			),
			"y": MergeCallback(
				func(dst any) any {
					return JArr{1, 2, 3, 4}
				},
			),
		},
		JObj{
			"z": 20,
		},
		JObj{
			"z": MergeCallback(
				func(dst any) any {
					a, _ := dst.(int)
					return a + 12
				},
			),
		},
	)
	aExpected := JObj{
		"a": 1,
		"b": 1,
		"c": JObj{
			"d": 2,
			"e": 1,
		},
		"f": JObj{
			"a": 1,
		},
		"x": 12,
		"y": JArr{1, 2, 3, 4},
		"z": 32,
	}
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}
}

func TestKeep(t *testing.T) {
	a := Merge(
		JObj{
			"a": 1,
			"b": 2,
			"c": 3,
		},
		JObj{
			"a": 10,
			"b": Keep(),
			"c": 30,
		},
	)
	aExpected := JObj{
		"a": 10,
		"b": 2,
		"c": 30,
	}
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}
}

func TestReplace(t *testing.T) {
	a := Merge(
		JObj{
			"a": 1,
		},
		Replace(JObj{
			"x": 10,
		}),
	)
	aExpected := JObj{
		"x": 10,
	}
	if !reflect.DeepEqual(a, aExpected) {
		t.Fatal()
	}
}

func TestMergeCallbackGenerator(t *testing.T) {
	var count int
	var genHandler func(v any) MergeHandler
	genHandler = func(v any) MergeHandler {
		cb := func(dst any) any {
			count++
			if dst == nil {
				return v
			} else {
				return genHandler(v.(int) + dst.(int))
			}
		}
		return MergeCallback(cb)
	}

	v := Merge(
		JObj{
			"a": 1,
		},
		JObj{
			"a": 2,
		},
		JObj{
			"a": 3,
		},
		JObj{
			"a": 4,
		},
		JObj{
			"a": genHandler(5),
		},
	)
	if Int(v, "a") != 15 {
		t.Fatal()
	}
	if count != 5 {
		t.Fatal()
	}

	shouldPanic(t, func() {
		Merge(
			JObj{
				"a": MergeCallback(func(dst any) any {
					return MergeCallback(func(dst any) any { return nil })
				}),
			},
		)
	})

	v = Merge(
		JArr{
			DiscardKey(),
			MergeCallback(func(dst any) any {
				return DiscardKey()
			}),
		},
	)
	vExpected := JArr{nil, nil}
	if !reflect.DeepEqual(v, vExpected) {
		t.Fatal()
	}
}

func shouldPanic(t *testing.T, f func()) {
	defer func() {
		if recover() == nil {
			t.Fatal()
		}
	}()
	f()
}

func TestNilDs(t *testing.T) {
	// this test just ensure no panic
	Merge(JObj(nil), JObj{"a": "b"})
	merge(JArr(nil), JArr{1, 2})
}
