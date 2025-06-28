package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func printUsageAndExit() {
	var cmds []string
	for k := range cmdFn {
		cmds = append(cmds, k)
	}
	sort.Strings(cmds)

	fmt.Fprintf(os.Stderr, "usage: %s <service path> (%s)\n", os.Args[0], strings.Join(cmds, "|"))

	os.Exit(1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func (s svc) statePath() string {
	return filepath.Join(string(s), ".daemonize.state")
}

func (s svc) supervisorPidPath() string {
	return filepath.Join(s.statePath(), "pid")
}

func (s svc) supervisorLog() string {
	return filepath.Join(s.statePath(), "log")
}

func (s svc) runPath() string {
	return filepath.Join(string(s), "run")
}

func (s svc) runLogPath() string {
	return filepath.Join(string(s), "log")
}

func (s svc) runLogStdoutPath() string {
	return filepath.Join(s.runLogPath(), "out")
}

func (s svc) runLogStderrPath() string {
	return filepath.Join(s.runLogPath(), "err")
}
