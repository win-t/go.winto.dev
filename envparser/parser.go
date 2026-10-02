package envparser

import (
	"encoding"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

// Unmarshal into struct.
//
// target must be non-nil pointer to struct.
//
// "env" tag in each field in target struct will be fetched from environment variable.
// If "env" tag is not specified or specified but no name specified, field name will be used as env name.
// If "env" tag has "nounset" option, the env will be kept, otherwise it will be unset.
// If "env" tag has "skip" option, the field will be skipped.
// If "env" tag has "required" option, it will error if the env is not set.
// If "env" tag has "usage:" prefix, it must be the last, and it will be used for [RegisterFlagSet]
//
// if the field implement [Unmarshaler] interface, it will be used.
func Unmarshal(target any) error {
	return UnmarshalWithPrefix(target, "")
}

// Register each field of the struct to fset.
func RegisterFlagSet(target any, fset *flag.FlagSet) {
	for _, f := range getAll(target) {
		name := f.config.name
		value := f.Get()
		usage := f.config.usage
		switch value := value.(type) {
		case bool:
			fset.BoolVar(f.value.Addr().Interface().(*bool), name, value, usage)
		case time.Duration:
			fset.DurationVar(f.value.Addr().Interface().(*time.Duration), name, value, usage)
		case float64:
			fset.Float64Var(f.value.Addr().Interface().(*float64), name, value, usage)
		case int64:
			fset.Int64Var(f.value.Addr().Interface().(*int64), name, value, usage)
		case int:
			fset.IntVar(f.value.Addr().Interface().(*int), name, value, usage)
		case string:
			fset.StringVar(f.value.Addr().Interface().(*string), name, value, usage)
		case uint64:
			fset.Uint64Var(f.value.Addr().Interface().(*uint64), name, value, usage)
		case uint:
			fset.UintVar(f.value.Addr().Interface().(*uint), name, value, usage)
		default:
			fset.Var(f, name, usage)
		}
	}
}

type flagVal struct {
	config envConfig
	value  reflect.Value
}

func (f *flagVal) Get() any {
	return f.value.Interface()
}

func (f *flagVal) Set(val string) error {
	return setValue(f.value, val)
}

func (f *flagVal) String() string {
	return fmt.Sprint(f.value)
}

func getAll(target any) []*flagVal {
	targetVal := valueOfPointerToStruct(target)

	names := make(map[string]struct{})

	var ret []*flagVal
	for i, t := 0, targetVal.Type(); i < t.NumField(); i++ {
		envConfig := lookupEnvConfig(targetVal.Type().Field(i))
		if envConfig.skip {
			continue
		}

		if _, ok := names[envConfig.name]; ok {
			panic("envparser: found duplicate name in struct tag")
		}
		names[envConfig.name] = struct{}{}

		ret = append(ret, &flagVal{
			config: envConfig,
			value:  targetVal.Field(i),
		})
	}

	return ret
}

// Like [Unmarshal] but we can specify the prefix key.
func UnmarshalWithPrefix(target any, prefix string) error {
	var parseError ParseError

	for _, f := range getAll(target) {
		key := prefix + f.config.name
		val, ok := os.LookupEnv(key)
		if !ok {
			if f.config.required {
				parseError.append(key, "", ErrCauseRequired)
			}
			continue
		}
		if !f.config.noUnset {
			os.Unsetenv(key)
		}

		if err := setValue(f.value, val); err != nil {
			parseError.append(key, val, err)
		}
	}

	if len(parseError.Items) > 0 {
		return &parseError
	}

	return nil
}

// List env names from target.
//
// target must be non-nil pointer to struct.
func ListEnvName(target any) []string {
	var ret []string
	for _, v := range getAll(target) {
		ret = append(ret, v.config.name)
	}
	return ret
}

var (
	unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	textType        = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	binaryType      = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()
)

func setValueIfImplemented(f reflect.Value, val string) (bool, error) {
	if f.Type().Implements(unmarshalerType) {
		return true, f.Interface().(Unmarshaler).UnmarshalEnv(val)
	}
	if f.Type().Implements(textType) {
		return true, f.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(val))
	}
	if f.Type().Implements(binaryType) {
		return true, f.Interface().(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte(val))
	}
	return false, nil
}

func setValue(f reflect.Value, val string) error {
	if fn, ok := nativeUnmarshaler[f.Type()]; ok {
		v, err := fn(val)
		if err != nil {
			return err
		}
		f.Set(reflect.ValueOf(v))
		return nil
	}
	if ok, err := setValueIfImplemented(f.Addr(), val); ok {
		return err
	}
	if f.Kind() == reflect.Pointer && f.IsNil() {
		new := reflect.New(f.Type().Elem())
		if ok, err := setValueIfImplemented(new, val); ok {
			if err == nil {
				f.Set(new)
			}
			return err
		}
	}
	if f.Kind() == reflect.String {
		f.SetString(val)
		return nil
	}
	err := json.Unmarshal([]byte(val), f.Addr().Interface())
	if err == nil {
		return nil
	}
	if f.Kind() == reflect.Slice {
		if f.Type().Elem().Kind() == reflect.String {
			ss := strings.Split(val, ",")
			for i := range ss {
				ss[i] = strings.TrimSpace(ss[i])
			}
			f.Set(reflect.ValueOf(ss))
			return nil
		}
		if json.Unmarshal([]byte("["+val+"]"), f.Addr().Interface()) == nil {
			return nil
		}
	}
	return err
}

type envConfig struct {
	name     string
	noUnset  bool
	skip     bool
	required bool
	usage    string
}

func lookupEnvConfig(f reflect.StructField) (c envConfig) {
	if !f.IsExported() {
		return envConfig{skip: true}
	}

	config, ok := f.Tag.Lookup("env")
	if !ok {
		c.name = f.Name
		return c
	}
	configParts := strings.Split(config, ",")
	c.name = configParts[0]
	if c.name == "" {
		c.name = f.Name
	}

	for _, opt := range configParts[1:] {
		if opt == "nounset" {
			c.noUnset = true
		} else if opt == "skip" {
			c.skip = true
		} else if opt == "required" {
			c.required = true
		} else if strings.HasPrefix(opt, "usage:") {
			break
		} else if opt != "" {
			panic("envparser: unknown tag option: " + opt)
		}
	}
	if idx := strings.Index(config, "usage:"); idx != -1 {
		c.usage = config[len("usage:")+idx:]
	}
	return c
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
