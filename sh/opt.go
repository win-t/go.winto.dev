package sh

import "os/exec"

type execOpt struct {
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

func Args(args ...string) OptFn {
	return func(b *execOpt) {
		b.args = args
	}
}

func Stdin(stdin string) OptFn {
	return func(b *execOpt) {
		b.stdin = stdin
	}
}

func DiscardStdout() OptFn {
	return func(b *execOpt) {
		b.noStdout = true
	}
}

func DiscardStderr() OptFn {
	return func(b *execOpt) {
		b.stderrDst = nil
	}
}

func StoreStderr(dst *string) OptFn {
	return func(b *execOpt) {
		b.stderrDst = dst
	}
}

func Tap(f func(*exec.Cmd)) OptFn {
	return func(b *execOpt) {
		b.tapCmd = append(b.tapCmd, f)
	}
}

func StoreExitCode(dst *int) OptFn {
	return func(b *execOpt) {
		b.exitCodeDst = dst
	}
}

func StoreError(dst *error) OptFn {
	return func(b *execOpt) {
		b.errDst = dst
	}
}
