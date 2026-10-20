// Package analytics answers municipal stakeholder questions from joined data.
package analytics

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"baltimore-engine/processor"
)

// Result is a generic tabular answer to one question.
type Result struct {
	Question string
	Headers  []string
	Rows     [][]string
	Note     string // optional caveat printed under the table
	BarCol   int    // 1-based numeric column to also draw as a histogram; 0 = none
}

// RunAll runs every stakeholder question concurrently against the joined data
// and returns results in question order. The joined data is only read, so the
// goroutines need no locking; each writes to its own slot.
func RunAll(j processor.Joined) []Result {
	questions := []func(processor.Joined) Result{Q1, Q2, Q3, Q4, Q5}
	results := make([]Result, len(questions))

	var wg sync.WaitGroup
	for i, q := range questions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = q(j)
		}()
	}
	wg.Wait()
	return results
}

// sortedHoods returns neighborhoods ordered by less, ties broken by name so
// output is deterministic.
func sortedHoods(j processor.Joined, less func(a, b *processor.NeighborhoodStats) bool) []*processor.NeighborhoodStats {
	out := make([]*processor.NeighborhoodStats, 0, len(j.Neighborhoods))
	for _, n := range j.Neighborhoods {
		out = append(out, n)
	}
	sort.Slice(out, func(a, b int) bool {
		if less(out[a], out[b]) {
			return true
		}
		if less(out[b], out[a]) {
			return false
		}
		return out[a].Name < out[b].Name
	})
	return out
}

// pearson returns the Pearson correlation coefficient of xs and ys, or NaN if
// it is undefined (fewer than 2 points or zero variance).
func pearson(xs, ys []float64) float64 {
	n := float64(len(xs))
	if n < 2 {
		return math.NaN()
	}
	var sx, sy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
	}
	mx, my := sx/n, sy/n
	var cov, vx, vy float64
	for i := range xs {
		dx, dy := xs[i]-mx, ys[i]-my
		cov += dx * dy
		vx += dx * dx
		vy += dy * dy
	}
	if vx == 0 || vy == 0 {
		return math.NaN()
	}
	return cov / math.Sqrt(vx*vy)
}

func pct(part, whole int) string {
	if whole == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(part)/float64(whole))
}
