package report

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"

	"capstone/internal/stats"
)

// barWidth is the length in characters of a full-scale bar.
const barWidth = 16

// bar draws v as a run of '#' scaled so that full fills barWidth. Any positive
// value gets at least one '#', so small-but-nonzero values stay visible.
func bar(v, full float64) string {
	if full <= 0 || v <= 0 || math.IsNaN(v) {
		return ""
	}
	n := int(math.Round(v / full * barWidth))
	return strings.Repeat("#", min(max(n, 1), barWidth))
}

// table is a thin wrapper over tabwriter for aligned, header-ruled tables.
type table struct {
	tw *tabwriter.Writer
}

func newTable(w io.Writer, headers ...string) *table {
	t := &table{tw: tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)}
	t.row(toAny(headers)...)
	rules := make([]any, len(headers))
	for i, h := range headers {
		rules[i] = strings.Repeat("-", len(h))
	}
	t.row(rules...)
	return t
}

func (t *table) row(cells ...any) {
	for _, c := range cells {
		fmt.Fprintf(t.tw, "%v\t", c)
	}
	fmt.Fprintln(t.tw)
}

func (t *table) flush() error { return t.tw.Flush() }

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// num formats an integer with thousands separators: 23676 -> "23,676".
func num(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return "-" + num(-n)
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// corr formats a correlation coefficient with its strength in words,
// e.g. "+0.62 (strong positive)".
func corr(r float64) string {
	if math.IsNaN(r) {
		return "n/a"
	}
	return fmt.Sprintf("%+.2f (%s)", r, stats.Strength(r))
}
