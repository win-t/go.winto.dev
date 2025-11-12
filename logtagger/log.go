// Package logtagger provides a writer that tags each line with a specified tag.
package logtagger

import (
	"io"
	"sync"
)

type LogTagger struct {
	mu  sync.Mutex
	Dst io.Writer
}

// New creates a new LogTagger that writes to the given destination.
//
// The Writer returned by [LogTagger.Tag] can be used concurrently.
func New(dst io.Writer) *LogTagger {
	return &LogTagger{Dst: dst}
}

type Writer struct {
	*LogTagger
	tag string
}

// Tag returns a Writer that tags each line with the specified tag.
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
