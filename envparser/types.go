package envparser

import (
	"encoding/base64"
	"encoding/json"
	"os"
)

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
