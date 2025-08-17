// Package jdig provides a simple way to deal with JSON but in ineffecent way.
package jdig

import (
	"encoding/json"
	"reflect"
)

type IError = error

type Error struct{ IError }

func (e Error) Unwrap() error { return e.IError }

func Unmarshal(data any) any {
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
	if err != nil {
		panic(Error{IError: err})
	}
	return v
}

func Marshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(Error{IError: err})
	}
	return string(data)
}

func Any(v any, keys ...any) any {
	for _, key := range keys {
		if v == nil {
			break
		}
		switch key := key.(type) {
		case string:
			if m, ok := v.(map[string]any); ok {
				v = m[key]
			} else {
				v = nil
			}
		case int:
			if a, ok := v.([]any); ok && 0 <= key && key < len(a) {
				v = a[key]
			} else {
				v = nil
			}
		default:
			panic("jdig: key must be string or integer")
		}
	}
	return v
}

func get[T any](v any, keys ...any) (ret T) {
	value := Any(v, keys...)
	ret, _ = value.(T)
	return ret
}

func Obj(v any, keys ...any) map[string]any {
	return get[map[string]any](v, keys...)
}

func Arr(v any, keys ...any) []any {
	return get[[]any](v, keys...)
}

func String(v any, keys ...any) string {
	return get[string](v, keys...)
}

func Float(v any, keys ...any) float64 {
	return get[float64](v, keys...)
}

func Bool(v any, keys ...any) bool {
	return get[bool](v, keys...)
}

func IsNull(v any, keys ...any) bool {
	return Any(v, keys...) == nil
}

func RecursiveDeleteKeyIfEmpty(v any) {
	if v == nil {
		return
	}
	switch v := v.(type) {
	case map[string]any:
		for k, vk := range v {
			if vk == nil {
				delete(v, k)
			} else if c, ok := vk.(map[string]any); ok && len(c) == 0 {
				delete(v, k)
			} else if c, ok := vk.([]any); ok && len(c) == 0 {
				delete(v, k)
			} else if reflect.ValueOf(vk).IsZero() {
				delete(v, k)
			} else {
				RecursiveDeleteKeyIfEmpty(vk)
			}
		}
	case []any:
		for i := range v {
			RecursiveDeleteKeyIfEmpty(v[i])
		}
	}
}
