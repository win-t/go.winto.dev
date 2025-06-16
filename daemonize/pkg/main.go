package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

type svc string

var cmdFn = map[string]func(svc){
	"reopen":  svc.cmdReopen,
	"restart": svc.cmdRestart,
	"start":   svc.cmdStart,
	"status":  svc.cmdStatus,
	"stop":    svc.cmdStop,
	"pid":     svc.cmdPid,
}

func Main() {
	runtime.GOMAXPROCS(1)

	if len(os.Args) != 3 {
		printUsageAndExit()
	}

	root, err := filepath.Abs(os.Args[1])
	check(err)
	svc := svc(root)

	if svc.continuationOfDoubleFork() {
		svc.startMainLoop()
		return
	}

	cmd := cmdFn[os.Args[2]]
	if cmd == nil {
		printUsageAndExit()
	}
	cmd(svc)
}

func (s svc) cmdReopen() {
	pid := s.getSupervisorPid()
	if pid == 0 {
		return
	}

	err := syscall.Kill(pid, syscall.SIGUSR2)
	check(err)
}

func (s svc) cmdRestart() {
	pid := s.getSupervisorPid()
	if pid == 0 {
		fmt.Printf("not running, it will do start instead")
		s.cmdStart()
		return
	}

	err := syscall.Kill(pid, syscall.SIGUSR1)
	check(err)
}

func (s svc) cmdStart() {
	if s.doubleForkIsNeeded() {
		s.doDoubleFork()
	}
}

func (s svc) cmdStatus() {
	pid := s.getSupervisorPid()
	if pid == 0 {
		fmt.Printf("stopped\n")
		return
	}

	fmt.Printf("running\n")
}

func (s svc) cmdStop() {
	pid := s.getSupervisorPid()
	if pid == 0 {
		return
	}

	err := syscall.Kill(pid, syscall.SIGTERM)
	check(err)
}

func (s svc) cmdPid() {
	pid := s.getSupervisorPid()
	if pid == 0 {
		return
	}

	fmt.Printf("%d\n", pid)
}
