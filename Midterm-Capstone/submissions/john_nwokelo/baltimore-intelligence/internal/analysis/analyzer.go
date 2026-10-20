package analysis

import (
	"baltimore-intelligence/internal/models"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type Stats struct {
	Name                string
	Total, Open, Closed int
	Category            map[string]int
	DurationSum         float64
	DurationN           int
	Priority            float64
}

func (s Stats) Rate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Open) / float64(s.Total) * 100
}
func (s Stats) AvgDays() float64 {
	if s.DurationN == 0 {
		return 0
	}
	return s.DurationSum / float64(s.DurationN)
}
func (s Stats) Common() (string, int) {
	best := ""
	max := 0
	for k, v := range s.Category {
		if v > max || (v == max && k < best) {
			best, max = k, v
		}
	}
	return best, max
}
func isClosed(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "closed" || s == "resolved" || s == "complete" || s == "completed"
}
func Analyze(items []models.Joined, workers int) []Stats {
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	if workers > len(items) && len(items) > 0 {
		workers = len(items)
	}
	if len(items) == 0 {
		return nil
	}
	parts := make(chan map[string]*Stats, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		from := i * len(items) / workers
		to := (i + 1) * len(items) / workers
		wg.Add(1)
		go func(segment []models.Joined) {
			defer wg.Done()
			local := map[string]*Stats{}
			for _, j := range segment {
				if j.Neighborhood == "" {
					continue
				}
				s := local[j.Neighborhood]
				if s == nil {
					s = &Stats{Name: j.Neighborhood, Category: map[string]int{}}
					local[j.Neighborhood] = s
				}
				s.Total++
				cat := strings.TrimSpace(j.Request.Category)
				if cat == "" {
					cat = "Unknown"
				}
				s.Category[cat]++
				if isClosed(j.Request.Status) {
					s.Closed++
					if j.Request.Created > 0 && j.Request.Closed >= j.Request.Created {
						s.DurationSum += float64(j.Request.Closed-j.Request.Created) / 86400000
						s.DurationN++
					}
				} else {
					s.Open++
				}
			}
			parts <- local
		}(items[from:to])
	}
	go func() { wg.Wait(); close(parts) }()
	merged := map[string]*Stats{}
	for part := range parts {
		for name, s := range part {
			dst := merged[name]
			if dst == nil {
				dst = &Stats{Name: name, Category: map[string]int{}}
				merged[name] = dst
			}
			dst.Total += s.Total
			dst.Open += s.Open
			dst.Closed += s.Closed
			dst.DurationSum += s.DurationSum
			dst.DurationN += s.DurationN
			for k, v := range s.Category {
				dst.Category[k] += v
			}
		}
	}
	out := make([]Stats, 0, len(merged))
	maxTotal, maxAvg, totalDurations := 0, 0.0, 0
	for _, s := range merged {
		totalDurations += s.DurationN
		if s.Total > maxTotal {
			maxTotal = s.Total
		}
		if s.AvgDays() > maxAvg {
			maxAvg = s.AvgDays()
		}
	}
	for _, s := range merged {
		// When no valid resolution intervals exist, renormalize the two
		// measurable components. Never present a missing time metric as zero.
		volumeWeight, unresolvedWeight := 40.0, 40.0
		if totalDurations == 0 {
			volumeWeight, unresolvedWeight = 50.0, 50.0
		}
		if maxTotal > 0 {
			s.Priority += volumeWeight * float64(s.Total) / float64(maxTotal)
		}
		s.Priority += unresolvedWeight * s.Rate() / 100
		if totalDurations > 0 && maxAvg > 0 {
			s.Priority += 20 * s.AvgDays() / maxAvg
		}
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
