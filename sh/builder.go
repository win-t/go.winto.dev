package sh

import "os/exec"

var passStderrMarker string

type builder struct {
	shell       string
	cmd         string
	args        []string
	stdin       string
	noStdout    bool
	stderrDst   *string
	tapCmd      []func(*exec.Cmd)
	exitCodeDst *int
	errDst      *error
}

func newBuilder(shell string, cmd string, opts ...func(*builder)) *builder {
	b := &builder{
		cmd:       cmd,
		shell:     shell,
		stderrDst: &passStderrMarker,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func Args(args ...string) func(*builder) {
	return func(b *builder) {
		b.args = args
	}
}

func Stdin(stdin string) func(*builder) {
	return func(b *builder) {
		b.stdin = stdin
	}
}

func DiscardStdout(b *builder) {
	b.noStdout = true
}

func DiscardStderr(b *builder) {
	b.stderrDst = nil
}

func StoreStderr(dst *string) func(*builder) {
	return func(b *builder) {
		b.stderrDst = dst
	}
}

func Tap(f func(*exec.Cmd)) func(*builder) {
	return func(b *builder) {
		b.tapCmd = append(b.tapCmd, f)
	}
}

func StoreExitCode(dst *int) func(*builder) {
	return func(b *builder) {
		b.exitCodeDst = dst
	}
}

func StoreError(dst *error) func(*builder) {
	return func(b *builder) {
		b.errDst = dst
	}
}
