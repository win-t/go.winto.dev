// Package sh provides a wrapper for shell commands.
package sh

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
)

// Executes "sh -c ..." and returns its stdout, stderr, and exit code.
func Sh(cmd string, args ...string) (stdout, stderr string, code int) {
	return shell("sh", cmd, args...)
}

// Executes "bash -c ..." and returns its stdout, stderr, and exit code.
func Bash(cmd string, args ...string) (stdout, stderr string, code int) {
	return shell("bash", cmd, args...)
}

// Executes "dash -c ..." and returns its stdout, stderr, and exit code.
func Dash(cmd string, args ...string) (stdout, stderr string, code int) {
	return shell("dash", cmd, args...)
}

func shell(shell, cmd string, args ...string) (string, string, int) {
	var outBuf, errBuf bytes.Buffer
	proc := exec.Command(shell, append([]string{"-c", cmd, "-"}, args...)...)
	proc.Stdout = &outBuf
	proc.Stderr = &errBuf

	err := proc.Run()
	if err == nil {
		return outBuf.String(), errBuf.String(), 0
	}

	if ee, ok := err.(*exec.ExitError); ok {
		code := ee.ExitCode()
		if !ee.Exited() {
			if status, ok := ee.Sys().(syscall.WaitStatus); ok {
				code = 128 + int(status.Signal())
			}
		}
		return outBuf.String(), errBuf.String(), code
	}

	return "", "cannot execute shell: " + err.Error(), -1
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
