package pgtestutil

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
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

	containerName := "pgtestutil-" + randomHex()
	adminPass := "p" + randomHex()

	port, stopProxy, err := runProxy(containerName)
	if err != nil {
		return nil, err
	}

	exec.Command(
		"docker", "run",
		"-d", "--name", containerName,
		"--restart", "unless-stopped",
		"-l", "go.winto.dev/pgtestutil=true",
		"-e", "POSTGRES_PASSWORD="+adminPass,
		image,
	).Run()

	closeFn := func() {
		stopProxy()
		exec.Command("docker", "rm", "-fv", containerName).Run()
	}

	target := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("postgres", adminPass),
		Host:     fmt.Sprintf("localhost:%d", port),
		Path:     "postgres",
		RawQuery: "sslmode=disable",
	}

	until := time.Now().Add(30 * time.Second)
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

func runProxy(containerName string) (int, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())

	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		return 0, cancel, err
	}
	go func() {
		<-ctx.Done()
		l.Close()
	}()

	go func() {
		for {
			conn, err := l.AcceptTCP()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			go proxy(ctx, conn, containerName)
		}
	}()

	return l.Addr().(*net.TCPAddr).Port, cancel, nil
}

func proxy(ctx context.Context, conn *net.TCPConn, containerName string) {
	defer conn.Close()
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerName, "nc", "localhost", "5432")
	cmd.Stdin = conn
	cmd.Stdout = conn
	err := cmd.Run()
	if err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "error starting proxy: %v\n", err)
	}
}
