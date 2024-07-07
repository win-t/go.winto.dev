package pkg

import (
	"fmt"
	"os"
)

func readPidEnviron(pid int) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
}

func forkExec(attr *os.ProcAttr) (*os.Process, error) {
	return os.StartProcess("/proc/self/exe", os.Args, attr)
}
