package jdig

import "encoding/json"

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

func Marshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic("jdig: marshal error: " + err.Error())
	}
	return string(data)
}
