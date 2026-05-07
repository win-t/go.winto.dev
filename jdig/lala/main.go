package main

import (
	"fmt"

	"go.winto.dev/jdig"
)

type jobj = map[string]any
type jarr = []any

func main() {
	a := jdig.Merge(jobj(nil), jobj{
		"foo": "bar",
	})

	fmt.Println(jdig.Marshal(a))
}
