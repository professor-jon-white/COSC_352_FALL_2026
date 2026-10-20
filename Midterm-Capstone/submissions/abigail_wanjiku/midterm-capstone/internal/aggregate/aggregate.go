// Package aggregate rolls neighborhood-month summaries up into coarser views.
package aggregate

import (
	"cmp"
	"slices"

	"capstone/internal/model"
)

// ByNeighborhood merges every month of each neighborhood into one Summary
// whose Key.Month is zero. Results are sorted by neighborhood.
func ByNeighborhood(summaries []model.Summary) []model.Summary {
	merged := make(map[string]*model.Summary, len(summaries)/12+1) // ~12 months per neighborhood
	for i := range summaries {
		s := &summaries[i]
		m, ok := merged[s.Key.Neighborhood]
		if !ok {
			m = model.NewSummary(model.Key{Neighborhood: s.Key.Neighborhood})
			merged[s.Key.Neighborhood] = m
		}
		m.Merge(s)
	}

	out := make([]model.Summary, 0, len(merged))
	for _, m := range merged {
		out = append(out, *m)
	}
	slices.SortFunc(out, func(a, b model.Summary) int {
		return cmp.Compare(a.Key.Neighborhood, b.Key.Neighborhood)
	})
	return out
}

// TopByCrime returns the n summaries with the highest CrimeCount, breaking
// ties by neighborhood name. The input is not modified.
func TopByCrime(summaries []model.Summary, n int) []model.Summary {
	out := slices.Clone(summaries)
	slices.SortFunc(out, func(a, b model.Summary) int {
		return cmp.Or(
			cmp.Compare(b.CrimeCount, a.CrimeCount),
			cmp.Compare(a.Key.Neighborhood, b.Key.Neighborhood),
		)
	})
	return out[:min(n, len(out))]
}

// MostCommon returns the key with the highest count in m and its count,
// breaking ties alphabetically. It returns "", 0 for an empty map.
func MostCommon(m map[string]int) (string, int) {
	var best string
	var bestN int
	for k, n := range m {
		if n > bestN || (n == bestN && k < best) {
			best, bestN = k, n
		}
	}
	return best, bestN
}
