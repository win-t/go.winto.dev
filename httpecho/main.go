package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var buf [6]byte

	id := os.Getenv("ID")
	if id == "" {
		_, err := io.ReadFull(rand.Reader, buf[:])
		check(err)
		id = base64.RawURLEncoding.EncodeToString(buf[:])
	}

	_, err := io.ReadFull(rand.Reader, buf[:])
	check(err)
	instance := base64.RawURLEncoding.EncodeToString(buf[:])

	runExtraUDPEcho(port, id, instance)

	_, err = strconv.Atoi(port)
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
	var pool sync.Pool
	const maxSize = 6 << 20        // 6MiB, assuming header is 1MiB and body is 5MiB
	concurrencyLimit := int32(100) // set rough upper limit of memory consumption, 100 * 6MiB = 600MiB

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("got http request: %s > %s %s\n", r.RemoteAddr, r.Method, r.URL.EscapedPath())

		w.Header().Set("X-Id", id)
		w.Header().Set("X-Instance", instance)

		localAddr, _ := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
		if localAddr != nil {
			w.Header().Add("X-Local-Addr", localAddr.String())
		}

		if r.RemoteAddr != "" {
			w.Header().Add("X-Remote-Addr", r.RemoteAddr)
		}

		c := atomic.AddInt32(&concurrencyLimit, -1)
		defer atomic.AddInt32(&concurrencyLimit, 1)
		if c < 0 {
			http.Error(w, "concurrency limit reached", http.StatusServiceUnavailable)
			return
		}

		buf, _ := pool.Get().(*bytes.Buffer)
		if buf == nil {
			buf = new(bytes.Buffer)
		}
		buf.Reset()
		defer pool.Put(buf)

		lw := &limitWriter{buf, maxSize}
		_ = r.Write(lw)
		if lw.limit < 0 {
			w.Header().Add("X-Truncated", "true")
		}

		if utf8.Valid(buf.Bytes()) {
			w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		} else {
			w.Header().Add("Content-Type", "application/octet-stream")
		}

		_, _ = io.Copy(w, buf)
	}
}

type limitWriter struct {
	w     io.Writer
	limit int
}

func (l *limitWriter) Write(p []byte) (n int, err error) {
	if l.limit <= 0 {
		return 0, nil // discard
	}
	n, err = l.w.Write(p)
	l.limit -= n
	return n, err
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
