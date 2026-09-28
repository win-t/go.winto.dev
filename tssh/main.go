package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"go.winto.dev/errors"
	"go.winto.dev/errors/errorsmain"
	"tailscale.com/client/local"
	_ "tailscale.com/feature/ssh"
	"tailscale.com/logtail"
	"tailscale.com/ssh/tailssh"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

func init() {
	os.Setenv("TS_DEBUG_DERP_WS_CLIENT", "1")
	os.Setenv("TS_NO_LOGS_NO_SUPPORT", "1")
	logtail.Disable()
}

func main() {
	ctx := context.Background()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	errorsmain.ExecErrCallback(
		func() { run(ctx) },
		func(err error) { log.Print(errors.Format(err)) },
	)
}

func run(ctx context.Context) {
	shell, _ := exec.LookPath("bash")
	if shell == "" {
		shell, _ = exec.LookPath("sh")
	}
	errors.Expect(shell != "", "expecting working shell")

	var verboseLogf func(format string, v ...any)
	if os.Getenv("TSSH_VERBOSE") == "true" {
		verboseLogf = log.Printf
	}

	hostname, _ := os.Hostname()
	ts := tsnet.Server{
		Hostname:  "tssh-" + hostname,
		Ephemeral: true,
		UserLogf:  log.Printf,
		Logf:      verboseLogf,
	}

	tsstatus, err := ts.Up(ctx)
	errors.Check(err)
	defer time.Sleep(1 * time.Second) // give some time so all pending FIN is sent

	log.Printf("tssh: server is up, IPs=%v", tsstatus.TailscaleIPs)

	client, err := ts.LocalClient()
	errors.Check(err)

	ln, err := ts.ListenSSH(":22")
	errors.Check(err)

	h := handler{
		ctx:    ctx,
		shell:  shell,
		client: client,
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		<-ctx.Done()
		log.Printf("tssh: shutting down")
		wg.Wait()
		ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if ctx.Err() != nil {
			break
		}
		errors.Check(err)
		wg.Go(func() {
			sess := c.(*tailssh.Session)
			if err := errors.Catch0(func() {
				h.handle(sess)
			}); err != nil {
				log.Print("tssh: " + errors.Format(err))
				sess.Stderr().Write([]byte("\nInternal error, please check server logs\n"))
				sess.Exit(1)
			}
			sess.Close()
		})
	}
}

const tsshCap tailcfg.PeerCapability = "tailscale.winto.dev/cap/tssh"

type handler struct {
	ctx    context.Context
	shell  string
	client *local.Client
}

func (h *handler) handle(sess *tailssh.Session) {
	ptyReq, winCh, isPty := sess.Pty()

	if !h.hasCap(sess) {
		log.Printf("tssh: session denied: addr=%s, user=%s\n", sess.RemoteAddr().String(), sess.User())
		nl := "\n"
		if isPty {
			nl = "\n\r"
		}
		sess.Stderr().Write([]byte(nl + "Permission denied, missing " + string(tsshCap) + " capability" + nl))
		sess.Exit(1)
		return
	}

	log.Printf("tssh: new session: addr=%s, pty=%t\n", sess.RemoteAddr().String(), isPty)
	defer log.Printf("tssh: session closed: addr=%s, pty=%t\n", sess.RemoteAddr().String(), isPty)

	if !isPty {
		h.handleNoPty(sess)
		return
	}

	cmd := h.newCmd(sess)
	cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term)

	ptmx, tty, err := pty.Open()
	errors.Check(err)
	defer ptmx.Close()

	go func() { io.Copy(ptmx, sess) }()
	go func() { io.Copy(sess, ptmx) }()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.SysProcAttr.Setctty = true

	pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(ptyReq.Window.Height), Cols: uint16(ptyReq.Window.Width)})
	go func() {
		for win := range winCh {
			pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(win.Height), Cols: uint16(win.Width)})
		}
	}()

	err = cmd.Start()
	tty.Close()
	errors.Check(err)

	h.sessWait(sess, cmd)
}

func (h *handler) hasCap(sess *tailssh.Session) bool {
	ctx, cancel := context.WithTimeout(sess.Context(), 10*time.Second)
	defer cancel()

	who, err := h.client.WhoIs(ctx, sess.RemoteAddr().String())
	errors.Check(err)

	return who.CapMap.HasCapability(tsshCap)
}

func (h *handler) newCmd(sess *tailssh.Session) *exec.Cmd {
	var args []string
	if raw := sess.RawCommand(); raw != "" {
		args = append(args, "-c", raw)
	}
	cmd := exec.Command(h.shell, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TSSH_PID="+strconv.Itoa(os.Getpid()))
	cmd.Env = append(cmd.Env, sess.Environ()...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd
}

func (h *handler) handleNoPty(sess *tailssh.Session) {
	cmd := h.newCmd(sess)

	stdin, err := cmd.StdinPipe()
	errors.Check(err)
	go func() { io.Copy(stdin, sess); stdin.Close() }()

	stdout, err := cmd.StdoutPipe()
	errors.Check(err)
	go func() { io.Copy(sess, stdout); stdout.Close() }()

	stderr, err := cmd.StderrPipe()
	errors.Check(err)
	go func() { io.Copy(sess.Stderr(), stderr); stderr.Close() }()

	err = cmd.Start()
	errors.Check(err)
	h.sessWait(sess, cmd)
}

func (h *handler) sessWait(sess *tailssh.Session, cmd *exec.Cmd) {
	var waitErr error
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	ctxDone := false
	select {
	case <-sess.Context().Done():
		ctxDone = true
	case <-h.ctx.Done():
		ctxDone = true
	case waitErr = <-waitCh:
	}
	if ctxDone {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		waitErr = <-waitCh
	}

	sess.Exit(h.waitExitCode(waitErr))
}

func (h *handler) waitExitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				return 128 + int(ws.Signal())
			}
			return ws.ExitStatus()
		}
	}
	return 1
}
