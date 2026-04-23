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

func NewDocker(driverName string, pgMajorVersion int) (*Manager, error) {
	if !DockerAvailable() {
		return nil, fmt.Errorf("docker is not available")
	}

	image := "postgres:alpine"
	if pgMajorVersion != 0 {
		image = fmt.Sprintf("postgres:%d-alpine", pgMajorVersion)
	}

	containerName := "pgtestutil-" + randomHex()
	adminPass := "p" + randomHex()

	ctx, cancel := context.WithCancel(context.Background())

	exec.Command(
		"docker", "run",
		"-d", "--name", containerName,
		"-l", "go.winto.dev/pgtestutil=true",
		"-e", "POSTGRES_PASSWORD="+adminPass,
		image,
	).Run()

	closeFn := func() {
		cancel()
		exec.Command("docker", "rm", "-fv", containerName).Run()
	}

	until := time.Now().Add(30 * time.Second)
	for {
		out, _ := exec.Command("docker", "exec", containerName, "sh", "-ceu", `
			if ! command -v socat > /dev/null; then
				apk add -U socat > /dev/null
			fi
			if pg_isready -U postgres > /dev/null; then
				printf "ready"
			fi
		`).Output()
		if string(out) == "ready" {
			break
		}
		if time.Now().After(until) {
			closeFn()
			return nil, fmt.Errorf("failed to wait until postgres is ready")
		}
		time.Sleep(1 * time.Second)
	}

	port, err := proxyServer(ctx, containerName)
	if err != nil {
		closeFn()
		return nil, err
	}

	target := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("postgres", adminPass),
		Host:     fmt.Sprintf("localhost:%d", port),
		Path:     "postgres",
		RawQuery: "sslmode=disable",
	}

	return newManager(driverName, target.String(), closeFn, true)
}

func DockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}

func proxyServer(ctx context.Context, containerName string) (int, error) {
	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		return 0, err
	}
	go func() { <-ctx.Done(); l.Close() }()

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

	return l.Addr().(*net.TCPAddr).Port, nil
}

func proxy(ctx context.Context, conn *net.TCPConn, containerName string) {
	connClosed := false
	defer func() {
		if !connClosed {
			conn.Close()
		}
	}()

	f, err := conn.File()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgtestutil: failed to get unerlying tcp conn file: %v\n", err)
		return
	}
	fClosed := false
	defer func() {
		if !fClosed {
			f.Close()
		}
	}()

	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerName, "socat", "-", "TCP:127.0.0.1:5432")
	cmd.Stdin = f
	cmd.Stdout = f
	err = cmd.Start()
	if err != nil {
		if ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "pgtestutil: error starting proxy: %v\n", err)
		}
		return
	}

	conn.Close()
	connClosed = true

	f.Close()
	fClosed = true

	err = cmd.Wait()
	if err != nil {
		if ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "pgtestutil: error waiting proxy: %v\n", err)
		}
	}
}
