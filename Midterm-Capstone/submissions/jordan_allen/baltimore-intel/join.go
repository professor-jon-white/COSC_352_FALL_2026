package main

import "sort"

// JoinedRow is one neighborhood that appears in BOTH datasets.
type JoinedRow struct {
	Name             string
	Crime            *CrimeStats
	Req              *RequestStats
	CrimeQuartile    int // 1 = lowest crime volume ... 4 = highest
	RequestQuartile  int // 1 = fewest 311 requests ... 4 = most
	DisorderQuartile int // 1 = fewest physical-disorder requests ... 4 = most
}

// Joined is the read-only result of the join. After joinDatasets returns, the
// analysis goroutines only read it, so they can share it without locks.
type Joined struct {
	Rows        []*JoinedRow // sorted by neighborhood name
	CrimeOnly   []string     // neighborhoods seen only in the crime data
	RequestOnly []string     // neighborhoods seen only in the 311 data
	Crime       *CrimeAgg    // city-wide crime aggregates (hourly profile etc.)
	Req         *RequestAgg  // city-wide 311 aggregates
}

// joinDatasets performs an inner join of the two per-neighborhood aggregates on
// the normalised neighborhood name and labels every joined neighborhood with its
// quartile for crime volume, 311 volume and physical-disorder volume.
func joinDatasets(c *CrimeAgg, r *RequestAgg) *Joined {
	j := &Joined{Crime: c, Req: r}
	for name, cs := range c.byNeigh {
		if rs, ok := r.byNeigh[name]; ok {
			j.Rows = append(j.Rows, &JoinedRow{Name: name, Crime: cs, Req: rs})
		} else {
			j.CrimeOnly = append(j.CrimeOnly, name)
		}
	}
	for name := range r.byNeigh {
		if _, ok := c.byNeigh[name]; !ok {
			j.RequestOnly = append(j.RequestOnly, name)
		}
	}
	sort.Strings(j.CrimeOnly)
	sort.Strings(j.RequestOnly)
	sort.Slice(j.Rows, func(a, b int) bool { return j.Rows[a].Name < j.Rows[b].Name })

	n := len(j.Rows)
	cq := quartiles(n, func(i int) float64 { return float64(j.Rows[i].Crime.Total) })
	rq := quartiles(n, func(i int) float64 { return float64(j.Rows[i].Req.Total) })
	dq := quartiles(n, func(i int) float64 { return float64(j.Rows[i].Req.Disorder) })
	for i, row := range j.Rows {
		row.CrimeQuartile = cq[i]
		row.RequestQuartile = rq[i]
		row.DisorderQuartile = dq[i]
	}
	return j
}

// quartiles assigns each of n items to a quartile 1..4 by ascending key value.
// Ties are broken by index (stable), so results are deterministic.
func quartiles(n int, key func(i int) float64) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return key(idx[a]) < key(idx[b]) })
	q := make([]int, n)
	for pos, i := range idx {
		q[i] = pos*4/n + 1
	}
	return q
}
