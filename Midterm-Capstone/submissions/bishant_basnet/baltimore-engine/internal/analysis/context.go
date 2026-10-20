// Package analysis performs the cross-dataset join and answers the five
// stakeholder research questions.
package analysis

import (
	"sort"
	"strings"
	"sync"

	"github.com/bishant/baltimore-engine/internal/model"
)

// HoodStats is the joined, per-neighborhood view of both datasets.
type HoodStats struct {
	Name          string
	Crimes        int
	Requests      int
	Blight        int       // blight/sanitation-type 311 requests
	ResolutionHrs []float32 // float32 halves memory vs float64; precision is ample for hours
}

// Context is the immutable, joined dataset handed to every question.
// Because nothing mutates it after BuildContext returns, the five questions
// can read it concurrently with no locks.
type Context struct {
	Hoods          map[string]*HoodStats
	Joined         []*HoodStats // neighborhoods present in BOTH datasets, sorted by name
	CrimeByHour    [24]int
	ReqByHour      [24]int
	CrimeTimed     int
	ReqTimed       int
	TotalCrimes    int
	TotalRequests  int
	JoinedCrimes   int
	JoinedRequests int
	TotalBlight    int
	TopN           int
}

var blightWords = map[string]bool{
	"RAT": true, "RATS": true, "RODENT": true, "RODENTS": true, "TRASH": true,
	"DIRTY": true, "VACANT": true, "GRAFFITI": true, "DUMPING": true, "DUMP": true,
	"BULK": true, "SANITATION": true, "WEEDS": true, "GRASS": true,
	"ABANDONED": true, "LITTER": true, "BOARDING": true,
}

// IsBlight classifies a 311 SR type using whole-word matching (so that
// e.g. "OPERATIONS" does not match "RAT").
func IsBlight(srType string) bool {
	for _, w := range strings.FieldsFunc(strings.ToUpper(srType), func(r rune) bool {
		return r < 'A' || r > 'Z'
	}) {
		if blightWords[w] {
			return true
		}
	}
	return false
}

type partial struct {
	hoods                map[string]*HoodStats
	crimeHour, reqHour   [24]int
	crimeTimed, reqTimed int
}

func chunk(n, k, i int) (int, int) {
	size := (n + k - 1) / k
	lo, hi := i*size, (i+1)*size
	if lo > n {
		lo = n
	}
	if hi > n {
		hi = n
	}
	return lo, hi
}

// BuildContext joins the datasets on neighborhood using a lock-free
// map-reduce: each worker aggregates a disjoint slice into its OWN map
// (no shared writes, so no mutex contention), then the partials are merged.
func BuildContext(crimes []model.CrimeRecord, reqs []model.ServiceRequest, workers, topN int) *Context {
	if workers < 1 {
		workers = 1
	}
	parts := make([]*partial, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			p := &partial{hoods: make(map[string]*HoodStats, 300)}
			get := func(n string) *HoodStats {
				h := p.hoods[n]
				if h == nil {
					h = &HoodStats{Name: n}
					p.hoods[n] = h
				}
				return h
			}
			lo, hi := chunk(len(crimes), workers, w)
			for i := lo; i < hi; i++ {
				c := &crimes[i] // pointer avoids copying the struct
				get(c.Neighborhood).Crimes++
				if c.HasTime {
					p.crimeHour[c.When.Hour()]++
					p.crimeTimed++
				}
			}
			lo, hi = chunk(len(reqs), workers, w)
			for i := lo; i < hi; i++ {
				r := &reqs[i]
				h := get(r.Neighborhood)
				h.Requests++
				if IsBlight(r.Type) {
					h.Blight++
				}
				if r.HasCreated {
					p.reqHour[r.Created.Hour()]++
					p.reqTimed++
					if r.HasClosed && !r.Closed.Before(r.Created) {
						h.ResolutionHrs = append(h.ResolutionHrs, float32(r.Closed.Sub(r.Created).Hours()))
					}
				}
			}
			parts[w] = p
		}(w)
	}
	wg.Wait()

	ctx := &Context{Hoods: make(map[string]*HoodStats), TopN: topN,
		TotalCrimes: len(crimes), TotalRequests: len(reqs)}
	for _, p := range parts {
		for i := 0; i < 24; i++ {
			ctx.CrimeByHour[i] += p.crimeHour[i]
			ctx.ReqByHour[i] += p.reqHour[i]
		}
		ctx.CrimeTimed += p.crimeTimed
		ctx.ReqTimed += p.reqTimed
		for name, h := range p.hoods {
			f := ctx.Hoods[name]
			if f == nil {
				ctx.Hoods[name] = h
				continue
			}
			f.Crimes += h.Crimes
			f.Requests += h.Requests
			f.Blight += h.Blight
			f.ResolutionHrs = append(f.ResolutionHrs, h.ResolutionHrs...)
		}
	}
	for _, h := range ctx.Hoods {
		if h.Crimes > 0 && h.Requests > 0 {
			ctx.Joined = append(ctx.Joined, h)
			ctx.JoinedCrimes += h.Crimes
			ctx.JoinedRequests += h.Requests
		}
		ctx.TotalBlight += h.Blight
	}
	sort.Slice(ctx.Joined, func(a, b int) bool { return ctx.Joined[a].Name < ctx.Joined[b].Name })
	return ctx
}
