package main

import (
	"fmt"
	"math"
	"strings"
)

// bar returns a horizontal bar whose length is proportional to value/max.
func bar(value, max float64, width int, glyph string) string {
	if max <= 0 || value <= 0 || math.IsNaN(value) {
		return ""
	}
	n := int(math.Round(value / max * float64(width)))
	if n < 1 {
		n = 1
	}
	return strings.Repeat(glyph, n)
}

// fmtVal prints integers without decimals and everything else with two.
func fmtVal(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// fmtFloat formats v with prec decimals, or "n/a" for NaN/Inf.
func fmtFloat(v float64, prec int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "n/a"
	}
	return fmt.Sprintf("%.*f", prec, v)
}

// pct formats a/b as a percentage, or "n/a" when b is zero.
func pct(a, b int) string {
	if b == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(a)/float64(b))
}

// nz maps NaN to zero so it can be drawn as an (empty) bar.
func nz(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// truncate shortens s to at most n characters, ending with "..." if cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 3 {
		return string(r[:n])
	}
	return string(r[:n-3]) + "..."
}

// writeBarChart draws one horizontal bar per label, scaled to the largest value.
func writeBarChart(sb *strings.Builder, labels []string, values []float64, unit string, width int) {
	maxV, maxL := 0.0, 0
	for i, v := range values {
		if v > maxV {
			maxV = v
		}
		if len(labels[i]) > maxL {
			maxL = len(labels[i])
		}
	}
	for i, v := range values {
		fmt.Fprintf(sb, "  %-*s | %s %s%s\n", maxL, labels[i], bar(v, maxV, width, "█"), fmtVal(v), unit)
	}
}

// writeDualHourChart draws a 24-row histogram with two bars per hour: a (█) and b (▒).
func writeDualHourChart(sb *strings.Builder, a, b []float64, width int) {
	maxV := 0.0
	for i := range a {
		maxV = math.Max(maxV, math.Max(a[i], b[i]))
	}
	for h := range a {
		fmt.Fprintf(sb, "  %02d:00 | %s %.1f%%\n", h, bar(a[h], maxV, width, "█"), a[h])
		fmt.Fprintf(sb, "        | %s %.1f%%\n", bar(b[h], maxV, width, "▒"), b[h])
	}
}

// writeTable prints a left-aligned text table with a header rule.
func writeTable(sb *strings.Builder, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	line := func(cells []string) {
		var lb strings.Builder
		lb.WriteString("  ")
		for i, c := range cells {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				lb.WriteString("  ")
			}
			fmt.Fprintf(&lb, "%-*s", widths[i], c)
		}
		sb.WriteString(strings.TrimRight(lb.String(), " "))
		sb.WriteString("\n")
	}
	line(headers)
	rule := make([]string, len(headers))
	for i, w := range widths {
		rule[i] = strings.Repeat("-", w)
	}
	line(rule)
	for _, r := range rows {
		line(r)
	}
}
