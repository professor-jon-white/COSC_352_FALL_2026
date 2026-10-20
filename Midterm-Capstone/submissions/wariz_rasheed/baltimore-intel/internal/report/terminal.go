package report

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode/utf8"
)

const (
	termWidth = 78
	barWidth  = 36
)

type terminal struct{}

func (terminal) Render(w io.Writer, s Summary) error {
	bw := bufio.NewWriter(w)
	rule := strings.Repeat("=", termWidth)

	fmt.Fprintln(bw, rule)
	fmt.Fprintln(bw, s.Title)
	if s.Subtitle != "" {
		fmt.Fprintln(bw, s.Subtitle)
	}
	fmt.Fprintln(bw, rule)
	writeTable(bw, s.Overview)

	for i, r := range s.Results {
		fmt.Fprintln(bw)
		fmt.Fprintln(bw, rule)
		fmt.Fprintf(bw, "QUESTION %d of %d: %s\n", i+1, len(s.Results), r.Title)
		fmt.Fprintf(bw, "For: %s\n", r.Audience)
		fmt.Fprintln(bw, rule)
		fmt.Fprintln(bw, wrap(r.Question, termWidth, ""))
		fmt.Fprintln(bw)
		fmt.Fprintln(bw, wrap("FINDING: "+r.Finding, termWidth, ""))
		for _, c := range r.Charts {
			writeChart(bw, c)
		}
		for _, t := range r.Tables {
			writeTable(bw, t)
		}
		if len(r.Notes) > 0 {
			fmt.Fprintln(bw)
			for _, n := range r.Notes {
				fmt.Fprintln(bw, wrap("Note: "+n, termWidth-2, "  "))
			}
		}
	}

	if len(s.Footer) > 0 {
		fmt.Fprintln(bw)
		fmt.Fprintln(bw, rule)
		for _, f := range s.Footer {
			fmt.Fprintln(bw, f)
		}
	}
	return bw.Flush()
}

func writeTable(w io.Writer, t Table) {
	if len(t.Headers) == 0 {
		return
	}
	fmt.Fprintln(w)
	if t.Title != "" {
		fmt.Fprintf(w, "  %s\n", t.Title)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  "+strings.Join(t.Headers, "\t"))
	rules := make([]string, len(t.Headers))
	for i, h := range t.Headers {
		rules[i] = strings.Repeat("-", utf8.RuneCountInString(h))
	}
	fmt.Fprintln(tw, "  "+strings.Join(rules, "\t"))
	for _, row := range t.Rows {
		fmt.Fprintln(tw, "  "+strings.Join(row, "\t"))
	}
	tw.Flush()
}

func writeChart(w io.Writer, c Chart) {
	if len(c.Bars) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", c.Title)
	labelW := 0
	for _, b := range c.Bars {
		if n := utf8.RuneCountInString(b.Label); n > labelW {
			labelW = n
		}
	}
	max := maxAbs(c.Bars)
	for _, b := range c.Bars {
		glyph := "█"
		if b.Value < 0 {
			glyph = "▒"
		}
		fmt.Fprintf(w, "  %-*s │%s %s\n", labelW, b.Label, strings.Repeat(glyph, barLen(b.Value, max, barWidth)), b.Text)
	}
}

// wrap word-wraps text to width columns, prefixing every line with indent.
func wrap(text string, width int, indent string) string {
	words := strings.Fields(text)
	var b strings.Builder
	lineLen := 0
	for i, word := range words {
		wl := utf8.RuneCountInString(word)
		switch {
		case i == 0:
			b.WriteString(indent)
			b.WriteString(word)
			lineLen = len(indent) + wl
		case lineLen+1+wl > width:
			b.WriteString("\n")
			b.WriteString(indent)
			b.WriteString(word)
			lineLen = len(indent) + wl
		default:
			b.WriteString(" ")
			b.WriteString(word)
			lineLen += 1 + wl
		}
	}
	return b.String()
}
