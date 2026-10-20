// Package stats holds the small statistical helpers the analyses share.
package stats

import (
	"math"
	"sort"
)

// Mean returns the arithmetic mean, or 0 for an empty slice.
func Mean(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	var s float64
	for _, v := range x {
		s += v
	}
	return s / float64(len(x))
}

// Median returns the median without modifying its input (it sorts a copy, so
// the same slice can be shared by goroutines).
func Median(x []float64) float64 {
	n := len(x)
	if n == 0 {
		return 0
	}
	c := make([]float64, n)
	copy(c, x)
	sort.Float64s(c)
	if n%2 == 1 {
		return c[n/2]
	}
	return (c[n/2-1] + c[n/2]) / 2
}

// Pearson returns the Pearson correlation coefficient, or 0 if undefined.
func Pearson(x, y []float64) float64 {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	if n < 2 {
		return 0
	}
	mx, my := Mean(x[:n]), Mean(y[:n])
	var sxy, sxx, syy float64
	for i := 0; i < n; i++ {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return 0
	}
	return sxy / math.Sqrt(sxx*syy)
}

// Ranks returns 1-based ranks; tied values share their average rank.
func Ranks(x []float64) []float64 {
	n := len(x)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return x[idx[a]] < x[idx[b]] })
	r := make([]float64, n)
	for i := 0; i < n; {
		j := i
		for j+1 < n && x[idx[j+1]] == x[idx[i]] {
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

// Spearman returns the Spearman rank correlation, which is robust to the
// heavy skew of neighborhood counts.
func Spearman(x, y []float64) float64 {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	if n < 2 {
		return 0
	}
	return Pearson(Ranks(x[:n]), Ranks(y[:n]))
}

// PartialSpearman returns the Spearman correlation between x and y after
// removing the part of each that is explained by z (for example, overall
// neighborhood size). It returns 0 when the adjustment is undefined.
func PartialSpearman(x, y, z []float64) float64 {
	rxy := Spearman(x, y)
	rxz := Spearman(x, z)
	ryz := Spearman(y, z)
	d := (1 - rxz*rxz) * (1 - ryz*ryz)
	if d <= 0 {
		return 0
	}
	return (rxy - rxz*ryz) / math.Sqrt(d)
}

// PercentileRanks maps each value to 0-100 by its rank within the slice.
func PercentileRanks(x []float64) []float64 {
	n := len(x)
	out := make([]float64, n)
	if n == 0 {
		return out
	}
	if n == 1 {
		out[0] = 50
		return out
	}
	r := Ranks(x)
	for i, v := range r {
		out[i] = (v - 1) / float64(n-1) * 100
	}
	return out
}

// Describe turns a correlation coefficient into plain language.
func Describe(rho float64) string {
	a := math.Abs(rho)
	var strength string
	switch {
	case a < 0.1:
		return "no meaningful relationship"
	case a < 0.3:
		strength = "weak"
	case a < 0.5:
		strength = "moderate"
	case a < 0.7:
		strength = "strong"
	default:
		strength = "very strong"
	}
	if rho < 0 {
		return strength + " negative relationship"
	}
	return strength + " positive relationship"
}
