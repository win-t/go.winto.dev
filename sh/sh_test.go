package sh

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestNormal(t *testing.T) {
	if Sh(`echo Hello World`) != "Hello World" {
		t.Fatal()
	}
}

func TestArgs(t *testing.T) {
	if Sh(`printf '%s+%s\n\n\n' $1 $2`, Args("Hello", "World")) != "Hello+World" {
		t.Fatal()
	}
}

func TestStdin(t *testing.T) {
	if Sh(`cat`, Stdin("Hello World")) != "Hello World" {
		t.Fatal()
	}
}

func TestDiscard(t *testing.T) {
	if Sh(`echo Hello World`, DiscardStdout(), DiscardStderr()) != "" {
		t.Fatal()
	}
}

func TestStderr(t *testing.T) {
	var stderr string
	Sh(`echo Hello World >&2`, StoreStderr(&stderr))
	if stderr != "Hello World\n" {
		t.Fatal()
	}
}

func TestTap(t *testing.T) {
	if Sh(`echo $ENV_INPUT`, TapExecCmd(func(c *exec.Cmd) { c.Env = append(c.Env, "ENV_INPUT=Hello World") })) != "Hello World" {
		t.Fatal()
	}
}

func TestExitCode(t *testing.T) {
	var exitCode int
	Sh(`kill $$ TERM`, StoreExitCode(&exitCode))
	if exitCode != 143 {
		t.Fatal()
	}
}

func TestStoreError(t *testing.T) {
	var errDst error
	Sh(`exec false`, StoreError(&errDst))

	var realErr *exec.ExitError
	if !errors.As(errDst, &realErr) {
		t.Fatal()
	}
	if realErr.ExitCode() != 1 {
		t.Fatal()
	}
}

func TestEscape(t *testing.T) {
	if Escape(`ab'cd`, `hello world`) != `'ab'\''cd' 'hello world'` {
		t.Fatal()
	}
}

func TestEnv(t *testing.T) {
	if Sh(`echo $ENV_INPUT`, EnvMap(map[string]string{"ENV_INPUT": "Hello World"})) != "Hello World" {
		t.Fatal()
	}
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond))
	defer cancel()

	var code int
	Sh(`sleep 10`, Context(ctx), StoreExitCode(&code), DiscardStderr(), DiscardStdout())

	if code != 137 {
		t.Fatal()
	}
}

func TestPanicOnErr(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal()
		}
	}()

	Sh(`exit 10`, PanicOnErr())
	t.Fatal()
}

func TestInitVar(t *testing.T) {
	out := Sh(`
		printf ">>%s-%s<<" "$MY_VAR" "$(sh -c 'echo "$MY_VAR"')"
	`, Var(map[string]string{
		"MY_VAR": "Hello",
	}))
	if out != ">>Hello-<<" {
		t.Fatal()
	}
}
