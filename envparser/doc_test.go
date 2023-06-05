package envparser_test

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.winto.dev/envparser"
)

type AddOne int

func (a *AddOne) UnmarshalEnv(e string) error {
	if err := json.Unmarshal([]byte(e), a); err != nil {
		return err
	}
	*a += 1
	return nil
}

type AddSlice []int

func (a AddSlice) UnmarshalEnv(e string) error {
	var v int
	if err := json.Unmarshal([]byte(e), &v); err != nil {
		return err
	}
	for i := range a {
		a[i] += v
	}
	return nil
}

func ExampleUnmarshal() {
	fakeEnv := map[string]string{
		"TestKey":     "test value",
		"TestKey2":    "12",
		"unexported":  "some text",
		"Composite":   `{"A": 11, "B": true}`,
		"ADD_ONE":     "22",
		"AddSlice":    "4",
		"Time":        "2021-09-14T12:13:14.123123+09:00",
		"StringSlice": "a, b, c",
		"IntSlice":    "1,2,3",
		"Dur":         "1m30s",
		"Loc":         "Asia/Jakarta",
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
		TestKey    string
		TestKey2   int
		TestKey3   bool
		unexported string
		Composite  struct {
			A int
			B bool
		}
		AddOne      AddOne `env:"ADD_ONE"`
		AddSlice    AddSlice
		Time        time.Time
		StringSlice []string
		IntSlice    []int
		Dur         time.Duration
		Loc         *time.Location
	}
	config.AddSlice = []int{1, 2, 3}
	config.unexported = "unexported"

	err := envparser.Unmarshal(&config)
	if err != nil {
		panic(err)
	}

	testresult := config.TestKey == "test value" &&
		config.TestKey2 == 12 &&
		config.TestKey3 == false &&
		config.unexported == "unexported" &&
		config.Composite.A == 11 &&
		config.Composite.B == true &&
		config.AddOne == 23 &&
		config.AddSlice[0] == 5 &&
		config.AddSlice[1] == 6 &&
		config.AddSlice[2] == 7 &&
		config.Time.UTC() == time.Date(2021, 9, 14, 3, 13, 14, 123123000, time.UTC) &&
		config.StringSlice[0] == "a" &&
		config.StringSlice[1] == "b" &&
		config.StringSlice[2] == "c" &&
		config.IntSlice[0] == 1 &&
		config.IntSlice[1] == 2 &&
		config.IntSlice[2] == 3 &&
		config.Dur == 1*time.Minute+30*time.Second &&
		config.Loc.String() == "Asia/Jakarta"

	fmt.Println(testresult)
	// Output: true
}
