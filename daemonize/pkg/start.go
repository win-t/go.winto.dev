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
	os.RemoveAll(s.statePath())
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
		p.Wait()

		var buf [8]byte
		rInfo.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _ := rInfo.Read(buf[:])
		if slices.Compare(buf[:n], []byte("ok")) != 0 {
			stdout, _ := os.ReadFile(s.supervisorStdout())
			os.Stdout.Write(stdout)
			stderr, _ := os.ReadFile(s.supervisorStderr())
			os.Stderr.Write(stderr)
			fmt.Fprintln(os.Stderr, "Failed to start daemonize process")
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
	runtime.GOMAXPROCS(2)

	defer func() {
		printf(
			"[%s] the daemonize exited\n",
			time.Now().Format(time.RFC3339),
		)
	}()

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	stdout, stderr := s.setupForwarder(ctx, &wg)
	defer func() { stdout.Close(); stderr.Close() }()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGUSR1)

	for {
		cmd := exec.Command(s.runPath())
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, stdout, stderr

		waitErrCh := make(chan error, 1)
		go func() { waitErrCh <- cmd.Run() }()

		startedOn := time.Now()
		printf(
			"[%s] the process is starting\n",
			startedOn.Format(time.RFC3339),
		)

		select {
		case err := <-waitErrCh:
			if err == nil {
				printf(
					"[%s] the process exited successfully\n",
					time.Now().Format(time.RFC3339),
				)
				return
			}

			sleepDur := max(100*time.Millisecond, (15*time.Second)-time.Since(startedOn))
			if realErr := (*os.PathError)(nil); errors.As(err, &realErr) {
				printf(
					"[%s] failed to execute '%s' (%s), restarting in %s\n",
					time.Now().Format(time.RFC3339),
					s.runPath(),
					realErr.Err.Error(),
					sleepDur.Round(time.Millisecond),
				)
			} else if realErr := (*exec.ExitError)(nil); errors.As(err, &realErr) {
				wait := realErr.Sys().(syscall.WaitStatus)
				if wait.Exited() {
					printf(
						"[%s] the process exited with status code %d, restarting in %s\n",
						time.Now().Format(time.RFC3339),
						wait.ExitStatus(),
						sleepDur.Round(time.Millisecond),
					)
				} else {
					printf(
						"[%s] the process killed by signal '%s', restarting in %s\n",
						time.Now().Format(time.RFC3339),
						wait.Signal(),
						sleepDur.Round(time.Millisecond),
					)
				}
			} else {
				panic(err)
			}

			select {
			case <-time.After(sleepDur):
			case sig := <-sigCh:
				switch sig {
				case syscall.SIGTERM:
					printf(
						"[%s] got signal '%s', while in restart back-off\n",
						time.Now().Format(time.RFC3339),
						sig.String(),
					)
					return
				case syscall.SIGUSR1:
					printf(
						"[%s] got signal '%s', while in restart back-off, restart immediately\n",
						time.Now().Format(time.RFC3339),
						sig.String(),
					)
				default:
					panic("should not happen")
				}
			}

		case sig := <-sigCh:
			switch sig {
			case syscall.SIGTERM:
				printf(
					"[%s] got signal '%s', forward it to the process and wait for 15 seconds\n",
					time.Now().Format(time.RFC3339),
					sig.String(),
				)
			case syscall.SIGUSR1:
				printf(
					"[%s] got signal '%s' to restart the process, send signal '%s' and wait for 15 seconds\n",
					time.Now().Format(time.RFC3339),
					sig.String(),
					syscall.SIGTERM.String(),
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
					"[%s] process is not exited within 15 seconds, kill it\n",
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
