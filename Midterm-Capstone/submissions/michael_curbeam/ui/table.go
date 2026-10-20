// Package ui renders analytics results as ASCII tables and histograms.
package ui

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"baltimore-engine/analytics"
)

const (
	barWidth  = 40
	titleRule = "="
)

// RenderAll prints a banner followed by every result as a titled table,
// plus a histogram for results that request one.
func RenderAll(w io.Writer, results []analytics.Result) {
	banner := "BALTIMORE DATA ENGINE - 311 x VACANT BUILDINGS"
	fmt.Fprintln(w, strings.Repeat(titleRule, len(banner)))
	fmt.Fprintln(w, banner)
	fmt.Fprintln(w, strings.Repeat(titleRule, len(banner)))
	fmt.Fprintln(w)

	for _, r := range results {
		fmt.Fprintln(w, r.Question)
		RenderTable(w, r.Headers, r.Rows)
		if r.BarCol > 0 {
			RenderHistogram(w, r.Headers, r.Rows, r.BarCol-1)
		}
		if r.Note != "" {
			fmt.Fprintln(w, "Note:", r.Note)
		}
		fmt.Fprintln(w)
	}
}

// RenderTable prints headers and rows as an ASCII grid. Numeric cells are
// right-aligned; everything else is left-aligned.
func RenderTable(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, c := range row {
			if i < len(widths) {
				widths[i] = max(widths[i], utf8.RuneCountInString(c))
			}
		}
	}

	sep := "+"
	for _, n := range widths {
		sep += strings.Repeat("-", n+2) + "+"
	}
	line := func(cells []string, align bool) {
		fmt.Fprint(w, "|")
		for i, n := range widths {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			pad := strings.Repeat(" ", n-utf8.RuneCountInString(c))
			if align && isNumeric(c) {
				fmt.Fprintf(w, " %s%s |", pad, c)
			} else {
				fmt.Fprintf(w, " %s%s |", c, pad)
			}
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, sep)
	line(headers, false)
	fmt.Fprintln(w, sep)
	for _, row := range rows {
		line(row, true)
	}
	fmt.Fprintln(w, sep)
}

// RenderHistogram draws a horizontal bar chart of the numeric column valCol,
// labelling each bar with the row's non-numeric label (the first column that
// is not a rank number).
func RenderHistogram(w io.Writer, headers []string, rows [][]string, valCol int) {
	if len(rows) == 0 || valCol < 0 || valCol >= len(headers) {
		return
	}
	labels := make([]string, len(rows))
	vals := make([]float64, len(rows))
	labelW, maxVal := 0, 0.0
	for i, row := range rows {
		if valCol >= len(row) {
			return
		}
		v, err := strconv.ParseFloat(strings.TrimSuffix(row[valCol], "%"), 64)
		if err != nil {
			return
		}
		vals[i] = v
		labels[i] = labelFor(row)
		labelW = max(labelW, utf8.RuneCountInString(labels[i]))
		maxVal = max(maxVal, v)
	}

	fmt.Fprintf(w, "\n  %s\n", headers[valCol])
	for i, v := range vals {
		n := 0
		if maxVal > 0 {
			n = int(v / maxVal * barWidth)
		}
		if v > 0 && n == 0 {
			n = 1
		}
		pad := strings.Repeat(" ", labelW-utf8.RuneCountInString(labels[i]))
		fmt.Fprintf(w, "  %s%s | %s %s\n", labels[i], pad, strings.Repeat("#", n), rows[i][valCol])
	}
	fmt.Fprintln(w)
}

// labelFor returns the first non-numeric cell in the row, or the first cell.
func labelFor(row []string) string {
	for _, c := range row {
		if !isNumeric(c) {
			return c
		}
	}
	if len(row) > 0 {
		return row[0]
	}
	return ""
}

func isNumeric(s string) bool {
	s = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(s, "+"), "-"), "%")
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
