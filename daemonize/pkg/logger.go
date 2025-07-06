package pkg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func (s svc) setupForwarder(ctx context.Context, wg *sync.WaitGroup) (*os.File, *os.File) {
	stdoutCh := make(chan struct{}, 1)
	stdout := setupForwarderBackend(ctx, wg, s.runLogStdoutPath(), stdoutCh)

	stderrCh := make(chan struct{}, 1)
	stderr := setupForwarderBackend(ctx, wg, s.runLogStderrPath(), stderrCh)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR2)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigCh:
				fmt.Printf(
					"[%s] Reopening log files...\n",
					time.Now().Format(time.RFC3339),
				)
				stdoutCh <- struct{}{}
				stderrCh <- struct{}{}
			}
		}
	}()

	return stdout, stderr
}

func setupForwarderBackend(parentCtx context.Context, wg *sync.WaitGroup, path string, notifyCh <-chan struct{}) *os.File {
	r, w, err := os.Pipe()
	check(err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		fwd := forwardState{src: r}
		for parentCtx.Err() == nil {
			func() {
				ctx, cancelCtx := context.WithCancel(parentCtx)
				defer cancelCtx()
				go func() {
					select {
					case <-ctx.Done():
					case <-notifyCh:
					}
					cancelCtx()
				}()
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf(
							"[%s] panic in log forwarder to: %v\n",
							time.Now().Format(time.RFC3339),
							r,
						)
					}
				}()
				fwd.doForward(ctx, path)
			}()
		}

		if len(fwd.chunk) == 0 {
			return
		}

		// last attempt to write the remaining chunk
		var err error
		func() {
			var dst *os.File
			dst, err = openLogFile(path)
			if err != nil {
				return
			}
			defer dst.Close()

			dst.SetWriteDeadline(time.Now().Add(5 * time.Second))

			var n int
			n, err = dst.Write(fwd.chunk)
			fwd.chunk = fwd.chunk[n:]
		}()

		if len(fwd.chunk) != 0 {
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			fmt.Printf(
				"[%s] discarding %d bytes of log to %s because of error '%s'\n",
				time.Now().Format(time.RFC3339),
				len(fwd.chunk),
				path,
				errMsg,
			)
		}
	}()

	return w
}

type forwardState struct {
	buf   [4 << 10]byte
	src   *os.File
	chunk []byte
}

func (fwd *forwardState) doForward(ctx context.Context, path string) {
	var wg sync.WaitGroup
	defer wg.Wait()

	var cancelCtx context.CancelFunc
	ctx, cancelCtx = context.WithCancel(ctx)
	defer cancelCtx()

	err := fwd.src.SetReadDeadline(time.Time{})
	check(err)

	dst, err := openLogFile(path)
	if err != nil {
		fmt.Printf(
			"[%s] failed to open '%s' (%s), suspend log forwarder for 15 second\n",
			time.Now().Format(time.RFC3339),
			path,
			err.Error(),
		)
		select {
		case <-ctx.Done():
		case <-time.After(15 * time.Second):
		}
		return
	}
	defer dst.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		fwd.src.SetReadDeadline(time.Unix(0, 0))
		dst.SetWriteDeadline(time.Unix(0, 0))
	}()

	for {
		if len(fwd.chunk) == 0 {
			n, err := fwd.src.Read(fwd.buf[:])
			fwd.chunk = fwd.buf[:n]
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					return
				} else if errors.Is(err, io.EOF) {
					if len(fwd.chunk) == 0 {
						<-ctx.Done()
						return
					}
				} else {
					panic(err)
				}
			}
		}

		if len(fwd.chunk) == 0 {
			panic("should not happen")
		}

		n, err := dst.Write(fwd.chunk)
		fwd.chunk = fwd.chunk[n:]
		if err != nil {
			if ctx.Err() == nil {
				fmt.Printf(
					"[%s] failed to write to '%s' (%s)\n",
					time.Now().Format(time.RFC3339),
					path,
					err.Error(),
				)
			}
			return
		}
	}
}

func openLogFile(path string) (*os.File, error) {
	os.MkdirAll(filepath.Dir(path), 0o700)
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}
