package responsewriter

import (
	"bufio"
	"net"
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter

	// The status code that already written, otherwise 0.
	Status() int

	// Size of body that already written.
	Size() int

	// Tells if hijacked.
	Hijacked() bool
}

// copied from net/http source code
type rwUnwrapper interface {
	Unwrap() http.ResponseWriter
}

type wrapped struct {
	rw       http.ResponseWriter
	status   int
	size     int
	hijacked bool
}

var (
	_ http.ResponseWriter = (*wrapped)(nil)
	_ http.Hijacker       = (*wrapped)(nil)
	_ rwUnwrapper         = (*wrapped)(nil)
)

func Wrap(rw http.ResponseWriter) ResponseWriter {
	saved := rw
	for rw != nil {
		if w, ok := rw.(*wrapped); ok {
			return w
		}
		if u, ok := rw.(rwUnwrapper); ok {
			rw = u.Unwrap()
		} else {
			rw = nil
		}
	}
	return &wrapped{rw: saved}
}

func (w *wrapped) Unwrap() http.ResponseWriter {
	return w.rw
}

func (rw *wrapped) Header() http.Header {
	return rw.rw.Header()
}

func (rw *wrapped) WriteHeader(s int) {
	if rw.status == 0 {
		rw.status = s
		rw.rw.WriteHeader(s)
	}
}

func (rw *wrapped) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.rw.Write(b)
	rw.size += size
	return size, err
}

func (rw *wrapped) Status() int {
	return rw.status
}

func (rw *wrapped) Size() int {
	return rw.size
}

func (rw *wrapped) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, bufrw, err := http.NewResponseController(rw.rw).Hijack()
	rw.hijacked = err == nil
	return c, bufrw, err
}

func (rw *wrapped) Hijacked() bool {
	return rw.hijacked
}
