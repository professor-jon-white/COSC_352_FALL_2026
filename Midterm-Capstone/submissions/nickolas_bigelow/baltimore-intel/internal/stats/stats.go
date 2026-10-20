// Package stats holds the small set of statistics the analyses need.
package stats

import (
	"math"
	"sort"
)

func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func StdDev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m, s := Mean(xs), 0.0
	for _, x := range xs {
		s += (x - m) * (x - m)
	}
	return math.Sqrt(s / float64(len(xs)-1))
}

// Quantile uses linear interpolation; it sorts a copy so callers' data is untouched.
func Quantile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	pos := q * float64(len(s)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	return s[lo] + (s[hi]-s[lo])*(pos-float64(lo))
}

func Median(xs []float64) float64 { return Quantile(xs, 0.5) }

// ZScores standardizes xs; a constant series maps to zeros.
func ZScores(xs []float64) []float64 {
	m, sd := Mean(xs), StdDev(xs)
	out := make([]float64, len(xs))
	if sd == 0 {
		return out
	}
	for i, x := range xs {
		out[i] = (x - m) / sd
	}
	return out
}

func Pearson(xs, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) < 3 {
		return math.NaN()
	}
	mx, my := Mean(xs), Mean(ys)
	var sxy, sxx, syy float64
	for i := range xs {
		dx, dy := xs[i]-mx, ys[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return math.NaN()
	}
	return sxy / math.Sqrt(sxx*syy)
}

// Ranks assigns average ranks to ties (needed for Spearman).
func Ranks(xs []float64) []float64 {
	idx := make([]int, len(xs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return xs[idx[a]] < xs[idx[b]] })
	r := make([]float64, len(xs))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && xs[idx[j+1]] == xs[idx[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			r[idx[k]] = avg
		}
		i = j + 1
	}
	return r
}

func Spearman(xs, ys []float64) float64 { return Pearson(Ranks(xs), Ranks(ys)) }

// LinearFit returns slope and intercept of the least-squares line y = a + b*x.
func LinearFit(xs, ys []float64) (intercept, slope float64) {
	mx, my := Mean(xs), Mean(ys)
	var sxy, sxx float64
	for i := range xs {
		sxy += (xs[i] - mx) * (ys[i] - my)
		sxx += (xs[i] - mx) * (xs[i] - mx)
	}
	if sxx == 0 {
		return my, 0
	}
	slope = sxy / sxx
	return my - slope*mx, slope
}

// Describe a correlation coefficient in plain words for stakeholders.
func Strength(r float64) string {
	a := math.Abs(r)
	switch {
	case math.IsNaN(r):
		return "undefined"
	case a >= 0.7:
		return "strong"
	case a >= 0.4:
		return "moderate"
	case a >= 0.2:
		return "weak"
	default:
		return "negligible"
	}
}
