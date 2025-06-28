package main

import (
	"fmt"
	"os"
	"path/filepath"

	daemonize "go.winto.dev/daemonize/pkg"
	gorundir "go.winto.dev/gorundir/pkg"
)

var tools = map[string]func(){
	"gorundir":            gorundir.Main,
	"daemonize":           daemonize.Main,
	"proxy-service-setup": proxySetup,
}

func main() {
	toolName := filepath.Base(os.Args[0])

	var main func()
	if toolName == "webhosting-helper" {
		main = selfMain
	} else {
		var ok bool
		main, ok = tools[toolName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Tool '%s' not found.\n", toolName)
			os.Exit(1)
		}
	}

	main()
}

func selfMain() {
	if len(os.Args) != 2 || os.Args[1] != "install-symlinks" {
		fmt.Fprintln(os.Stderr, "Usage: webhosting-helper install-symlinks")
		os.Exit(1)
	}

	f, err := os.Executable()
	check(err)
	err = os.Chdir(filepath.Dir(f))
	check(err)
	for toolName := range tools {
		os.Remove(toolName)
		err = os.Symlink("webhosting-helper", toolName)
		check(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
