package envparser_test

import (
	"errors"
	"net/url"
	"os"
	"testing"

	"go.winto.dev/envparser"
)

func TestParserTypes(t *testing.T) {
	fakeEnv := map[string]string{
		"b64":       "YXNkZg",
		"FileBytes": "testdata/test.txt",
		"b64ofjson": "eyJoZWxsbyI6IndvcmxkIn0K",
		"testURL":   "https://google.com",
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
		B64       envparser.Base64 `env:"b64"`
		FileBytes envparser.File
		B64OfJson envparser.Base64OfJSON[struct {
			Hello string `json:"hello"`
		}] `env:"b64ofjson"`
		TestURL *url.URL `env:"testURL"`
	}

	err := envparser.Unmarshal(&config)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	if string(config.B64) != "asdf" ||
		string(config.FileBytes) != "hello\n" {
		t.FailNow()
	}

	if config.B64OfJson.Value.Hello != "world" {
		t.FailNow()
	}

	if config.TestURL == nil || config.TestURL.Scheme != "https" || config.TestURL.Host != "google.com" {
		t.FailNow()
	}
}

func TestTypesError(t *testing.T) {
	fakeEnv := map[string]string{
		"b64":       "a",
		"FileBytes": "testdata/nonexisted",
		"b64ofjson": "e",
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
		B64       envparser.Base64 `env:"b64"`
		FileBytes envparser.File
		B64OfJson envparser.Base64OfJSON[struct {
			Hello string `json:"hello"`
		}] `env:"b64ofjson"`
	}

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

	if string(config.B64) != "" ||
		string(config.FileBytes) != "" {
		t.FailNow()
	}

	if config.B64OfJson.Value.Hello != "" {
		t.FailNow()
	}
}
