package sh

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestNormal(t *testing.T) {
	out := Sh("echo Hello World").Run()
	if out != "Hello World\n" {
		t.Errorf("Expected 'Hello World', got '%s'", out)
	}
}

func TestArgs(t *testing.T) {
	out := Sh("printf '%s+%s' $1 $2", Args("Hello", "World")).Run()
	if out != "Hello+World" {
		t.Errorf("Expected 'Hello World', got '%s'", out)
	}
}

func TestStdin(t *testing.T) {
	out := Sh("cat", Stdin("Hello World")).Run()
	if out != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", out)
	}
}

func TestDiscardStdout(t *testing.T) {
	out := Sh("echo Hello World", DiscardStdout).Run()
	if out != "" {
		t.Errorf("Expected '', got '%s'", out)
	}
}

func TestStderr(t *testing.T) {
	var stderr string
	Sh("echo Hello World >&2", StoreStderr(&stderr)).Run()
	if stderr != "Hello World\n" {
		t.Errorf("Expected 'Hello World', got '%s'", stderr)
	}
}

func TestTap(t *testing.T) {
	out := Sh("cat", Tap(func(c *exec.Cmd) {
		c.Stdin = strings.NewReader("Hello World")
	})).Run()
	if out != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", out)
	}
}

func TestExitCode(t *testing.T) {
	var exitCode int
	Sh("kill $$ TERM", StoreExitCode(&exitCode)).Run()
	if exitCode != 143 {
		t.Errorf("Expected exit code 143, got %d", exitCode)
	}
}

func TestStoreError(t *testing.T) {
	var errDst error
	Sh("exec false", StoreError(&errDst)).Run()
	var realErr *exec.ExitError
	if !errors.As(errDst, &realErr) {
		t.Error("Expected an error, got nil")
	}
	if realErr.ExitCode() != 1 {
		t.Errorf("Expected exit code 1, got %d", realErr.ExitCode())
	}
}

func TestEscape(t *testing.T) {
	if Escape(`ab'cd`, `hello world`) != `'ab'\''cd' 'hello world'` {
		t.Error("Escape function did not return expected result")
	}
}
