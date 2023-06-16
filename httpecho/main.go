package main

import (
	"fmt"
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

	runExtraUDPEcho()

	_, err := strconv.Atoi(port)
	check(err)

	err = http.ListenAndServe(":"+port, http.HandlerFunc(handler))
	check(err)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got http request: %s > %s %s\n", r.RemoteAddr, r.Method, r.URL.EscapedPath())

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

func runExtraUDPEcho() {
	port := os.Getenv("UDPPORT")
	if port == "" {
		return
	}

	_, err := strconv.Atoi(port)
	check(err)

	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	check(err)

	conn, err := net.ListenUDP("udp", addr)
	check(err)

	go func() {
		var readBuf, writeBuf [64 << 10]byte // 64KiB
		for {
			n, peer, err := conn.ReadFromUDP(readBuf[:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error read udp: %s\n", err.Error())
				continue
			}

			fmt.Printf("got udp request: %s > size=%d\n", peer.String(), n)

			dataIn := readBuf[:n]
			dataOut := writeBuf[:0]

			dataOut = append(dataOut, peer.String()...)
			dataOut = append(dataOut, " > "...)
			dataOut = append(dataOut, dataIn...)

			_, _ = conn.WriteToUDP(dataOut, peer)
		}
	}()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
