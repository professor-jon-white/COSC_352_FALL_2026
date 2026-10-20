package analysis

import (
	"math"
	"sort"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/report"
)

// Analyzer is the Strategy interface: one implementation per research question.
type Analyzer interface {
	Name() string
	Stakeholder() string
	Run(d *join.Dataset) report.Section
}

// All lists the analyzers to run, in order.
func All() []Analyzer {
	return []Analyzer{TopBoth{}, SanitationVsViolence{}, Underreporting{}, HourOfDay{}, MonthlyTrend{}}
}

type entry struct {
	Name  string
	Stats *join.Stats
}

// entries returns neighborhoods sorted by name, so output is deterministic.
func entries(d *join.Dataset) []entry {
	out := make([]entry, 0, len(d.ByHood))
	for n, s := range d.ByHood {
		out = append(out, entry{n, s})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// rankBy returns each neighborhood's rank (1 = highest value).
func rankBy(es []entry, val func(*join.Stats) int) map[string]int {
	tmp := make([]entry, len(es))
	copy(tmp, es)
	sort.SliceStable(tmp, func(i, j int) bool { return val(tmp[i].Stats) > val(tmp[j].Stats) })
	r := make(map[string]int, len(tmp))
	for i, e := range tmp {
		r[e.Name] = i + 1
	}
	return r
}

// pearson computes the correlation coefficient r between x and y.
func pearson(x, y []float64) float64 {
	n := float64(len(x))
	if n < 2 {
		return 0
	}
	var sx, sy float64
	for i := range x {
		sx += x[i]
		sy += y[i]
	}
	mx, my := sx/n, sy/n
	var num, dx, dy float64
	for i := range x {
		a, b := x[i]-mx, y[i]-my
		num += a * b
		dx += a * a
		dy += b * b
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return num / math.Sqrt(dx*dy)
}

// minCrimes scales a "big enough neighborhood" cutoff to the dataset size,
// so the analyzers behave sensibly on both the full data and the small sample.
// On the full 2024 data, divisor 400 gives about 100 and divisor 146 about 300.
func minCrimes(d *join.Dataset, divisor int) int {
	total := 0
	for _, s := range d.ByHood {
		total += s.Crime
	}
	m := total / divisor
	if m < 5 {
		m = 5
	}
	return m
}
