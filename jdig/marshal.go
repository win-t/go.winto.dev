package jdig

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Wrap [json.Unmarshal], accept only string or []byte.
func Unmarshal(data any, options ...func(*json.Decoder)) (any, error) {
	var input []byte
	switch v := data.(type) {
	case string:
		input = []byte(v)
	case []byte:
		input = v
	default:
		panic("jdig: input must be string or []byte")
	}
	var v any
	d := json.NewDecoder(bytes.NewReader(input))
	for _, opt := range options {
		opt(d)
	}
	err := d.Decode(&v)
	return v, err
}

func MustUnmarshal(data any, options ...func(*json.Decoder)) any {
	v, err := Unmarshal(data, options...)
	if err != nil {
		panic("jdig: unmarshal error: " + err.Error())
	}
	return v
}

func Marshal(v any, options ...func(*json.Encoder)) string {
	var sb strings.Builder
	e := json.NewEncoder(&sb)
	for _, opt := range options {
		opt(e)
	}

	err := e.Encode(v)
	if err != nil {
		panic("jdig: marshal error: " + err.Error())
	}
	return sb.String()
}
