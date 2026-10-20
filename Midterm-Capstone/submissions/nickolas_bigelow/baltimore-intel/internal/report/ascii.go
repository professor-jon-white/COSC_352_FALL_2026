package report

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Charset lets the same charts render with Unicode blocks (default) or plain
// ASCII for terminals/fonts without block glyphs.
type Charset struct {
	Full    string
	Partial []string // 1/8 .. 7/8 widths
	Shades  []string // heatmap, light -> dark
	Dot     string
	Many    string
}

var Unicode = Charset{
	Full:    "█",
	Partial: []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"},
	Shades:  []string{" ", "░", "▒", "▓", "█"},
	Dot:     "•", Many: "●",
}

var ASCII = Charset{
	Full:    "#",
	Partial: []string{"", "", "", "+", "+", "+", "+"},
	Shades:  []string{" ", ".", ":", "*", "#", "@"},
	Dot:     "o", Many: "@",
}

func width(s string) int { return utf8.RuneCountInString(s) }

func pad(s string, w int, right bool) string {
	n := w - width(s)
	if n <= 0 {
		return s
	}
	if right {
		return strings.Repeat(" ", n) + s
	}
	return s + strings.Repeat(" ", n)
}

func trunc(s string, w int) string {
	if width(s) <= w {
		return s
	}
	r := []rune(s)
	return string(r[:w-1]) + "…"
}

// BarLines draws a horizontal bar chart.
func BarLines(b *Bars, cs Charset, barWidth int) []string {
	if len(b.Values) == 0 {
		return []string{"  (no data)"}
	}
	labW, maxV := 0, 0.0
	for i, l := range b.Labels {
		labW = max(labW, width(trunc(l, 28)))
		if !math.IsNaN(b.Values[i]) {
			maxV = math.Max(maxV, b.Values[i])
		}
	}
	format := b.Format
	if format == "" {
		format = "%.0f"
	}
	var out []string
	for i, v := range b.Values {
		bar := ""
		if maxV > 0 && v > 0 {
			eighths := int(math.Round(v / maxV * float64(barWidth*8)))
			bar = strings.Repeat(cs.Full, eighths/8)
			if rem := eighths % 8; rem > 0 {
				bar += cs.Partial[rem-1]
			}
		}
		val := fmt.Sprintf(format, v)
		if math.IsNaN(v) {
			val = "n/a"
		}
		out = append(out, fmt.Sprintf("  %s │%s %s%s", pad(trunc(b.Labels[i], 28), labW, false),
			pad(bar, barWidth, false), val, b.Unit))
	}
	return out
}

// HeatLines draws a shaded grid with a legend.
func HeatLines(h *Heatmap, cs Charset) []string {
	maxV := 0.0
	for _, row := range h.Values {
		for _, v := range row {
			maxV = math.Max(maxV, v)
		}
	}
	labW := 0
	for _, l := range h.RowLabels {
		labW = max(labW, width(l))
	}
	cellW := 2
	var out []string
	head := "  " + strings.Repeat(" ", labW) + " "
	for i, c := range h.ColLabels {
		if i%3 == 0 {
			head += pad(c, cellW*3, false)
		}
	}
	out = append(out, head)
	for r, row := range h.Values {
		line := "  " + pad(h.RowLabels[r], labW, false) + " "
		for _, v := range row {
			lvl := 0
			if maxV > 0 {
				lvl = int(math.Round(v / maxV * float64(len(cs.Shades)-1)))
			}
			line += strings.Repeat(cs.Shades[lvl], cellW)
		}
		out = append(out, line)
	}
	legend := "  legend: "
	for i, s := range cs.Shades {
		lo := maxV * float64(i) / float64(len(cs.Shades)-1)
		legend += fmt.Sprintf("[%s]≈%.0f ", strings.Repeat(s, 2), lo)
	}
	return append(out, legend)
}

// ScatterLines draws an x/y plot on a character grid.
func ScatterLines(s *Scatter, cs Charset, w, h int) []string {
	if len(s.X) == 0 {
		return []string{"  (no data)"}
	}
	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for i := range s.X {
		minX, maxX = math.Min(minX, s.X[i]), math.Max(maxX, s.X[i])
		minY, maxY = math.Min(minY, s.Y[i]), math.Max(maxY, s.Y[i])
	}
	if maxX == minX {
		maxX++
	}
	if maxY == minY {
		maxY++
	}
	grid := make([][]int, h)
	for i := range grid {
		grid[i] = make([]int, w)
	}
	for i := range s.X {
		cx := int((s.X[i] - minX) / (maxX - minX) * float64(w-1))
		cy := int((s.Y[i] - minY) / (maxY - minY) * float64(h-1))
		grid[h-1-cy][cx]++
	}
	yl := fmt.Sprintf("%.0f", maxY)
	labW := max(width(yl), width(fmt.Sprintf("%.0f", minY)))
	out := []string{"  " + s.YLabel}
	for r, row := range grid {
		lab := ""
		if r == 0 {
			lab = yl
		} else if r == h-1 {
			lab = fmt.Sprintf("%.0f", minY)
		}
		line := "  " + pad(lab, labW, true) + " │"
		for _, n := range row {
			switch {
			case n == 0:
				line += " "
			case n == 1:
				line += cs.Dot
			default:
				line += cs.Many
			}
		}
		out = append(out, line)
	}
	out = append(out, "  "+strings.Repeat(" ", labW)+" └"+strings.Repeat("─", w))
	xr := fmt.Sprintf("%.0f", maxX)
	out = append(out, "  "+strings.Repeat(" ", labW+2)+pad(fmt.Sprintf("%.0f", minX), w-width(xr), false)+xr)
	out = append(out, "  "+strings.Repeat(" ", labW+2)+s.XLabel+fmt.Sprintf("   (%s = 1 neighborhood, %s = several)", cs.Dot, cs.Many))
	return out
}

// TableLines draws a boxed table.
func TableLines(t *Table) []string {
	ws := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		ws[i] = width(h)
	}
	for _, row := range t.Rows {
		for i := range ws {
			if i < len(row) {
				ws[i] = max(ws[i], width(trunc(row[i], 34)))
			}
		}
	}
	sep := func(l, m, r string) string {
		parts := make([]string, len(ws))
		for i, w := range ws {
			parts[i] = strings.Repeat("─", w+2)
		}
		return "  " + l + strings.Join(parts, m) + r
	}
	line := func(cells []string, header bool) string {
		parts := make([]string, len(ws))
		for i, w := range ws {
			c := ""
			if i < len(cells) {
				c = trunc(cells[i], 34)
			}
			right := !header && i < len(t.Right) && t.Right[i]
			parts[i] = " " + pad(c, w, right) + " "
		}
		return "  │" + strings.Join(parts, "│") + "│"
	}
	out := []string{sep("┌", "┬", "┐"), line(t.Headers, true), sep("├", "┼", "┤")}
	for _, r := range t.Rows {
		out = append(out, line(r, false))
	}
	return append(out, sep("└", "┴", "┘"))
}
