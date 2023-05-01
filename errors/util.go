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

	if errSlice, ok := err.(unwrapslice); ok {
		sb.WriteString("Multi Error:\n")
		for _, err := range errSlice.Unwrap() {
			sb.WriteString("- ")
			sb.WriteString(makeOneLine(err.Error()))
			sb.WriteByte('\n')
		}
	} else {
		firstError := true
		add := func(err error) {
			if firstError {
				firstError = false
			} else {
				sb.WriteString("Caused by ")
			}

			sb.WriteString("Error => ")
			sb.WriteString(makeOneLine(err.Error()))
			sb.WriteByte('\n')

			firstErrTrace := true
			for _, l := range StackTrace(err) {
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
		}

		for err != nil {
			add(err)
			err = Unwrap(err)
		}
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
	return FormatWithFilter(err, nil)
}
