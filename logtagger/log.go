package logtagger

import (
	"io"
	"sync"
)

type LogTagger struct {
	mu  sync.Mutex
	Dst io.Writer
}

func New(dst io.Writer) *LogTagger {
	return &LogTagger{Dst: dst}
}

type Writer struct {
	*LogTagger
	tag string
}

func (l *LogTagger) Tag(tag string) *Writer {
	return &Writer{
		LogTagger: l,
		tag:       tag,
	}
}

var pool sync.Pool

func (pw *Writer) Write(p []byte) (int, error) {
	buf, _ := pool.Get().([]byte)
	defer func() {
		if len(buf) > 0 {
			pool.Put(buf[:0])
		}
	}()

	next := 0
	for i := range p {
		if p[i] != '\n' {
			continue
		}
		buf = append(buf, "["...)
		buf = append(buf, pw.tag...)
		buf = append(buf, "] "...)
		buf = append(buf, p[next:i+1]...)
		next = i + 1
	}
	if next < len(p) {
		buf = append(buf, "["...)
		buf = append(buf, pw.tag...)
		buf = append(buf, ") "...)
		buf = append(buf, p[next:]...)
		buf = append(buf, '\n')
	}

	toWrite := buf

	pw.mu.Lock()
	defer pw.mu.Unlock()

	for len(toWrite) > 0 {
		n, err := pw.Dst.Write(toWrite)
		if err != nil {
			return 0, err
		}
		toWrite = toWrite[n:]
	}

	return len(p), nil
}
