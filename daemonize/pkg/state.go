package pkg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"
)

const envStateKey = "AA5523AC76631703"

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
	err := os.WriteFile(s.supervisorPidPath(), []byte(fmt.Sprintf("%d %s\n", os.Getpid(), envstateGetID())), 0o600)
	check(err)
}

func (s svc) getSupervisorPid() int {
	data, err := os.ReadFile(s.supervisorPidPath())
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return 0
	}
	check(err)

	var pid int
	var stateID string
	fmt.Sscanf(string(data), "%d %s", &pid, &stateID)

	liveStateId, err := readEnvstateIDFromPid(pid)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return 0
	}
	check(err)

	if stateID != liveStateId {
		return 0
	}

	return pid
}

func readEnvstateIDFromPid(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
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
