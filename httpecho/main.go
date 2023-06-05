package main

import (
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	_, err := strconv.Atoi(port)
	check(err)

	err = http.ListenAndServe(":"+port, http.HandlerFunc(handler))
	check(err)
}

func handler(w http.ResponseWriter, r *http.Request) {
	cty, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if strings.HasPrefix(cty, "text/") || r.ContentLength == 0 {
		w.Header().Add("Content-Type", "text/plain")
	} else {
		w.Header().Add("Content-Type", "application/octet-stream")
	}

	localAddr, _ := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if localAddr != nil {
		w.Header().Add("X-Local-Addr", localAddr.String())
	}

	if r.RemoteAddr != "" {
		w.Header().Add("X-Remote-Addr", r.RemoteAddr)
	}

	r.Write(w)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
