package jdig

import (
	"encoding/json"
	"strings"
)

// Wrap [json.Unmarshal], accept only string or []byte.
func Unmarshal(data any) (any, error) {
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
	err := json.Unmarshal(input, &v)
	return v, err
}

func MustUnmarshal(data any) any {
	v, err := Unmarshal(data)
	if err != nil {
		panic("jdig: unmarshal error: " + err.Error())
	}
	return v
}

type MarshalOpt struct{ f func(*json.Encoder) }

func SetEscapeHTML(on bool) MarshalOpt {
	return MarshalOpt{f: func(e *json.Encoder) { e.SetEscapeHTML(on) }}
}

func SetIndent(prefix, indent string) MarshalOpt {
	return MarshalOpt{f: func(e *json.Encoder) { e.SetIndent(prefix, indent) }}
}

func Marshal(v any, options ...MarshalOpt) string {
	var sb strings.Builder
	e := json.NewEncoder(&sb)
	for _, opt := range options {
		opt.f(e)
	}

	err := e.Encode(v)
	if err != nil {
		panic("jdig: marshal error: " + err.Error())
	}
	return sb.String()
}
