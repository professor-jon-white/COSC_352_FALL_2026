// Package match joins crimes and 311 requests into one combined record per
// (neighborhood, month).
package match

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"capstone/internal/model"
)

// clockSkew is how far ClosedAt may precede CreatedAt and still count as a
// same-moment close. Many 311 close times are truncated to the minute, so a
// request created at 2:37:43 PM can show as closed at 2:37:00 PM.
const clockSkew = time.Hour

// IsNight reports whether t (Baltimore time) falls between 8 PM and 6 AM.
// Times of exactly midnight are excluded: 00:00:00 is far more common than
// any other minute in the crime data, which suggests it often stands in for
// an unknown time of day.
func IsNight(t time.Time) bool {
	h, m, s := t.Clock()
	if h == 0 && m == 0 && s == 0 {
		return false
	}
	return h >= 20 || h < 6
}

// ByNeighborhoodMonth groups crimes and requests by model.Key and returns one
// Summary per key that appears in either dataset (a full outer join), sorted
// by neighborhood then month. Inputs are expected to be filtered to one year
// and normalized by the ingest stage.
//
// A request counts toward the month it was created in, including its time to
// close, even if it closed in a later month.
func ByNeighborhoodMonth(crimes []model.Crime, requests []model.ServiceRequest) []model.Summary {
	b := buckets{}
	for i := range crimes {
		b.addCrime(&crimes[i])
	}
	for i := range requests {
		b.addRequest(&requests[i])
	}
	out := b.summaries(nil)
	sortSummaries(out)
	return out
}

// ByNeighborhoodMonthConcurrent returns the same result as
// ByNeighborhoodMonth, computed by workers goroutines.
//
// Work is sharded by neighborhood: worker w handles only the neighborhoods
// for which shard(name) == w. Every record of a neighborhood, and so every
// month of it, lands in exactly one worker, which means each Summary is built
// by one goroutine from start to finish. Workers share only the input slices,
// which nobody writes, so no locks are needed.
func ByNeighborhoodMonthConcurrent(crimes []model.Crime, requests []model.ServiceRequest, workers int) []model.Summary {
	workers = max(workers, 1)
	parts := make([][]model.Summary, workers) // parts[w] is written only by worker w

	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := buckets{} // private to this worker
			for i := range crimes {
				if shard(crimes[i].Neighborhood, workers) == w {
					b.addCrime(&crimes[i])
				}
			}
			for i := range requests {
				if shard(requests[i].Neighborhood, workers) == w {
					b.addRequest(&requests[i])
				}
			}
			parts[w] = b.summaries(nil)
		}()
	}
	wg.Wait() // every write to parts happens before this returns

	// Shards hold disjoint keys, so merging is just concatenation; sorting
	// then gives the same order as the sequential version.
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]model.Summary, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	sortSummaries(out)
	return out
}

// shard maps a neighborhood to a worker in [0, n) using FNV-1a, so the same
// name always goes to the same worker without any shared lookup table.
func shard(neighborhood string, n int) int {
	h := uint32(2166136261)
	for i := 0; i < len(neighborhood); i++ {
		h ^= uint32(neighborhood[i])
		h *= 16777619
	}
	return int(h % uint32(n))
}

// buckets accumulates one Summary per key. It is not safe for concurrent use;
// the concurrent matcher gives each worker its own.
type buckets map[model.Key]*model.Summary

func (b buckets) get(k model.Key) *model.Summary {
	s, ok := b[k]
	if !ok {
		s = model.NewSummary(k)
		b[k] = s
	}
	return s
}

func (b buckets) addCrime(c *model.Crime) {
	s := b.get(model.KeyFor(c.Neighborhood, c.OccurredAt))
	s.CrimeCount++
	s.CrimesByType[c.CrimeType]++
	if c.Outdoor && IsNight(c.OccurredAt) {
		s.NightOutdoorCrimes++
	}
}

func (b buckets) addRequest(r *model.ServiceRequest) {
	s := b.get(model.KeyFor(r.Neighborhood, r.CreatedAt))
	s.RequestCount++
	s.RequestsByType[r.RequestType]++
	if r.ClosedAt.IsZero() {
		return // still open
	}
	d := r.ClosedAt.Sub(r.CreatedAt)
	switch {
	case d < -clockSkew:
		s.BadCloseTimes++
		return
	case d < 0:
		d = 0
	}
	days := d.Hours() / 24
	s.ClosedCount++
	s.CloseDaysTotal += days
	s.ClosedByType[r.RequestType]++
	s.CloseDaysByType[r.RequestType] += days
}

// summaries appends every Summary in b to out, in no particular order.
func (b buckets) summaries(out []model.Summary) []model.Summary {
	out = slices.Grow(out, len(b))
	for _, s := range b {
		out = append(out, *s)
	}
	return out
}

func sortSummaries(s []model.Summary) {
	slices.SortFunc(s, func(a, b model.Summary) int {
		return cmp.Or(
			cmp.Compare(a.Key.Neighborhood, b.Key.Neighborhood),
			a.Key.Month.Compare(b.Key.Month),
		)
	})
}
