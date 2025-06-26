package sh

import (
	"context"
	"os/exec"
	"regexp"
)

type shellOpt struct {
	shell       string
	cmd         string
	args        []string
	stdin       string
	noStdout    bool
	stderrDst   *string
	envs        []string
	tapCmd      []func(*exec.Cmd)
	exitCodeDst *int
	errDst      *error
	ctx         context.Context
	panicOnErr  bool
	useStdout   bool
	initVars    map[string]string
}

func Args(args ...string) OptFn {
	return func(b *shellOpt) {
		b.args = args
	}
}

func Stdin(stdin string) OptFn {
	return func(b *shellOpt) {
		b.stdin = stdin
	}
}

func DiscardStdout() OptFn {
	return func(b *shellOpt) {
		b.noStdout = true
	}
}

func DiscardStderr() OptFn {
	return func(b *shellOpt) {
		b.stderrDst = nil
	}
}

func StoreStderr(dst *string) OptFn {
	return func(b *shellOpt) {
		b.stderrDst = dst
	}
}

func TapExecCmd(f func(*exec.Cmd)) OptFn {
	return func(b *shellOpt) {
		b.tapCmd = append(b.tapCmd, f)
	}
}

func StoreExitCode(dst *int) OptFn {
	return func(b *shellOpt) {
		b.exitCodeDst = dst
	}
}

func StoreError(dst *error) OptFn {
	return func(b *shellOpt) {
		b.errDst = dst
	}
}

func Env(envs ...string) OptFn {
	return func(b *shellOpt) {
		b.envs = append(b.envs, envs...)
	}
}

func EnvMap(env map[string]string) OptFn {
	var envs []string
	for k, v := range env {
		envs = append(envs, k+"="+v)
	}
	return Env(envs...)
}

func Context(ctx context.Context) OptFn {
	return func(b *shellOpt) {
		b.ctx = ctx
	}
}

func PanicOnErr() OptFn {
	return func(b *shellOpt) {
		b.panicOnErr = true
	}
}

func UseStdout() OptFn {
	return func(b *shellOpt) {
		b.useStdout = true
	}
}

var validShellVarNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Use Var instead of [Env] or [EnvMap] to set shell variables,
// this variable will not be inherited to env of child processes.
func Var(vars map[string]string) OptFn {
	return func(b *shellOpt) {
		if b.initVars == nil {
			b.initVars = make(map[string]string)
		}
		for k, v := range vars {
			if !validShellVarNameRegex.MatchString(k) {
				panic("invalid shell variable name: " + k)
			}
			b.initVars[k] = v
		}
	}
}
