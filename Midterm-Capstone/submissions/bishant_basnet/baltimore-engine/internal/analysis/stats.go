package analysis

import (
	"math"
	"sort"
)

// Pearson correlation coefficient.
func Pearson(x, y []float64) float64 {
	n := len(x)
	if n < 2 || n != len(y) {
		return math.NaN()
	}
	var mx, my float64
	for i := range x {
		mx += x[i]
		my += y[i]
	}
	mx /= float64(n)
	my /= float64(n)
	var sxy, sxx, syy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return math.NaN()
	}
	return sxy / math.Sqrt(sxx*syy)
}

// Percentile with linear interpolation (p in 0..100). Does not mutate v.
func Percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	pos := p / 100 * float64(len(c)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	return c[lo] + (c[hi]-c[lo])*(pos-float64(lo))
}

func Median(v []float64) float64 { return Percentile(v, 50) }

// DescribeR turns a correlation into plain English for non-technical readers.
func DescribeR(r float64) string {
	if math.IsNaN(r) {
		return "undefined (not enough variation)"
	}
	a := math.Abs(r)
	dir := "positive"
	if r < 0 {
		dir = "negative"
	}
	switch {
	case a >= 0.7:
		return "strong " + dir
	case a >= 0.4:
		return "moderate " + dir
	case a >= 0.2:
		return "weak " + dir
	}
	return "negligible"
}

// quartiles sorts a COPY (the shared Context is never mutated) and splits it
// into four nearly equal groups, lowest key first.
func quartiles(h []*HoodStats, key func(*HoodStats) float64) [4][]*HoodStats {
	c := append([]*HoodStats(nil), h...)
	sort.SliceStable(c, func(a, b int) bool { return key(c[a]) < key(c[b]) })
	var q [4][]*HoodStats
	n := len(c)
	for i := 0; i < 4; i++ {
		q[i] = c[i*n/4 : (i+1)*n/4]
	}
	return q
}

func pct(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return 100 * float64(part) / float64(whole)
}
