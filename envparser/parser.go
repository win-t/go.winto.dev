package envparser

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"time"
)

type Unmarshaler interface{ UnmarshalEnv(val string) error }

var (
	unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	timeType        = reflect.TypeOf((*time.Time)(nil)).Elem()
	durationType    = reflect.TypeOf((*time.Duration)(nil)).Elem()
	locationType    = reflect.TypeOf((**time.Location)(nil)).Elem()
)

// Unmarshal into struct.
//
// target must be non-nil pointer to struct.
//
// "env" tag in each field in target struct will be fetched from environment variable.
// If "env" tag is not found, field name is used.
// If "env" tag is empty string, the field is skipped.
//
// if the field implement [Unmarshaler] interface, it will be used.
func Unmarshal(target any) error {
	return UnmarshalWithPrefix(target, "")
}

// Like [Unmarshal] but we can specify the prefix key.
func UnmarshalWithPrefix(target any, prefix string) error {
	targetVal := valueOfPointerToStruct(target)

	var parseError ParseError

	for i, t := 0, targetVal.Type(); i < t.NumField(); i++ {
		key := lookupEnvName(targetVal.Type().Field(i))
		if key == "" {
			continue
		}

		val, ok := os.LookupEnv(prefix + key)
		if !ok {
			continue
		}

		f := targetVal.Field(i)
		switch {
		case f.Addr().Type().Implements(unmarshalerType):
			if err := f.Addr().Interface().(Unmarshaler).UnmarshalEnv(val); err != nil {
				parseError.append(key, val, err)
			}
		case f.Kind() == reflect.String:
			f.SetString(val)
		case f.Type() == timeType:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				parseError.append(key, val, err)
			} else {
				f.Set(reflect.ValueOf(t))
			}
		case f.Type() == durationType:
			d, err := time.ParseDuration(val)
			if err != nil {
				parseError.append(key, val, err)
			} else {
				f.Set(reflect.ValueOf(d))
			}
		case f.Type() == locationType:
			l, err := time.LoadLocation(val)
			if err != nil {
				parseError.append(key, val, err)
			} else {
				f.Set(reflect.ValueOf(l))
			}
		default:
			if err := json.Unmarshal([]byte(val), f.Addr().Interface()); err != nil {
				if f.Kind() == reflect.Slice {
					if f.Type().Elem().Kind() == reflect.String {
						ss := strings.Split(val, ",")
						for i := range ss {
							ss[i] = strings.TrimSpace(ss[i])
						}
						f.Set(reflect.ValueOf(ss))
					} else {
						if err2 := json.Unmarshal([]byte("["+val+"]"), f.Addr().Interface()); err2 != nil {
							parseError.append(key, val, err) // append first error
						}
					}
				} else {
					parseError.append(key, val, err)
				}
			}
		}
	}

	if len(parseError.Items) > 0 {
		return &parseError
	}

	return nil
}

func lookupEnvName(f reflect.StructField) string {
	if !f.IsExported() {
		return ""
	}

	key, ok := f.Tag.Lookup("env")
	if ok {
		return key
	}

	return f.Name
}

func valueOfPointerToStruct(target any) reflect.Value {
	var targetVal reflect.Value
	if v := reflect.ValueOf(target); v.Kind() == reflect.Ptr {
		targetVal = v.Elem()
	}
	if targetVal.Kind() != reflect.Struct {
		panic("envparser: target must be non-nil pointer to struct")
	}

	return targetVal
}

// List env names from target.
//
// target must be non-nil pointer to struct.
func ListEnvName(target any) []string {
	targetVal := valueOfPointerToStruct(target)

	var ret []string
	for i, t := 0, targetVal.Type(); i < t.NumField(); i++ {
		key := lookupEnvName(targetVal.Type().Field(i))
		if key == "" {
			continue
		}

		ret = append(ret, key)
	}

	return ret
}
