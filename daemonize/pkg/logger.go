package pkg

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
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

	wg.Add(1)
	go func() {
		defer wg.Done()
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

func setupForwarderBackend(ctx context.Context, wg *sync.WaitGroup, path string, notifyCh notifyCh) *os.File {
	os.MkdirAll(filepath.Dir(path), 0o700)

	r, w, err := os.Pipe()
	check(err)

	wg.Add(1)
	go func() {
		defer wg.Done()
		state := srcState{file: r, reader: bufio.NewReader(r)}
		for ctx.Err() == nil {
			if state.reopenAndForward(ctx, path, notifyCh) {
				return
			}
		}
	}()

	return w
}

type srcState struct {
	file   *os.File
	reader *bufio.Reader
	chunk  []byte
}

func (src *srcState) reopenAndForward(ctx context.Context, path string, notifyCh notifyCh) bool {
	var wg sync.WaitGroup
	defer wg.Wait()

	var cancelCtx context.CancelFunc
	ctx, cancelCtx = context.WithCancel(ctx)
	defer cancelCtx()

	err := src.file.SetReadDeadline(time.Time{})
	check(err)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NONBLOCK, 0o600)
	if err != nil {
		printf(
			"[%s] WARN: failed to open '%s' (%s), log will be discarded, trying to reopen in 15 seconds\n",
			time.Now().Format(time.RFC3339),
			path,
			err.Error(),
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
			case <-time.After(15 * time.Second):
				notifyCh.notify()
			}
		}()
	}
	var fileClosed atomic.Bool
	closeFile := func() {
		if fileClosed.CompareAndSwap(false, true) {
			if file != nil {
				file.Close()
			}
		}
	}
	defer closeFile()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
		case <-notifyCh.ch:
		}
		err := src.file.SetReadDeadline(time.Unix(0, 0))
		check(err)
		closeFile()
	}()

	for {
		if len(src.chunk) == 0 {
			src.chunk, err = src.reader.ReadSlice('\n')
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					return false
				} else if errors.Is(err, io.EOF) {
					if len(src.chunk) == 0 {
						return true
					}
				} else if !errors.Is(err, bufio.ErrBufferFull) {
					panic(err)
				}
			}
		}

		if len(src.chunk) > 0 {
			var n int
			var err error
			if file != nil {
				n, err = file.Write(src.chunk)
			} else {
				n = len(src.chunk)
			}
			src.chunk = src.chunk[n:]
			if err != nil {
				if !fileClosed.Load() {
					printf(
						"[%s] WARN: failed to write to '%s' (%s), force reopen\n",
						time.Now().Format(time.RFC3339),
						path,
						err.Error(),
					)
				}
				return false
			}
		}
	}
}
