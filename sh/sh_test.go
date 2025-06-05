package sh

import (
	"errors"
	"os/exec"
	"testing"
)

func TestNormal(t *testing.T) {
	if Sh("echo Hello World") != "Hello World\n" {
		t.Fatal()
	}
}

func TestArgs(t *testing.T) {
	if Sh("printf '%s+%s' $1 $2", Args("Hello", "World")) != "Hello+World" {
		t.Fatal()
	}
}

func TestStdin(t *testing.T) {
	if Sh("cat", Stdin("Hello World")) != "Hello World" {
		t.Fatal()
	}
}

func TestDiscard(t *testing.T) {
	if Sh("echo Hello World", DiscardStdout, DiscardStderr) != "" {
		t.Fatal()
	}
}

func TestStderr(t *testing.T) {
	var stderr string
	Sh("echo Hello World >&2", StoreStderr(&stderr))
	if stderr != "Hello World\n" {
		t.Fatal()
	}
}

func TestTap(t *testing.T) {
	if Sh("echo $ENV_INPUT", Tap(func(c *exec.Cmd) { c.Env = append(c.Env, "ENV_INPUT=Hello World") })) != "Hello World\n" {
		t.Fatal()
	}
}

func TestExitCode(t *testing.T) {
	var exitCode int
	Sh("kill $$ TERM", StoreExitCode(&exitCode))
	if exitCode != 143 {
		t.Fatal()
	}
}

func TestStoreError(t *testing.T) {
	var errDst error
	Sh("exec false", StoreError(&errDst))

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
