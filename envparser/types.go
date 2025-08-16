package envparser

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"reflect"
	"time"
)

type Unmarshaler interface{ UnmarshalEnv(val string) error }

var (
	unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	timeType        = reflect.TypeOf((*time.Time)(nil)).Elem()
	durationType    = reflect.TypeOf((*time.Duration)(nil)).Elem()
	locationType    = reflect.TypeOf((**time.Location)(nil)).Elem()
	urlType         = reflect.TypeOf((**url.URL)(nil)).Elem()
)

var nativeUnmarshaler = map[reflect.Type]func(val string) (any, error){
	timeType:     func(val string) (any, error) { return time.Parse(time.RFC3339Nano, val) },
	durationType: func(val string) (any, error) { return time.ParseDuration(val) },
	locationType: func(val string) (any, error) { return time.LoadLocation(val) },
	urlType:      func(val string) (any, error) { return url.Parse(val) },
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
