package main

import (
	"math"
	"sort"
)

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// pearson returns the Pearson correlation coefficient of x and y, or NaN when
// it is undefined (fewer than two points, mismatched lengths, zero variance).
func pearson(x, y []float64) float64 {
	n := len(x)
	if n < 2 || n != len(y) {
		return math.NaN()
	}
	mx, my := mean(x), mean(y)
	var sxy, sxx, syy float64
	for i := 0; i < n; i++ {
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

// ranks returns 1-based ascending ranks; tied values share their average rank.
func ranks(xs []float64) []float64 {
	n := len(xs)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return xs[idx[a]] < xs[idx[b]] })
	out := make([]float64, n)
	for i := 0; i < n; {
		j := i
		for j+1 < n && xs[idx[j+1]] == xs[idx[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			out[idx[k]] = avg
		}
		i = j + 1
	}
	return out
}

// spearman is the rank correlation: Pearson applied to the ranks.
func spearman(x, y []float64) float64 {
	return pearson(ranks(x), ranks(y))
}

// medianF32 returns the median of xs without modifying it (NaN when empty).
func medianF32(xs []float32) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	cp := make([]float32, len(xs))
	copy(cp, xs)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	n := len(cp)
	if n%2 == 1 {
		return float64(cp[n/2])
	}
	return (float64(cp[n/2-1]) + float64(cp[n/2])) / 2
}

// meanF32 returns the arithmetic mean of xs (NaN when empty).
func meanF32(xs []float32) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	sum := 0.0
	for _, x := range xs {
		sum += float64(x)
	}
	return sum / float64(len(xs))
}

// describeCorr turns a correlation coefficient into a plain-English strength.
func describeCorr(r float64) string {
	if math.IsNaN(r) {
		return "undefined"
	}
	a := math.Abs(r)
	var strength string
	switch {
	case a < 0.2:
		strength = "very weak"
	case a < 0.4:
		strength = "weak"
	case a < 0.6:
		strength = "moderate"
	case a < 0.8:
		strength = "strong"
	default:
		strength = "very strong"
	}
	if r < 0 {
		return strength + " negative"
	}
	return strength + " positive"
}
