package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

var prefixFunc = map[string]func(string) string{
	"__ENV_":      noopPrefix,
	"__ENVXML_":   xmlPrefix,
	"__ENVJSON_":  jsonPrefix,
	"__ENVJSONC_": jsonContentPrefix,
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s {filename}\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       if filename is -, it will use stdin and stdout\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	var data []byte
	var err error
	if filename == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(filename)
	}
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

	var prefixes []string
	for p := range prefixFunc {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })

	for _, key := range keys {
		val := os.Getenv(key)
		for _, prefix := range prefixes {
			data = bytes.ReplaceAll(data, []byte(prefix+key), []byte(prefixFunc[prefix](val)))
		}
	}

	if filename == "-" {
		io.Copy(os.Stdout, bytes.NewReader(data))
	} else {
		err = os.WriteFile(filename, data, 0o644)
	}
	check(err)
}

func noopPrefix(t string) string {
	return t
}

func xmlPrefix(t string) string {
	var buf bytes.Buffer
	err := xml.EscapeText(&buf, []byte(t))
	check(err)
	return buf.String()
}

func jsonPrefix(t string) string {
	data, err := json.Marshal(t)
	check(err)
	return string(data)
}

func jsonContentPrefix(t string) string {
	r := jsonPrefix(t)
	return r[1 : len(r)-2]
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
