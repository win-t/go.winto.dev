// Package sh provides a wrapper for shell commands.
package sh

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type OptFn func(*execOpt)

var useStderrMarker string

// Sh executes "sh -c cmd"
func Sh(cmd string, opts ...OptFn) string {
	return doShell("sh", cmd, opts...)
}

// Bash executes "bash -c cmd"
func Bash(cmd string, opts ...OptFn) string {
	return doShell("bash", cmd, opts...)
}

// Dash executes "dash -c cmd"
func Dash(cmd string, opts ...OptFn) string {
	return doShell("dash", cmd, opts...)
}

func doShell(shell string, cmd string, opts ...OptFn) string {
	b := &execOpt{
		cmd:       cmd,
		shell:     shell,
		stderrDst: &useStderrMarker,
	}
	for _, opt := range opts {
		opt(b)
	}

	var outBuf, errBuf bytes.Buffer
	proc := exec.Command(b.shell, append([]string{"-c", b.cmd, "-"}, b.args...)...)
	if b.stdin != "" {
		proc.Stdin = strings.NewReader(b.stdin)
	}
	if !b.noStdout {
		proc.Stdout = &outBuf
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
	stdout := strings.TrimRight(outBuf.String(), "\r\n") // simulate shell command substitution behavior
	if err == nil {
		if b.stderrDst != nil && b.stderrDst != &useStderrMarker {
			*b.stderrDst = errBuf.String()
		}
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
