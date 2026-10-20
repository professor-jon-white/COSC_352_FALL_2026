package report

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Renderer is the Strategy interface for output formatting.
type Renderer interface {
	Render(r *Report) string
}

// NewRenderer is a Factory selecting a renderer by name.
func NewRenderer(format string) (Renderer, error) {
	switch strings.ToLower(format) {
	case "text", "terminal":
		return TextRenderer{BarWidth: 40}, nil
	case "markdown", "md":
		return MarkdownRenderer{BarWidth: 40}, nil
	}
	return nil, fmt.Errorf("unknown output format %q (use text or markdown)", format)
}

// TextRenderer renders for a terminal.
type TextRenderer struct{ BarWidth int }

func (t TextRenderer) Render(r *Report) string {
	var sb strings.Builder
	sb.WriteString(r.Title + "\n" + strings.Repeat("=", utf8.RuneCountInString(r.Title)) + "\n")
	for _, m := range r.Meta {
		sb.WriteString(m + "\n")
	}
	for _, s := range r.Sections {
		sb.WriteString("\n\n" + s.Heading + "\n" + strings.Repeat("-", utf8.RuneCountInString(s.Heading)) + "\n")
		if s.Subheading != "" {
			sb.WriteString(s.Subheading + "\n")
		}
		for _, bl := range s.Blocks {
			switch bl.Kind {
			case KindPara:
				sb.WriteString("\n" + bl.Text + "\n")
			case KindBullet:
				sb.WriteString("  * " + bl.Text + "\n")
			case KindChart:
				sb.WriteString("\n" + ASCIIChart(bl.Chart, t.BarWidth))
			case KindTable:
				sb.WriteString("\n" + ASCIITable(bl.Table))
			}
		}
	}
	return sb.String()
}

// MarkdownRenderer renders a report.md file.
type MarkdownRenderer struct{ BarWidth int }

func (m MarkdownRenderer) Render(r *Report) string {
	var sb strings.Builder
	sb.WriteString("# " + r.Title + "\n\n")
	for _, line := range r.Meta {
		sb.WriteString("- " + line + "\n")
	}
	for _, s := range r.Sections {
		sb.WriteString("\n## " + s.Heading + "\n\n")
		if s.Subheading != "" {
			sb.WriteString("*" + s.Subheading + "*\n\n")
		}
		for _, bl := range s.Blocks {
			switch bl.Kind {
			case KindPara:
				sb.WriteString(bl.Text + "\n\n")
			case KindBullet:
				sb.WriteString("- " + bl.Text + "\n")
			case KindChart:
				sb.WriteString("\n```text\n" + ASCIIChart(bl.Chart, m.BarWidth) + "```\n\n")
			case KindTable:
				sb.WriteString("\n" + MarkdownTable(bl.Table) + "\n")
			}
		}
	}
	return sb.String()
}

// ASCIIChart draws a horizontal bar chart.
func ASCIIChart(c Chart, width int) string {
	var sb strings.Builder
	if c.Title != "" {
		sb.WriteString(c.Title + "\n")
	}
	if len(c.Values) == 0 {
		sb.WriteString("  (no data)\n")
		return sb.String()
	}
	maxV, labelW, integral := 0.0, 0, true
	for i, v := range c.Values {
		if v > maxV {
			maxV = v
		}
		if l := utf8.RuneCountInString(labelAt(c, i)); l > labelW {
			labelW = l
		}
		if v != math.Trunc(v) {
			integral = false
		}
	}
	if labelW > 30 {
		labelW = 30
	}
	for i, v := range c.Values {
		n := 0
		if maxV > 0 && v > 0 {
			n = int(math.Round(v / maxV * float64(width)))
			if n == 0 {
				n = 1
			}
		}
		sb.WriteString(fmt.Sprintf("  %-*s | %s %s%s\n", labelW, truncate(labelAt(c, i), labelW),
			strings.Repeat("█", n), FormatNum(v, integral), c.Unit))
	}
	return sb.String()
}

func labelAt(c Chart, i int) string {
	if i < len(c.Labels) {
		return c.Labels[i]
	}
	return ""
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-3]) + "..."
}

// FormatNum formats with thousands separators (integers) or one decimal.
func FormatNum(v float64, integral bool) string {
	if !integral {
		return fmt.Sprintf("%.1f", v)
	}
	return Commas(int64(v))
}

// Commas formats 1234567 as "1,234,567".
func Commas(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// ASCIITable draws a boxed table.
func ASCIITable(t Table) string {
	w := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		w[i] = utf8.RuneCountInString(h)
	}
	for _, row := range t.Rows {
		for i := 0; i < len(row) && i < len(w); i++ {
			if l := utf8.RuneCountInString(row[i]); l > w[i] {
				w[i] = l
			}
		}
	}
	var sb strings.Builder
	sep := "  +"
	for _, x := range w {
		sep += strings.Repeat("-", x+2) + "+"
	}
	line := func(cells []string) {
		sb.WriteString("  |")
		for i := range w {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			sb.WriteString(fmt.Sprintf(" %-*s |", w[i], c))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(sep + "\n")
	line(t.Headers)
	sb.WriteString(sep + "\n")
	for _, row := range t.Rows {
		line(row)
	}
	sb.WriteString(sep + "\n")
	return sb.String()
}

// MarkdownTable renders a GitHub-flavored Markdown table.
func MarkdownTable(t Table) string {
	esc := func(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
	var sb strings.Builder
	sb.WriteString("|")
	for _, h := range t.Headers {
		sb.WriteString(" " + esc(h) + " |")
	}
	sb.WriteString("\n|")
	for range t.Headers {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")
	for _, row := range t.Rows {
		sb.WriteString("|")
		for i := range t.Headers {
			c := ""
			if i < len(row) {
				c = row[i]
			}
			sb.WriteString(" " + esc(c) + " |")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
