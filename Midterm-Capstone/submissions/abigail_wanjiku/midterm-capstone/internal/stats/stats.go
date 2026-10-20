// Package stats provides the small set of statistics the reports need.
package stats

import (
	"math"
	"slices"
)

// Ranks returns the 1-based rank of each value in xs, smallest first. Tied
// values share the average of the ranks they span, so [10, 20, 20, 30] ranks
// as [1, 2.5, 2.5, 4].
func Ranks(xs []float64) []float64 {
	order := make([]int, len(xs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		switch {
		case xs[a] < xs[b]:
			return -1
		case xs[a] > xs[b]:
			return 1
		}
		return 0
	})

	ranks := make([]float64, len(xs))
	for i := 0; i < len(order); {
		j := i
		for j+1 < len(order) && xs[order[j+1]] == xs[order[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1 // ranks i+1 .. j+1
		for k := i; k <= j; k++ {
			ranks[order[k]] = avg
		}
		i = j + 1
	}
	return ranks
}

// Pearson returns the Pearson correlation of xs and ys, or NaN if either has
// no variance or the lengths differ.
func Pearson(xs, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) < 2 {
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

// Spearman returns the Spearman rank correlation of xs and ys: the Pearson
// correlation of their ranks. It measures whether ys tends to rise with xs
// without assuming a straight-line relationship, and outliers cannot dominate it.
func Spearman(xs, ys []float64) float64 {
	return Pearson(Ranks(xs), Ranks(ys))
}

// Mean returns the arithmetic mean of xs, or NaN if xs is empty.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// Percentile returns the p-th percentile (0-100) of xs using linear
// interpolation between closest ranks, or NaN if xs is empty.
func Percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	pos := p / 100 * float64(len(s)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	return s[lo] + (s[hi]-s[lo])*(pos-float64(lo))
}

// Strength describes the size of a correlation coefficient in words.
func Strength(r float64) string {
	a := math.Abs(r)
	dir := "positive"
	if r < 0 {
		dir = "negative"
	}
	switch {
	case math.IsNaN(r):
		return "undefined"
	case a < 0.1:
		return "no meaningful"
	case a < 0.3:
		return "weak " + dir
	case a < 0.5:
		return "moderate " + dir
	case a < 0.7:
		return "strong " + dir
	}
	return "very strong " + dir
}
