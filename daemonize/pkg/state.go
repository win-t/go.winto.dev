package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"
)

const envStateKey = "DAEMONIZE_STATE"

func envstateGetID() string {
	state := os.Getenv(envStateKey)
	if state == "" {
		return ""
	}
	stateSlice := strings.Split(state, "/")
	return stateSlice[0]
}

func envstateClear() {
	os.Unsetenv(envStateKey)
}

func envstateSetNext() int {
	state := os.Getenv(envStateKey)
	if state == "" {
		err := os.Setenv(envStateKey, fmt.Sprintf("%d/1", time.Now().Unix()))
		check(err)
		return 0
	}
	stateSlice := strings.Split(state, "/")
	phase, err := strconv.Atoi(stateSlice[1])
	check(err)
	err = os.Setenv(envStateKey, fmt.Sprintf("%s/%d", stateSlice[0], phase+1))
	check(err)
	return phase
}

func (s svc) writePidFile() {
	f, err := os.OpenFile(s.supervisorPidPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	check(err)
	defer f.Close()
	_, err = fmt.Fprintf(f, "%d %s\n", os.Getpid(), envstateGetID())
	check(err)
}

func (s svc) getSupervisorPid() int {
	pid, _ := s.getSupervisorPidState()
	return pid
}

func (s svc) getSupervisorPidState() (int, bool) {
	data, err := os.ReadFile(s.supervisorPidPath())
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return 0, false
	}
	check(err)

	if bytes.Count(data, []byte("\n")) == 0 {
		return 0, false
	}

	var pid int
	var stateID string
	_, err = fmt.Sscanf(string(data), "%d %s", &pid, &stateID)
	check(err)

	liveStateId, err := readEnvstateIDFromPid(pid)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return 0, true
	}
	check(err)

	if stateID != liveStateId {
		return 0, true
	}

	return pid, true
}

func readEnvstateIDFromPid(pid int) (string, error) {
	data, err := readPidEnviron(pid)
	if err != nil {
		return "", err
	}
	for _, env := range strings.Split(string(data), "\x00") {
		envSlice := strings.Split(env, "=")
		if envSlice[0] == envStateKey {
			return strings.Split(envSlice[1], "/")[0], nil
		}
	}
	return "", nil
}
