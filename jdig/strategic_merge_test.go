package jdig

import (
	"reflect"
	"testing"
)

func TestStrategicMerge(t *testing.T) {
	v1 := Merge(
		JArr{
			JObj{
				"name":  "a",
				"value": "a1",
			},
			11,
			JObj{
				"value": "xxx",
			},
			JObj{
				"name":  "b",
				"value": "b1",
			},
			JObj{
				"name": "to-be-deleted",
			},
			JObj{
				"name": "to-be-replaced",
				"a":    "b",
				"c":    "d",
			},
		},
		StrategicMerge("name",
			JArr{
				JObj{
					"name":  "c",
					"value": "c2",
				},
				JObj{
					"name":  "b",
					"value": "b2",
				},
				22,
				JObj{
					"$patch": "nothing-but-will-be-removed",
					"value":  "yyy",
				},
				JObj{
					"name":   "to-be-deleted",
					"$patch": "delete",
				},
				JObj{
					"name":   "to-be-replaced",
					"$patch": "replace",
					"x":      "y",
					"z":      "z",
				},
				JObj{
					"name":   "newkey",
					"$patch": "replace",
					"p":      "q",
				},
			},
		),
	)
	v2 := JArr{
		JObj{
			"name":  "a",
			"value": "a1",
		},
		11,
		JObj{
			"value": "xxx",
		},
		JObj{
			"name":  "b",
			"value": "b2",
		},
		JObj{
			"name": "to-be-replaced",
			"x":    "y",
			"z":    "z",
		},
		JObj{
			"name":  "c",
			"value": "c2",
		},
		22,
		JObj{
			"value": "yyy",
		},
		JObj{
			"name": "newkey",
			"p":    "q",
		},
	}
	if !reflect.DeepEqual(v1, v2) {
		t.Fatal()
	}

	if !reflect.DeepEqual(
		Merge(12, StrategicMerge("hello", JArr{11})),
		JArr{11},
	) {
		t.Fatal()
	}
}

func TestStrategicMergeFn(t *testing.T) {
	v1 := Merge(
		JArr{
			JObj{
				"keya":  "a",
				"keyb":  "a",
				"value": "a1",
			},
			11,
			JObj{
				"value": "xxx",
			},
			JObj{
				"keya":  "b",
				"keyb":  "b",
				"value": "b1",
			},
			JObj{
				"keya": "to-be-deleted",
				"keyb": "to-be-deleted",
			},
			JObj{
				"keya": "to-be-replaced",
				"keyb": "to-be-replaced",
				"a":    "b",
				"c":    "d",
			},
			JObj{
				"keya": "onlysameonkeya",
				"keyb": "hello",
				"data": 10,
			},
		},
		StrategicMergeFn(
			func(a, b JObj) bool {
				akeya, ok := AnyExists(a, "keya")
				if !ok {
					return false
				}
				akeyb, ok := AnyExists(a, "keyb")
				if !ok {
					return false
				}
				bkeya, ok := AnyExists(b, "keya")
				if !ok {
					return false
				}
				bkeyb, ok := AnyExists(b, "keyb")
				if !ok {
					return false
				}
				return akeya == bkeya && akeyb == bkeyb
			},
			JArr{
				JObj{
					"keya":  "c",
					"keyb":  "c",
					"value": "c2",
				},
				JObj{
					"keya":  "b",
					"keyb":  "b",
					"value": "b2",
				},
				22,
				JObj{
					"$patch": "nothing-but-will-be-removed",
					"value":  "yyy",
				},
				JObj{
					"keya":   "to-be-deleted",
					"keyb":   "to-be-deleted",
					"$patch": "delete",
				},
				JObj{
					"keya":   "to-be-replaced",
					"keyb":   "to-be-replaced",
					"$patch": "replace",
					"x":      "y",
					"z":      "z",
				},
				JObj{
					"keya":   "newkey",
					"keyb":   "newkey",
					"$patch": "replace",
					"p":      "q",
				},
				JObj{
					"keya": "onlysameonkeya",
					"keyb": "world",
					"data": 20,
				},
			},
		),
	)
	v2 := JArr{
		JObj{
			"keya":  "a",
			"keyb":  "a",
			"value": "a1",
		},
		11,
		JObj{
			"value": "xxx",
		},
		JObj{
			"keya":  "b",
			"keyb":  "b",
			"value": "b2",
		},
		JObj{
			"keya": "to-be-replaced",
			"keyb": "to-be-replaced",
			"x":    "y",
			"z":    "z",
		},
		JObj{
			"keya": "onlysameonkeya",
			"keyb": "hello",
			"data": 10,
		},
		JObj{
			"keya":  "c",
			"keyb":  "c",
			"value": "c2",
		},
		22,
		JObj{
			"value": "yyy",
		},
		JObj{
			"keya": "newkey",
			"keyb": "newkey",
			"p":    "q",
		},
		JObj{
			"keya": "onlysameonkeya",
			"keyb": "world",
			"data": 20,
		},
	}
	if !reflect.DeepEqual(v1, v2) {
		t.Fatal()
	}
}
