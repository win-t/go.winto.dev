package sh

import "os/exec"

type Builder struct {
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

func newBuilder(shell string, cmd string, opts ...func(*Builder)) *Builder {
	b := &Builder{
		cmd:   cmd,
		shell: shell,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func Args(args ...string) func(*Builder) {
	return func(b *Builder) {
		b.args = args
	}
}

func Stdin(stdin string) func(*Builder) {
	return func(b *Builder) {
		b.stdin = stdin
	}
}

func DiscardStdout(b *Builder) {
	b.noStdout = true
}

func StoreStderr(dst *string) func(*Builder) {
	return func(b *Builder) {
		b.stderrDst = dst
	}
}

func Tap(f func(*exec.Cmd)) func(*Builder) {
	return func(b *Builder) {
		b.tapCmd = append(b.tapCmd, f)
	}
}

func StoreExitCode(dst *int) func(*Builder) {
	return func(b *Builder) {
		b.exitCodeDst = dst
	}
}

func StoreError(dst *error) func(*Builder) {
	return func(b *Builder) {
		b.errDst = dst
	}
}
