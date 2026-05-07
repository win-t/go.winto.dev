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
