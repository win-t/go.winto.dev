package errors

import (
	"strings"
)

func makeOneLine(str string) string {
	str = strings.ReplaceAll(str, "\\", "\\\\")
	str = strings.ReplaceAll(str, "\r\n", "\n")
	str = strings.ReplaceAll(str, "\r", "\n")
	str = strings.ReplaceAll(str, "\n", "\\n")
	return str
}

// like Format, but you can filter what location to include in the formated string
func FormatWithFilter(err error, filter func(Location) bool) string {
	var sb strings.Builder

	firstError := true
	add := func(err error) error {
		if firstError {
			firstError = false
		} else {
			sb.WriteString("Caused by ")
		}

		sb.WriteString("Error => ")
		sb.WriteString(makeOneLine(err.Error()))
		sb.WriteByte('\n')

		if traced, ok := err.(*TracedErr); ok {
			firstErrTrace := true
			for _, l := range traced.Trace {
				if filter != nil && !filter(l) {
					continue
				}
				if firstErrTrace {
					sb.WriteString("  Stack Trace:\n")
					firstErrTrace = false
				}
				sb.WriteString("  - ")
				sb.WriteString(l.String())
				sb.WriteByte('\n')
			}
			return Unwrap(traced.Original)
		} else {
			return Unwrap(err)
		}
	}

	for err != nil {
		err = add(err)
	}

	return sb.String()
}

// like Format, but you can filter pkg location to include in the formated string
func FormatWithFilterPkgs(err error, pkgs ...string) string {
	return FormatWithFilter(err, func(l Location) bool { return l.InPkg(pkgs...) })
}

// Format representation of the Error, including stack trace.
//
// Use err.Error() if you want to get just the error string.
//
// the returned string is not stable, future version maybe returned different format.
func Format(err error) string {
	return FormatWithFilter(err, defaultFilter)
}

var defaultFilter func(Location) bool

func SetFormatFilterPkgs(pkgs ...string) {
	defaultFilter = func(l Location) bool { return l.InPkg(pkgs...) }
}
