package envparser

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
)

// Unmarshal into struct.
//
// target must be non-nil pointer to struct.
//
// "env" tag in each field in target struct will be fetched from environment variable.
// If "env" tag is empty string or field is not exported, the field is skipped.
// If "env" tag is not found, field name is used.
// If "env" tag has "nounset" option, the env will be kept, otherwise it will be unset.
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
		envConfig := lookupEnvConfig(targetVal.Type().Field(i))
		if envConfig.name == "" {
			continue
		}

		key := envConfig.name
		val, ok := os.LookupEnv(prefix + key)
		if !ok {
			continue
		}
		if !envConfig.noUnset {
			os.Unsetenv(prefix + key)
		}

		f := targetVal.Field(i)
		if f.Addr().Type().Implements(unmarshalerType) {
			if err := f.Addr().Interface().(Unmarshaler).UnmarshalEnv(val); err != nil {
				parseError.append(key, val, err)
			}
			continue
		}
		if f.Kind() == reflect.String {
			f.SetString(val)
			continue
		}
		fn, ok := nativeUnmarshaler[f.Type()]
		if ok {
			v, err := fn(val)
			if err != nil {
				parseError.append(key, val, err)
			} else {
				f.Set(reflect.ValueOf(v))
			}
			continue
		}
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

	if len(parseError.Items) > 0 {
		return &parseError
	}

	return nil
}

type envConfig struct {
	name    string
	noUnset bool
}

func lookupEnvConfig(f reflect.StructField) envConfig {
	if !f.IsExported() {
		return envConfig{}
	}

	config, ok := f.Tag.Lookup("env")
	if ok {
		configParts := strings.Split(config, ",")
		name := configParts[0]
		noUnset := false
		for _, c := range configParts[1:] {
			if c == "nounset" {
				noUnset = true
			} else if c != "" {
				panic("envparser: unknown tag option: " + c)
			}
		}
		return envConfig{name: name, noUnset: noUnset}
	}

	return envConfig{name: f.Name}
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
		envConfig := lookupEnvConfig(targetVal.Type().Field(i))
		if envConfig.name == "" {
			continue
		}

		ret = append(ret, envConfig.name)
	}

	return ret
}
