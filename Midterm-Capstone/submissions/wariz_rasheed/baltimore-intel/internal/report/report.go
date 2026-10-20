// Package report turns analysis results into output. It knows nothing about
// crime, 311 or licenses: it only renders generic tables and bar charts.
package report

import (
	"fmt"
	"io"
	"math"
)

// Bar is one bar of a horizontal bar chart.
type Bar struct {
	Label string
	Value float64
	Text  string // value as displayed, e.g. "12.4 (n=310)"
}

// Chart is a horizontal bar chart. Negative values are drawn with a different
// glyph so correlations below zero remain visible.
type Chart struct {
	Title string
	Bars  []Bar
}

// Table is a simple grid of pre-formatted strings.
type Table struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// Result is the answer to one research question.
type Result struct {
	ID       string
	Title    string
	Audience string // the municipal stakeholder this question is for
	Question string
	Finding  string // one-paragraph headline answer
	Charts   []Chart
	Tables   []Table
	Notes    []string // caveats and method notes
}

// Summary is the complete report.
type Summary struct {
	Title    string
	Subtitle string
	Overview Table
	Results  []Result
	Footer   []string
}

// Renderer writes a Summary in one output format.
type Renderer interface {
	Render(w io.Writer, s Summary) error
}

// New is the factory for renderers: "terminal" or "html".
func New(kind string) (Renderer, error) {
	switch kind {
	case "terminal":
		return terminal{}, nil
	case "html":
		return htmlRenderer{}, nil
	}
	return nil, fmt.Errorf("unknown report format %q", kind)
}

// barLen scales |v| to a bar length in [0, width]. Non-zero values always get
// at least one cell so they are visible.
func barLen(v, max float64, width int) int {
	if max <= 0 || v == 0 {
		return 0
	}
	n := int(math.Round(math.Abs(v) / max * float64(width)))
	if n < 1 {
		n = 1
	}
	return n
}

// maxAbs returns the largest absolute value among the bars.
func maxAbs(bars []Bar) float64 {
	m := 0.0
	for _, b := range bars {
		if a := math.Abs(b.Value); a > m {
			m = a
		}
	}
	return m
}
