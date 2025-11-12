package logtagger

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

var _ io.Writer = (*Writer)(nil)

func Test1(t *testing.T) {
	buf := &bytes.Buffer{}

	tagger := New(buf)
	t1 := tagger.Tag("hello")
	t2 := tagger.Tag("world")

	fmt.Fprintf(t1, "this is a\ntest")
	fmt.Fprintf(t2, "another\ntest")
	fmt.Fprintf(t1, "final line\n")
	fmt.Fprintln(t2, "also final line")

	expected := `[hello] this is a
[hello) test
[world] another
[world) test
[hello] final line
[world] also final line
`

	if buf.String() != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, buf.String())
	}
}
