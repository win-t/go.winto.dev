package envparser

import (
	"fmt"
	"strings"
)

type ParseError struct {
	Items []struct {
		Key   string
		Value string
		Cause error
	}
}

func (p *ParseError) Error() string {
	points := make([]string, len(p.Items))
	for i, item := range p.Items {
		points[i] = fmt.Sprintf("* cannot parse env %s: %s", item.Key, item.Cause.Error())
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(points), strings.Join(points, "\n\t"))
}

func (p *ParseError) append(key, value string, cause error) {
	p.Items = append(p.Items, struct {
		Key   string
		Value string
		Cause error
	}{
		Key:   key,
		Value: value,
		Cause: cause,
	})
}
