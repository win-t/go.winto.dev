package envparser_test

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"go.winto.dev/envparser"
)

func TestInvalidTarget(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.FailNow()
		}
	}()

	var config map[string]string

	envparser.Unmarshal(&config)
}

func TestError(t *testing.T) {
	fakeEnv := map[string]string{
		"TestKey2":   "aa",
		"ADD_ONE":    "aa",
		"SliceAdder": "aa",
		"Time":       "aa",
		"IntSlice":   "1,aa,3",
		"Dur":        "aa",
		"Loc":        "Asia/Somewhere",
	}

	for k, v := range fakeEnv {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range fakeEnv {
			os.Unsetenv(k)
		}
	}()

	var config struct {
		TestKey2   int
		AddOne     AddOne `env:"ADD_ONE"`
		SliceAdder AddSlice
		Time       time.Time
		IntSlice   []int
		Dur        time.Duration
		Loc        *time.Location
	}
	config.TestKey2 = 22
	config.AddOne = 44
	config.SliceAdder = []int{1, 2, 3}
	config.Dur = 3 * time.Minute

	err := envparser.Unmarshal(&config)
	if err == nil || err.Error() == "" {
		t.FailNow()
	}

	var parseError *envparser.ParseError
	if !errors.As(err, &parseError) {
		t.FailNow()
	}

	if len(parseError.Items) != len(fakeEnv) {
		for _, v := range parseError.Items {
			if _, ok := fakeEnv[v.Key]; !ok {
				t.FailNow()
			}
		}
	}

	defTime := time.Time{}

	if config.TestKey2 != 22 ||
		config.AddOne != 44 ||
		config.SliceAdder[0] != 1 ||
		config.SliceAdder[1] != 2 ||
		config.SliceAdder[2] != 3 ||
		config.Time != defTime ||
		len(config.IntSlice) != 0 ||
		config.Dur != 3*time.Minute ||
		config.Loc != nil {
		t.FailNow()
	}
}

func TestListEnvName(t *testing.T) {
	var config struct {
		TestKey2   int
		AddOne     AddOne `env:"ADD_ONE"`
		SliceAdder AddSlice
		Time       time.Time
		IntSlice   []int
		Dur        time.Duration
		Loc        *time.Location
		Skip       string `env:""`
	}
	names := envparser.ListEnvName(&config)
	if !reflect.DeepEqual(names, []string{
		"TestKey2",
		"ADD_ONE",
		"SliceAdder",
		"Time",
		"IntSlice",
		"Dur",
		"Loc",
	}) {
		t.Fatalf("invalid ListEnvName")
	}
}

func TestPrefix(t *testing.T) {
	fakeEnv := map[string]string{
		"PREFIX_A": "42",
		"PREFIX_B": "hello world",
	}
	for k, v := range fakeEnv {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range fakeEnv {
			os.Unsetenv(k)
		}
	}()

	var config struct {
		A int
		B string
	}

	err := envparser.UnmarshalWithPrefix(&config, "PREFIX_")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	if config.A != 42 || config.B != "hello world" {
		t.FailNow()
	}
}
