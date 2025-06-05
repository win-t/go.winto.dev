// Package sh provides a wrapper for shell commands.
package sh

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
)

func Sh(cmd string, opts ...func(*Builder)) *Builder {
	return newBuilder("sh", cmd, opts...)
}

func Bash(cmd string, opts ...func(*Builder)) *Builder {
	return newBuilder("bash", cmd, opts...)
}

func Dash(cmd string, opts ...func(*Builder)) *Builder {
	return newBuilder("dash", cmd, opts...)
}

func (b *Builder) Run() string {
	var outBuf, errBuf bytes.Buffer
	proc := exec.Command(b.shell, append([]string{"-c", b.cmd, "-"}, b.args...)...)
	if b.stdin != "" {
		proc.Stdin = strings.NewReader(b.stdin)
	}
	if !b.noStdout {
		proc.Stdout = &outBuf
	}
	if b.stderrDst != nil {
		proc.Stderr = &errBuf
	}
	for _, tap := range b.tapCmd {
		tap(proc)
	}

	err := proc.Run()
	if err == nil {
		if b.stderrDst != nil {
			*b.stderrDst = errBuf.String()
		}
		return outBuf.String()
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

	return outBuf.String()
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
