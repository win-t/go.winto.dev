package pkg

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
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
	// state directory already exists, so someone else is managed it.
	// wait few seconds before take it over.
	var pidStateExists bool
	for until := time.Now().Add(15 * time.Second); time.Now().Before(until); {
		var pid int
		pid, pidStateExists = s.getSupervisorPidState()
		if pidStateExists {
			if pid != 0 {
				return false
			} else {
				// pid state exists but process is not running, taiking over
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !pidStateExists {
		panic("daemonize state folder exists but pid state is not found")
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

		p, err := forkExec(&os.ProcAttr{
			Sys:   &syscall.SysProcAttr{Setsid: true},
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr, wInfo},
		})
		check(err)
		p.Wait()

		var buf [8]byte
		rInfo.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _ := rInfo.Read(buf[:])
		if slices.Compare(buf[:n], []byte("ok")) != 0 {
			stderr, _ := os.ReadFile(s.supervisorLog())
			os.Stderr.Write(stderr)
			fmt.Fprintln(os.Stderr, "Failed to start daemonize process")
			os.Exit(1)
		}

	case 1:
		stdin, err := os.Open("/dev/null")
		check(err)
		logFile, err := os.OpenFile(s.supervisorLog(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		check(err)

		_, err = forkExec(&os.ProcAttr{
			Sys:   &syscall.SysProcAttr{Setpgid: true},
			Files: []*os.File{stdin, logFile, logFile, os.NewFile(3, "info")},
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
	fmt.Printf(
		"[%s] daemonize started for service dir '%s'\n",
		time.Now().Format(time.RFC3339),
		string(s),
	)
	defer func() {
		fmt.Printf(
			"[%s] daemonize exited\n",
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
		go func() {
			err := cmd.Start()
			if err != nil {
				waitErrCh <- err
				return
			}
			fmt.Printf(
				"[%s] process '%s' started with pid %d\n",
				time.Now().Format(time.RFC3339),
				s.runPath(),
				cmd.Process.Pid,
			)
			waitErrCh <- cmd.Wait()
		}()

		beforeWait := time.Now()
		select {
		case err := <-waitErrCh:
			if err == nil {
				fmt.Printf(
					"[%s] the process exited successfully\n",
					time.Now().Format(time.RFC3339),
				)
				return
			}

			sleepDur := max(100*time.Millisecond, (15*time.Second)-time.Since(beforeWait))
			if realErr := (*exec.ExitError)(nil); errors.As(err, &realErr) {
				status := realErr.Sys().(syscall.WaitStatus)
				if status.Signaled() {
					fmt.Printf(
						"[%s] the process killed by signal %d (%s), restarting in %s\n",
						time.Now().Format(time.RFC3339),
						int(status.Signal()),
						status.Signal().String(),
						sleepDur.Round(time.Millisecond).String(),
					)
				} else {
					fmt.Printf(
						"[%s] the process exited with status code %d, restarting in %s\n",
						time.Now().Format(time.RFC3339),
						status.ExitStatus(),
						sleepDur.Round(time.Millisecond).String(),
					)
				}
			} else {
				fmt.Printf(
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
					fmt.Printf(
						"[%s] exit is requested while in restart back-off\n",
						time.Now().Format(time.RFC3339),
					)
					return
				case syscall.SIGUSR1:
					fmt.Printf(
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
				fmt.Printf(
					"[%s] exit is requested, send termination signal to the process\n",
					time.Now().Format(time.RFC3339),
				)
			case syscall.SIGUSR1:
				fmt.Printf(
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
				fmt.Printf(
					"[%s] process exited within 15 seconds\n",
					time.Now().Format(time.RFC3339),
				)
			case <-time.After(15 * time.Second):
				fmt.Printf(
					"[%s] the process is not exited within 15 seconds, force kill it\n",
					time.Now().Format(time.RFC3339),
				)
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				<-waitErrCh
			}

			if sig == syscall.SIGTERM {
				return
			}
		}
	}
}
