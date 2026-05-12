package main

import (
	"net/http"
	"net/http/cgi"
	"os"
	"runtime"

	"go.winto.dev/daemonize/pkg"
)

func main() {
	if os.Args[0] == "daemonize" {
		pkg.Main()
		return
	}

	runtime.GOMAXPROCS(1)
	err := cgi.Serve(http.HandlerFunc(handler))
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	root
}
