// Package report separates *what* an analysis found (a renderer-neutral tree
// of blocks) from *how* it is shown. Renderers implement the Strategy
// pattern: the same Report goes to the terminal, Markdown, HTML or CSV.
package report

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

// Report is the whole output of one run.
type Report struct {
	Title     string
	Generated time.Time
	DataNote  string
	Inputs    [][2]string // label, value
	Sections  []*Section
	Runtime   [][2]string
}

// Section answers one research question.
type Section struct {
	ID          string
	Stakeholder string
	Question    string
	Datasets    []string
	Finding     string // headline answer, one or two sentences
	Blocks      []Block
	Actions     []string // recommended actions
	Notes       []string // caveats / method
}

// Block is any renderable element.
type Block interface{ blockTitle() string }

type Table struct {
	Title   string
	Headers []string
	Rows    [][]string
	Right   []bool // right-align column i (numbers)
	CSVName string // file name when exported
}

type Bars struct {
	Title  string
	Labels []string
	Values []float64
	Format string // fmt verb for values, e.g. "%.0f"
	Unit   string
}

type Heatmap struct {
	Title     string
	RowLabels []string
	ColLabels []string
	Values    [][]float64
}

type Scatter struct {
	Title          string
	XLabel, YLabel string
	X, Y           []float64
	Labels         []string // optional labels for the most extreme points
}

type KeyValues struct {
	Title string
	Pairs [][2]string
}

func (t *Table) blockTitle() string     { return t.Title }
func (b *Bars) blockTitle() string      { return b.Title }
func (h *Heatmap) blockTitle() string   { return h.Title }
func (s *Scatter) blockTitle() string   { return s.Title }
func (k *KeyValues) blockTitle() string { return k.Title }

// Renderer is the output Strategy.
type Renderer interface {
	Name() string
	Render(w io.Writer, r *Report) error
}

// Formatters ----------------------------------------------------------------

func Int(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func F(v float64, prec int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "n/a"
	}
	return fmt.Sprintf("%.*f", prec, v)
}

func Pct(v float64) string {
	if math.IsNaN(v) {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", v*100)
}
