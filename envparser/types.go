package envparser

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"reflect"
	"time"
)

type Unmarshaler interface{ UnmarshalEnv(val string) error }

// for type that support parsing from string, but not implement TextUnmarshaler or BinaryUnmarshaler
// also not json unmarshalable
var nativeUnmarshaler = map[reflect.Type]func(val string) (any, error){
	reflect.TypeOf((*time.Duration)(nil)).Elem():  func(val string) (any, error) { return time.ParseDuration(val) },
	reflect.TypeOf((**time.Location)(nil)).Elem(): func(val string) (any, error) { return time.LoadLocation(val) },
}

type Base64 []byte

func (s *Base64) UnmarshalEnv(val string) error {
	data, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	*s = Base64(data)
	return nil
}

type File []byte

func (b *File) UnmarshalEnv(val string) error {
	data, err := os.ReadFile(val)
	if err != nil {
		return err
	}
	*b = File(data)
	return nil
}

type Base64OfJSON[T any] struct {
	Value T
}

func (b *Base64OfJSON[T]) UnmarshalEnv(val string) error {
	data, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.Value)
}
