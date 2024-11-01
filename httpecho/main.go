package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var buf [6]byte

	id := os.Getenv("ID")
	if id == "" {
		_, _ = io.ReadFull(rand.Reader, buf[:])
		id = base64.RawURLEncoding.EncodeToString(buf[:])
	}

	_, _ = io.ReadFull(rand.Reader, buf[:])
	instance := base64.RawURLEncoding.EncodeToString(buf[:])

	runExtraUDPEcho(port, id, instance)

	_, err := strconv.Atoi(port)
	check(err)

	cert := os.Getenv("TLS_CERT")
	key := os.Getenv("TLS_KEY")
	if cert == "" || key == "" {
		err = http.ListenAndServe(":"+port, handler(id, instance))
	} else {
		err = http.ListenAndServeTLS(":"+port, cert, key, handler(id, instance))
	}
	check(err)
}

func handler(id, instance string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("got http request: %s > %s %s\n", r.RemoteAddr, r.Method, r.URL.EscapedPath())

		if r.ContentLength > 10<<20 { // 10MiB
			http.Error(w, "request entity too large", http.StatusRequestEntityTooLarge)
			return
		}

		w.Header().Set("X-Id", id)
		w.Header().Set("X-Instance", instance)

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

		dir := ""
		if runtime.GOOS == "linux" {
			dir = "/dev/shm"
		}
		f, err := os.CreateTemp(dir, "httpecho-body-*")
		check(err)
		defer os.Remove(f.Name())
		defer f.Close()

		err = r.Write(f)
		check(err)

		_, err = f.Seek(0, io.SeekStart)
		check(err)

		_, _ = io.Copy(w, f)
	}
}

func runExtraUDPEcho(port, id, instance string) {
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

			dataOut = append(dataOut, id...)
			dataOut = append(dataOut, ": "...)
			dataOut = append(dataOut, instance...)
			dataOut = append(dataOut, ": "...)
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
