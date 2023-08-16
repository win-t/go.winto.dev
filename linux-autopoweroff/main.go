package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

var last int64

func main() {
	last = time.Now().Unix()
	go autopoweroff()
	panic(http.ListenAndServe(":8085", http.HandlerFunc(handler)))
}

func handler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.CommandContext(r.Context(), "loginctl", "list-sessions")
	cmd.Stdout, cmd.Stderr = w, w
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(w, "cmd error: %s\n", err.Error())
	}
	fmt.Fprintln(w, "=====")

	cmd = exec.CommandContext(r.Context(), "systemd-cgls", "-l", "-u", "user.slice")
	cmd.Stdout, cmd.Stderr = w, w
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(w, "cmd error: %s\n", err.Error())
	}
	fmt.Fprintln(w, "=====")

	poweroff := time.Unix(atomic.LoadInt64(&last), 0).Add(1 * time.Hour)
	fmt.Fprintf(w, "will auto poweroff at: %s\n", poweroff.Format(time.RFC3339Nano))
	fmt.Fprintln(w, "=====")
}

func autopoweroff() {
	step := func() bool {
		status, err := exec.Command("loginctl", "-o", "json").Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "exec error: %s\n", err.Error())
			return false
		}

		var data []struct{}
		err = json.Unmarshal([]byte(status), &data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unmarshal error: %s\n", err.Error())
			return false
		}

		now := time.Now().Unix()
		if len(data) != 0 {
			atomic.StoreInt64(&last, now)
			return false
		}

		if (now - last) < 3600 {
			return false
		}

		fmt.Println("Executing poweroff")
		cmd := exec.Command("poweroff")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cmd error: %s\n", err.Error())
			return false
		}

		return true
	}

	for !step() {
		time.Sleep(10 * time.Second)
	}
}

