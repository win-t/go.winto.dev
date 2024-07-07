package pkg

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"time"
)

func (s svc) doubleForkIsNeeded() bool {
	err := os.Mkdir(s.statePath(), 0o700)
	if err == nil {
		return true
	}
	if !errors.Is(err, fs.ErrExist) {
		panic(err)
	}
	if s.getSupervisorPid() != 0 {
		return false
	}
	_ = os.RemoveAll(s.statePath())
	err = os.Mkdir(s.statePath(), 0o700)
	check(err)
	return true
}

func (s svc) continuationOfDoubleFork() bool {
	id := envstateGetID()
	if id == "" {
		return false
	}

	s.doDoubleFork()
	return true
}

func (s svc) doDoubleFork() {
	switch envstateSetNext() {
	case 0:
		rInfo, wInfo, err := os.Pipe()
		check(err)

		p, err := os.StartProcess("/proc/self/exe", os.Args, &os.ProcAttr{
			Sys:   &syscall.SysProcAttr{Setsid: true},
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr, wInfo},
		})
		check(err)
		_, _ = p.Wait()

		var buf [8]byte
		_ = rInfo.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _ := rInfo.Read(buf[:])
		if slices.Compare(buf[:n], []byte("ok")) != 0 {
			stdout, _ := os.ReadFile(s.supervisorStdout())
			_, _ = os.Stdout.Write(stdout)
			stderr, _ := os.ReadFile(s.supervisorStderr())
			_, _ = os.Stderr.Write(stderr)
			_, _ = fmt.Fprintln(os.Stderr, "Failed to start daemonize process")
			os.Exit(1)
		}

	case 1:
		stdin, err := os.Open("/dev/null")
		check(err)
		stdout, err := os.OpenFile(s.supervisorStdout(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		check(err)
		stderr, err := os.OpenFile(s.supervisorStderr(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		check(err)

		_, err = os.StartProcess("/proc/self/exe", os.Args, &os.ProcAttr{
			Sys:   &syscall.SysProcAttr{Setpgid: true},
			Files: []*os.File{stdin, stdout, stderr, os.NewFile(3, "info")},
		})
		check(err)

		os.Exit(0)

	case 2:
		s.writePidFile()
		envstateClear()

		wInfo := os.NewFile(3, "info")
		_, err := wInfo.Write([]byte("ok"))
		check(err)

		err = wInfo.Close()
		check(err)

	default:
		panic("should not happen")
	}
}

func (s svc) startMainLoop() {
	// set GOMAXPROCS to 3 for:
	// - mainloop
	// - stdout forwarder
	// - stderr forwarder
	runtime.GOMAXPROCS(3)

	printf(
		"[%s] daemonize started for service dir '%s'\n",
		time.Now().Format(time.RFC3339),
		string(s),
	)
	defer func() {
		printf(
			"[%s] daemonize exited\n",
			time.Now().Format(time.RFC3339),
		)
	}()

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	stdout, stderr := s.setupForwarder(ctx, &wg)
	defer func() { _ = stdout.Close(); _ = stderr.Close() }()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGUSR1)

	for {
		cmd := exec.Command(s.runPath())
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, stdout, stderr

		printf(
			"[%s] starting the process\n",
			time.Now().Format(time.RFC3339),
		)

		waitErrCh := make(chan error, 1)
		go func() { waitErrCh <- cmd.Run() }()

		beforeWait := time.Now()
		select {
		case err := <-waitErrCh:
			if err == nil {
				printf(
					"[%s] the process exited successfully\n",
					time.Now().Format(time.RFC3339),
				)
				return
			}

			sleepDur := max(100*time.Millisecond, (15*time.Second)-time.Since(beforeWait))
			if realErr := (*exec.ExitError)(nil); errors.As(err, &realErr) {
				status := realErr.Sys().(syscall.WaitStatus)
				if status.Signaled() {
					printf(
						"[%s] the process killed by signal %d (%s), restarting in %s\n",
						time.Now().Format(time.RFC3339),
						int(status.Signal()),
						status.Signal().String(),
						sleepDur.Round(time.Millisecond).String(),
					)
				} else {
					printf(
						"[%s] the process exited with status code %d, restarting in %s\n",
						time.Now().Format(time.RFC3339),
						status.ExitStatus(),
						sleepDur.Round(time.Millisecond).String(),
					)
				}
			} else {
				printf(
					"[%s] failed to execute '%s' (%s), restarting in %s\n",
					time.Now().Format(time.RFC3339),
					s.runPath(),
					err.Error(),
					sleepDur.Round(time.Millisecond).String(),
				)
			}

			select {
			case <-time.After(sleepDur):
			case sig := <-sigCh:
				switch sig {
				case syscall.SIGTERM:
					printf(
						"[%s] exit is requested while in restart back-off\n",
						time.Now().Format(time.RFC3339),
					)
					return
				case syscall.SIGUSR1:
					printf(
						"[%s] restart is requested while in restart back-off\n",
						time.Now().Format(time.RFC3339),
					)
				default:
					panic("should not happen")
				}
			}

		case sig := <-sigCh:
			switch sig {
			case syscall.SIGTERM:
				printf(
					"[%s] exit is requested, send termination signal to the process\n",
					time.Now().Format(time.RFC3339),
				)
			case syscall.SIGUSR1:
				printf(
					"[%s] restart is requested, send termination signal to the process\n",
					time.Now().Format(time.RFC3339),
				)
			default:
				panic("should not happen")
			}

			err := cmd.Process.Signal(syscall.SIGTERM)
			check(err)

			select {
			case <-waitErrCh:
				printf(
					"[%s] process exited within 15 seconds\n",
					time.Now().Format(time.RFC3339),
				)
			case <-time.After(15 * time.Second):
				printf(
					"[%s] the process is not exited within 15 seconds, force kill it\n",
					time.Now().Format(time.RFC3339),
				)
				err = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				check(err)
				<-waitErrCh
			}

			if sig == syscall.SIGTERM {
				return
			}
		}
	}
}
