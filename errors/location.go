package errors

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

type pcbuff = [512]uintptr

var pcbuffPool sync.Pool

// Single location of the trace
type Location struct {
	file  string
	line  int
	func_ string
}

// String representation of Location
func (l *Location) String() string {
	if l.func_ == "" {
		return fmt.Sprintf("%s:%d", l.file, l.line)
	}

	return fmt.Sprintf("%s:%d (%s)", l.file, l.line, l.func_)
}

// The file path that this Location point to
func (l *Location) File() string { return l.file }

// The line that this Location point to
func (l *Location) Line() int { return l.line }

// The path-qualified function that this location point to
func (l *Location) Func() string { return l.func_ }

// return true if this location is in package pkgs
func (l *Location) InPkg(pkgs ...string) bool {
	for _, pkg := range pkgs {
		if nameInPkg(l.func_, pkg) {
			return true
		}
	}
	return false
}

// skip==0 mean stack trace for where getLocs is called
func getLocs(skip int) (locations []Location) {
	var data *pcbuff
	if tmp, ok := pcbuffPool.Get().(*pcbuff); ok {
		data = tmp
	} else {
		data = new(pcbuff)
	}

	pc := data[:]
	pc = pc[:runtime.Callers(skip+2, pc)]
	if len(pc) == 0 {
		return nil
	}

	locations = make([]Location, 0, len(pc))

	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if frame.Line != 0 && frame.File != "" &&
			!nameInPkg(frame.Function, "runtime") &&
			!nameInPkg(frame.Function, "go.winto.dev/errors") {
			locations = append(locations, Location{
				func_: frame.Function,
				file:  frame.File,
				line:  frame.Line,
			})
		}
		if !more {
			break
		}
	}

	pcbuffPool.Put(data)

	return
}

func nameInPkg(name, pkg string) bool {
	if name == pkg {
		return true
	}

	if !strings.HasPrefix(name, pkg) {
		return false
	}

	separator := name[len(pkg)]

	return separator == '.' || separator == '/'
}
