package jdig

import "reflect"

type JObj = map[string]any
type JArr = []any

func Any(v any, keys ...any) any {
	for _, key := range keys {
		if v == nil {
			break
		}
		switch key := key.(type) {
		case string:
			if m, ok := v.(JObj); ok {
				v = m[key]
			} else {
				v = nil
			}
		case int:
			if a, ok := v.(JArr); ok && 0 <= key && key < len(a) {
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

func get[T any](v any, keys ...any) T {
	v = Any(v, keys...)
	r, _ := v.(T)
	return r
}

func Obj(v any, keys ...any) JObj {
	return get[JObj](v, keys...)
}

func Arr(v any, keys ...any) JArr {
	return get[JArr](v, keys...)
}

func String(v any, keys ...any) string {
	return get[string](v, keys...)
}

func Bool(v any, keys ...any) bool {
	return get[bool](v, keys...)
}

func getConvert[T any](v any, keys ...any) T {
	v = Any(v, keys...)
	var r T
	if v == nil {
		return r
	}
	rT := reflect.TypeOf(r)
	val := reflect.ValueOf(v)
	if val.CanConvert(rT) {
		return val.Convert(rT).Interface().(T)
	}
	return r
}

func Float(v any, keys ...any) float64 {
	return getConvert[float64](v, keys...)
}

func Int(v any, keys ...any) int {
	return getConvert[int](v, keys...)
}

func DeepCopy(v any) any {
	switch v := v.(type) {
	case JObj:
		copied := make(JObj, len(v))
		for k, v := range v {
			copied[k] = DeepCopy(v)
		}
		return copied
	case JArr:
		copied := make(JArr, len(v))
		for i, v := range v {
			copied[i] = DeepCopy(v)
		}
		return copied
	default:
		return v
	}
}

// NormalizeNilArrayAndMaps normalizes nil arrays and maps to empty ones.
func NormalizeNilArrayAndMaps(v any) any {
	switch v := v.(type) {
	case JArr:
		if v == nil {
			return make(JArr, 0)
		}
		for i := range v {
			v[i] = NormalizeNilArrayAndMaps(v[i])
		}
		return v
	case JObj:
		if v == nil {
			return make(JObj)
		}
		for k, vv := range v {
			v[k] = NormalizeNilArrayAndMaps(vv)
		}
		return v
	default:
		return v
	}
}
