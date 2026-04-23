package pgtestutil

import (
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

func NewDocker(pgMajorVersion int) (*Manager, error) {
	if !DockerAvailable() {
		return nil, fmt.Errorf("docker is not available")
	}

	image := "postgres:alpine"
	if pgMajorVersion != 0 {
		image = fmt.Sprintf("postgres:%d-alpine", pgMajorVersion)
	}

	containerName := "pgtestutil" + randomHex()
	adminPass := "p" + randomHex()

	exec.Command(
		"docker", "run",
		"-d", "--name", containerName,
		"--restart", "unless-stopped",
		"-l", "go.winto.dev/pgtestutil=true",
		"-p", "5432",
		"-e", "POSTGRES_PASSWORD="+adminPass,
		image,
	).Run()

	closeFn := func() { exec.Command("docker", "rm", "-fv", containerName).Run() }

	until := time.Now().Add(300 * time.Second)

	var endpoint string
	for {
		out, err := exec.Command(
			"docker", "inspect", containerName,
			"-f", `{{ with (index (index .NetworkSettings.Ports "5432/tcp") 0) }}{{ .HostIp }}#{{ .HostPort }}{{ end }}`,
		).Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), "#")
			if parts[0] == "0.0.0.0" || parts[0] == "::" {
				parts[0] = "localhost"
			}
			endpoint = net.JoinHostPort(parts[0], parts[1])
			break
		}
		if time.Now().After(until) {
			closeFn()
			return nil, fmt.Errorf("failed to inspect docker container port mapping: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	target := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("postgres", adminPass),
		Host:     endpoint,
		Path:     "postgres",
		RawQuery: "sslmode=disable",
	}

	for {
		out, _ := exec.Command("docker", "exec", containerName, "sh", "-c", "pg_isready >/dev/null 2>&1 && printf ready").Output()
		if string(out) == "ready" {
			break
		}
		if time.Now().After(until) {
			closeFn()
			return nil, fmt.Errorf("failed to wait until postgres is ready")
		}
		time.Sleep(1 * time.Second)
	}

	return newManager(target.String(), closeFn, true)
}

func DockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}
