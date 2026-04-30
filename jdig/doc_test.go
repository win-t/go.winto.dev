package jdig_test

import (
	"fmt"
	"reflect"

	"go.winto.dev/jdig"
)

func Example() {
	a := jdig.MustUnmarshal(`
		{
			"a": {
				"b": [
					{"c": 1},
					{"e": 2}
				]
			},
			"hello": "world"
		}
	`)

	fmt.Println(jdig.Int(a, "a", "b", 0, "c"))
	fmt.Println(jdig.Float(a, "a", "b", 1, "e"))
	fmt.Println(jdig.String(a, "hello"))
	fmt.Println(jdig.String(a, "some", "unexisting", "path") == "")
	// Output:
	// 1
	// 2
	// world
	// true
}

func ExampleDiscardKey() {
	type jobj = map[string]any
	type jarr = []any

	a := jdig.MustUnmarshal(`
		{
			"a": {
				"b": "hei",
				"c": "hello"
			},
			"hello": "world",
			"world": "hello"
		}
	`)

	fmt.Println(reflect.DeepEqual(
		jdig.Merge(a, jobj{
			"a": jobj{
				"b": nil,
				"c": jdig.DiscardKey(),
			},
			"hello": nil,
			"world": jdig.DiscardKey(),
		}),
		jdig.MustUnmarshal(`{
			"a": {
				"b": null
			},
			"hello": null
		}`),
	))

	// Output:
	// true
}

func ExampleMergeCallback() {
	type jobj = map[string]any
	type jarr = []any

	var createAdderHandler func(v float64) jdig.MergeHandler
	createAdderHandler = func(v float64) jdig.MergeHandler {
		return jdig.MergeCallback(func(dst any, defaultFn func(dst any, src any) any) any {
			if dst == nil { // never return another MergeHandler when dst is nil
				return v
			}
			return createAdderHandler(dst.(float64) + v)
		})
	}

	a := jdig.Arr(jdig.MustUnmarshal(`
		[
			{"a": [10, {"a": 1}]},
			{"a": [20, {"a": 2}]},
			{"a": [30, {"a": 3}]}
		]
	`))

	a = append(a, jobj{
		"a": jarr{
			jdig.Keep(),
			jobj{
				"a": createAdderHandler(4),
			},
		},
	})

	fmt.Println(reflect.DeepEqual(
		jdig.Merge(a...),
		jdig.MustUnmarshal(`
			{"a": [30, {"a": 10}]}
		`),
	))
	// Output:
	// true
}

func ExampleStrategicMerge() {
	a := jdig.Arr(jdig.MustUnmarshal(`
		[
			{"name": "main-container",    "image": "main-v1"},
			{"name": "unused-container",  "image": "sidecar-v1"},
			{"name": "sidecar-container", "image": "sidecar-v1"}
		]
	`))

	b := jdig.Arr(jdig.MustUnmarshal(`
		[
			{"name": "main-container",      "image":  "main-v2"},
			{"name": "sidecar-container-2", "image":  "sidecar-v2"},
			{"name": "unused-container",    "$patch": "delete"}
		]
	`))

	expected := jdig.MustUnmarshal(`
		[
			{"name": "main-container",      "image":  "main-v2"},
			{"name": "sidecar-container",   "image": "sidecar-v1"},
			{"name": "sidecar-container-2", "image":  "sidecar-v2"}
		]
	`)

	merged := jdig.Merge(
		a,
		jdig.StrategicMerge("name", b),
	)

	fmt.Println(reflect.DeepEqual(merged, expected))
	// Output:
	// true
}
