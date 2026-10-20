// Package processor joins the 311 and vacant building datasets.
package processor

import (
	"context"
	"runtime"
	"strings"
	"sync"

	"baltimore-engine/ingest"
)

// chunkSize is the number of records each worker aggregates at a time.
const chunkSize = 2000

// NeighborhoodStats is the joined view of both datasets for one neighborhood.
type NeighborhoodStats struct {
	Name       string
	Requests   int            // 311 requests
	Open       int            // 311 requests not yet closed
	Closed     int            // 311 requests closed
	ByCategory map[string]int // 311 requests per Category()
	Vacant     int            // open vacant building notices
	Zip        string         // dominant ZIP of the neighborhood's 311 requests
	zipCounts  map[string]int
}

// ZipStats is the per-ZIP view. Vacancies have no ZIP in the source data, so
// they are attributed through each neighborhood's dominant ZIP.
type ZipStats struct {
	Zip             string
	Requests        int
	Open            int
	Enforcement     int // all code-enforcement 311 requests
	OpenEnforcement int // enforcement requests still open
	Vacant          int
}

// Joined is the output of the processor stage.
type Joined struct {
	Neighborhoods map[string]*NeighborhoodStats
	Zips          map[string]*ZipStats
}

func newJoined() Joined {
	return Joined{
		Neighborhoods: map[string]*NeighborhoodStats{},
		Zips:          map[string]*ZipStats{},
	}
}

func key(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func (j Joined) hood(name string) *NeighborhoodStats {
	n, ok := j.Neighborhoods[name]
	if !ok {
		n = &NeighborhoodStats{Name: name, ByCategory: map[string]int{}, zipCounts: map[string]int{}}
		j.Neighborhoods[name] = n
	}
	return n
}

func (j Joined) zip(z string) *ZipStats {
	s, ok := j.Zips[z]
	if !ok {
		s = &ZipStats{Zip: z}
		j.Zips[z] = s
	}
	return s
}

// Join aggregates both datasets by neighborhood and ZIP.
//
// Both datasets are split into chunks; a bounded pool of worker goroutines
// aggregates each chunk into a private partial Joined and sends it on a
// channel. The calling goroutine is the only one that merges, so no mutexes
// are needed.
func Join(ctx context.Context, srs []ingest.ServiceRequest, vbs []ingest.VacantBuilding) Joined {
	partials := make(chan Joined)
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup

	spawn := func(work func() Joined) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			partials <- work()
		}()
	}

	for lo := 0; lo < len(srs); lo += chunkSize {
		chunk := srs[lo:min(lo+chunkSize, len(srs))]
		spawn(func() Joined { return aggregateRequests(chunk) })
	}
	for lo := 0; lo < len(vbs); lo += chunkSize {
		chunk := vbs[lo:min(lo+chunkSize, len(vbs))]
		spawn(func() Joined { return aggregateVacancies(chunk) })
	}
	go func() {
		wg.Wait()
		close(partials)
	}()

	out := newJoined()
	for p := range partials {
		out.merge(p)
	}
	out.finalize()
	return out
}

func aggregateRequests(chunk []ingest.ServiceRequest) Joined {
	j := newJoined()
	for _, r := range chunk {
		closed := IsClosed(r.SRStatus)
		enforce := IsEnforcement(r.SRType)

		if name := key(r.Neighborhood); name != "" {
			n := j.hood(name)
			n.Requests++
			if closed {
				n.Closed++
			} else {
				n.Open++
			}
			n.ByCategory[Category(r.SRType)]++
			if z := strings.TrimSpace(r.ZipCode); z != "" {
				n.zipCounts[z]++
			}
		}
		if z := strings.TrimSpace(r.ZipCode); z != "" {
			s := j.zip(z)
			s.Requests++
			if !closed {
				s.Open++
			}
			if enforce {
				s.Enforcement++
				if !closed {
					s.OpenEnforcement++
				}
			}
		}
	}
	return j
}

func aggregateVacancies(chunk []ingest.VacantBuilding) Joined {
	j := newJoined()
	for _, b := range chunk {
		if name := key(b.Neighborhood); name != "" {
			j.hood(name).Vacant++
		}
	}
	return j
}

// merge adds partial p into j. Only the merging goroutine calls it.
func (j Joined) merge(p Joined) {
	for name, pn := range p.Neighborhoods {
		n := j.hood(name)
		n.Requests += pn.Requests
		n.Open += pn.Open
		n.Closed += pn.Closed
		n.Vacant += pn.Vacant
		for c, v := range pn.ByCategory {
			n.ByCategory[c] += v
		}
		for z, v := range pn.zipCounts {
			n.zipCounts[z] += v
		}
	}
	for z, pz := range p.Zips {
		s := j.zip(z)
		s.Requests += pz.Requests
		s.Open += pz.Open
		s.Enforcement += pz.Enforcement
		s.OpenEnforcement += pz.OpenEnforcement
	}
}

// finalize picks each neighborhood's dominant ZIP and attributes its
// vacancies to that ZIP.
func (j Joined) finalize() {
	for _, n := range j.Neighborhoods {
		best, bestN := "", 0
		for z, c := range n.zipCounts {
			if c > bestN || (c == bestN && z < best) {
				best, bestN = z, c
			}
		}
		n.Zip = best
		if best != "" {
			j.zip(best).Vacant += n.Vacant
		}
	}
}
