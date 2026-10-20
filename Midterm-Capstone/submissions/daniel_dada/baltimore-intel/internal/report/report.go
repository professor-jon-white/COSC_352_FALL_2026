package report

import (
	"fmt"
	"strings"
)

type Section struct {
	Title       string
	Stakeholder string
	Lines       []string
}

func (s Section) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== %s ===\nFor: %s\n\n", s.Title, s.Stakeholder)
	for _, l := range s.Lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}

// Bar draws one ASCII bar scaled so that max fills `width` characters.
// Long labels are truncated so the bars stay aligned.
func Bar(label string, v, max, width int) string {
	if len(label) > 29 {
		label = label[:29]
	}
	n := 0
	if max > 0 {
		n = v * width / max
	}
	return fmt.Sprintf("%-30s %s %d", label, strings.Repeat("█", n), v)
}
