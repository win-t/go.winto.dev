//go:build !(linux || darwin)

package gorundir

import (
	"os/exec"
)

func ensureGo(string) {
	if _, err := exec.LookPath("go"); err != nil {
		exitErr("gorundir: Please install Go")
	}
}
