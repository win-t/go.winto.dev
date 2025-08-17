// Package sh provides a wrapper for shell commands.
package sh

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type OptFn func(*shellOpt)

var useStderrMarker string

// Sh executes "sh -c cmd"
func Sh(cmd string, opts ...OptFn) string {
	return shell("sh", cmd, opts...)
}

// Bash executes "bash -c cmd"
func Bash(cmd string, opts ...OptFn) string {
	return shell("bash", cmd, opts...)
}

// Dash executes "dash -c cmd"
func Dash(cmd string, opts ...OptFn) string {
	return shell("dash", cmd, opts...)
}

func shell(shell string, cmd string, opts ...OptFn) string {
	b := &shellOpt{
		cmd:       cmd,
		shell:     shell,
		stderrDst: &useStderrMarker,
	}
	for _, opt := range opts {
		opt(b)
	}

	var fullCommand strings.Builder
	for k, v := range b.initVars {
		fullCommand.WriteString(k)
		fullCommand.WriteString("=")
		fullCommand.WriteString(Escape(v))
		fullCommand.WriteString(";")
	}
	fullCommand.WriteString(b.cmd)

	var outBuf, errBuf bytes.Buffer
	var proc *exec.Cmd
	args := append([]string{"-c", fullCommand.String(), "-"}, b.args...)
	if b.ctx != nil {
		proc = exec.CommandContext(b.ctx, b.shell, args...)
	} else {
		proc = exec.Command(b.shell, args...)
	}
	if b.envs != nil {
		proc.Env = append(os.Environ(), b.envs...)
	}
	if b.stdin != "" {
		proc.Stdin = strings.NewReader(b.stdin)
	}
	if b.stdinBytes != nil {
		proc.Stdin = bytes.NewReader(b.stdinBytes)
	}
	if !b.noStdout {
		if b.useStdout {
			proc.Stdout = os.Stdout
		} else {
			proc.Stdout = &outBuf
		}
	}
	if b.stderrDst != nil {
		if b.stderrDst == &useStderrMarker {
			proc.Stderr = os.Stderr
		} else {
			proc.Stderr = &errBuf
		}
	}
	for _, tap := range b.tapCmd {
		tap(proc)
	}

	err := proc.Run()
	var stdout string
	if b.stdoutDst == nil {
		stdout = strings.TrimRight(outBuf.String(), "\r\n") // simulate shell command substitution behavior
	} else {
		*b.stdoutDst = outBuf.Bytes()
	}
	if b.stderrDst != nil && b.stderrDst != &useStderrMarker {
		*b.stderrDst = errBuf.String()
	}
	if err == nil {
		return stdout
	}
	if b.errDst != nil {
		*b.errDst = err
	}

	if b.exitCodeDst != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code := ee.ExitCode()
			if !ee.Exited() {
				if status, ok := ee.Sys().(syscall.WaitStatus); ok {
					code = 128 + int(status.Signal())
				}
			}
			*b.exitCodeDst = code
		}
	}

	if b.panicOnErr {
		panic(err)
	}

	return stdout
}

// Escape escapes a string for use in shell commands, wrapping it in single quotes
func Escape(ss ...string) string {
	var sb strings.Builder
	for i, s := range ss {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(`'`)
		sb.WriteString(strings.ReplaceAll(s, `'`, `'\''`))
		sb.WriteString(`'`)
	}
	return sb.String()
}
