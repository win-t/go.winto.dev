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
	_ "tailscale.com/feature/ssh"
	"tailscale.com/logtail"
	"tailscale.com/ssh/tailssh"
	"tailscale.com/tsnet"
)

func init() {
	os.Setenv("TS_DEBUG_DERP_WS_CLIENT", "1")
	os.Setenv("TS_NO_LOGS_NO_SUPPORT", "1")
	logtail.Disable()
}
func main() { errorsmain.Exec(run) }

var shell string
var globalCtx context.Context

func run() {
	shell, _ = exec.LookPath("bash")
	if shell == "" {
		shell, _ = exec.LookPath("sh")
	}
	errors.Expect(shell != "", "expecting working shell")

	var cancelGlobalCtx context.CancelFunc
	globalCtx, cancelGlobalCtx = context.WithCancel(context.Background())
	defer cancelGlobalCtx()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigs:
		case <-globalCtx.Done():
		}
		signal.Stop(sigs)
		cancelGlobalCtx()
	}()

	hostname, _ := os.Hostname()
	ts := tsnet.Server{
		Hostname:  "tssh-" + hostname,
		Ephemeral: true,
	}

	tsstatus, err := ts.Up(globalCtx)
	errors.Check(err)
	defer time.Sleep(1 * time.Second) // give some time so all pending FIN is sent

	log.Printf("tssh: server is up, IPs=%v", tsstatus.TailscaleIPs)

	ln, err := ts.ListenSSH(":22")
	errors.Check(err)

	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		<-globalCtx.Done()
		log.Printf("tssh: shutting down")
		wg.Wait()
		ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if globalCtx.Err() != nil {
			break
		}
		errors.Check(err)
		wg.Go(func() {
			sess := c.(*tailssh.Session)
			if err := errors.Catch0(func() {
				handler(sess)
			}); err != nil {
				log.Print("tssh: " + errors.Format(err))
				sess.Stderr().Write([]byte("\nInternal error, please check server logs\n"))
				sess.Exit(1)
			}
			sess.Close()
		})
	}
}

func handler(sess *tailssh.Session) {
	ptyReq, winCh, isPty := sess.Pty()

	log.Printf("tssh: new session: addr=%s, pty=%t\n", sess.RemoteAddr().String(), isPty)
	defer log.Printf("tssh: session closed: addr=%s, pty=%t\n", sess.RemoteAddr().String(), isPty)

	if !isPty {
		noPty(sess)
		return
	}

	cmd := newCmd(sess)
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

	sessWait(sess, cmd)
}

func newCmd(sess *tailssh.Session) *exec.Cmd {
	var args []string
	if raw := sess.RawCommand(); raw != "" {
		args = append(args, "-c", raw)
	}
	cmd := exec.Command(shell, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TSSH_PID="+strconv.Itoa(os.Getpid()))
	cmd.Env = append(cmd.Env, sess.Environ()...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd
}

func noPty(sess *tailssh.Session) {
	cmd := newCmd(sess)

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
	sessWait(sess, cmd)
}

func sessWait(sess *tailssh.Session, cmd *exec.Cmd) {
	var waitErr error
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	ctxDone := false
	select {
	case <-sess.Context().Done():
		ctxDone = true
	case <-globalCtx.Done():
		ctxDone = true
	case waitErr = <-waitCh:
	}
	if ctxDone {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		waitErr = <-waitCh
	}

	sess.Exit(waitExitCode(waitErr))
}

func waitExitCode(err error) int {
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
