package pkg

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func (s svc) setupForwarder(ctx context.Context, wg *sync.WaitGroup) (*os.File, *os.File) {
	stdoutCh := newNotifyCh()
	stdout := setupForwarderBackend(ctx, wg, s.runLogStdoutPath(), stdoutCh)

	stderrCh := newNotifyCh()
	stderr := setupForwarderBackend(ctx, wg, s.runLogStderrPath(), stderrCh)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR2)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigCh:
				stdoutCh.notify()
				stderrCh.notify()
			}
		}
	}()

	return stdout, stderr
}

func setupForwarderBackend(parentCtx context.Context, wg *sync.WaitGroup, path string, notifyCh notifyCh) *os.File {
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
					case <-notifyCh.ch:
					}
					cancelCtx()
				}()
				fwd.doForward(ctx, path)
			}()
		}

		if len(fwd.chunk) == 0 {
			return
		}

		var err error
		func() {
			var dst *os.File
			dst, err = openLogFile(path)
			if err != nil {
				return
			}
			defer func() { _ = dst.Close() }()

			ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelCtx()
			go func() { <-ctx.Done(); _ = dst.SetWriteDeadline(time.Unix(0, 0)) }()

			var n int
			n, err = dst.Write(fwd.chunk)
			fwd.chunk = fwd.chunk[n:]
		}()

		if len(fwd.chunk) != 0 {
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			printf(
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
	src   *os.File
	buf   [4 << 10]byte
	chunk []byte
}

// return true mean the caller need to call this function again
func (fwd *forwardState) doForward(ctx context.Context, path string) {
	var wg sync.WaitGroup
	defer wg.Wait()

	var cancelCtx context.CancelFunc
	ctx, cancelCtx = context.WithCancel(ctx)
	defer cancelCtx()

	err := fwd.src.SetReadDeadline(time.Time{})
	check(err)

	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	dst, err := openLogFile(path)
	if err != nil {
		printf(
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
	defer func() { _ = dst.Close() }()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		_ = fwd.src.SetReadDeadline(time.Unix(0, 0))
		_ = dst.SetWriteDeadline(time.Unix(0, 0))
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
				printf(
					"[%s] failed to write to '%s' (%s), force reopen\n",
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
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NONBLOCK, 0o600)
}
