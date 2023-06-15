package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	prefix := "__ENV_"
	filename := os.Args[1]

	data, err := os.ReadFile(filename)
	check(err)

	var keys []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		keys = append(keys, parts[0])
	}
	// sort by key length, longer first
	// with ABCD=one and ABC=two
	// __ENV_ABCD will not be replaced by twoD
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })

	for _, key := range keys {
		val := os.Getenv(key)
		data = bytes.ReplaceAll(data, []byte(prefix+key), []byte(val))
	}

	err = os.WriteFile(filename, data, 0o644)
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
