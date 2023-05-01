//go:build linux || darwin
// +build linux darwin

package mainrun

import (
	"os"
	"syscall"
)

func getInterruptSigs() []os.Signal {
	return []os.Signal{syscall.SIGTERM, syscall.SIGINT}
}
