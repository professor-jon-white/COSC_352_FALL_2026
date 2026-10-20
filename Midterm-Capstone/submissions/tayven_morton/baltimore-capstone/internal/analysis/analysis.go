// Package analysis joins neighborhood summaries and computes descriptive metrics.
package analysis

import (
	"baltimore-capstone/internal/model"
	"sort"
	"sync"
)

type Row struct {
	Name      string
	Requests  *model.Stats
	Vacancies int
	Matched   bool
	Group     string
	Score     float64
}

// Join uses workers to match immutable input maps. Unmatched neighborhoods stay
// visible but are excluded from comparative cohorts and priority rankings.
func Join(requests, vacancies model.Dataset, workers int) []Row {
	if workers < 1 {
		workers = 1
	}
	keys := map[string]bool{}
	for k := range requests.Areas {
		keys[k] = true
	}
	for k := range vacancies.Areas {
		keys[k] = true
	}
	jobs := make(chan string)
	results := make(chan Row, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range jobs {
				r, rok := requests.Areas[k]
				v, vok := vacancies.Areas[k]
				if !rok {
					r = model.NewStats()
				}
				n := 0
				if vok {
					n = v.Count
				}
				results <- Row{Name: k, Requests: r, Vacancies: n, Matched: rok && vok, Group: "unmatched"}
			}
		}()
	}
	go func() {
		for k := range keys {
			jobs <- k
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	rows := []Row{}
	for r := range results {
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	var counts, requestsCounts []int
	for _, r := range rows {
		if r.Matched {
			counts = append(counts, r.Vacancies)
			requestsCounts = append(requestsCounts, r.Requests.Count)
		}
	}
	sort.Ints(counts)
	sort.Ints(requestsCounts)
	for i := range rows {
		r := &rows[i]
		if !r.Matched {
			continue
		}
		r.Group = "middle"
		if len(counts) >= 3 {
			low, high := counts[(len(counts)-1)/3], counts[2*(len(counts)-1)/3]
			if low < high {
				if r.Vacancies <= low {
					r.Group = "low"
				}
				if r.Vacancies > high {
					r.Group = "high"
				}
			}
		}
		r.Score = (Percentile(counts, r.Vacancies) + Percentile(requestsCounts, r.Requests.Count)) / 2
	}
	return rows
}

// Percentile returns the midrank empirical percentile, treating ties equally.
func Percentile(sorted []int, x int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	lo := sort.SearchInts(sorted, x)
	hi := sort.Search(len(sorted), func(i int) bool { return sorted[i] > x })
	return 100 * (float64(lo) + float64(hi-lo)/2) / float64(len(sorted))
}

func Median(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	v := append([]float64(nil), values...)
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2], true
	}
	return (v[n/2-1] + v[n/2]) / 2, true
}

type Pair struct {
	Name  string
	Count int
}

func Top(m map[string]int, limit int) []Pair {
	p := make([]Pair, 0, len(m))
	for k, v := range m {
		p = append(p, Pair{k, v})
	}
	sort.Slice(p, func(i, j int) bool {
		if p[i].Count == p[j].Count {
			return p[i].Name < p[j].Name
		}
		return p[i].Count > p[j].Count
	})
	if limit < len(p) {
		p = p[:limit]
	}
	return p
}

func Cohort(rows []Row, group string) (*model.Stats, int) {
	total := model.NewStats()
	n := 0
	for _, r := range rows {
		if r.Group != group {
			continue
		}
		n++
		total.Count += r.Requests.Count
		for k, v := range r.Requests.Types {
			total.Types[k] += v
		}
		for k, v := range r.Requests.Months {
			total.Months[k] += v
		}
		for k, v := range r.Requests.Durations {
			total.Durations[k] = append(total.Durations[k], v...)
		}
	}
	return total, n
}
