package main

import "time"

// Event is the unified, dataset-independent record produced by parsing one CSV row.
// It is a small value type (passed by value, no pointers) so it stays on the
// worker's stack and creates no garbage per row.
type Event struct {
	Neighborhood string
	Category     string
	When         time.Time
	HasDate      bool
	HasTime      bool
	Open         bool // 311 only: request not yet closed
	Violent      bool // crime only: violent offense
}

// Agg is the aggregate we keep instead of the raw rows. Memory use is
// O(#neighborhoods x #categories), NOT O(#rows): millions of rows can be
// streamed through without ever being held in memory at once.
//
// Thread safety: every worker goroutine owns a private Agg (confinement), so
// no mutex is needed on the hot path. Aggs are merged once, after the workers
// finish, by a single goroutine.
type Agg struct {
	Total, Skipped int
	ByNbhd         map[string]int
	ByCat          map[string]int
	NbhdCat        map[string]map[string]int
	ByMonth        map[string]int
	ByHour         [24]int
	OpenByNbhd     map[string]int
	ViolentByNbhd  map[string]int
}

func NewAgg() *Agg {
	return &Agg{
		ByNbhd:        map[string]int{},
		ByCat:         map[string]int{},
		NbhdCat:       map[string]map[string]int{},
		ByMonth:       map[string]int{},
		OpenByNbhd:    map[string]int{},
		ViolentByNbhd: map[string]int{},
	}
}

func (a *Agg) Add(e Event) {
	a.Total++
	a.ByNbhd[e.Neighborhood]++
	a.ByCat[e.Category]++
	m := a.NbhdCat[e.Neighborhood]
	if m == nil {
		m = map[string]int{}
		a.NbhdCat[e.Neighborhood] = m
	}
	m[e.Category]++
	if e.HasDate {
		a.ByMonth[e.When.Format("2006-01")]++
	}
	if e.HasTime {
		a.ByHour[e.When.Hour()]++
	}
	if e.Open {
		a.OpenByNbhd[e.Neighborhood]++
	}
	if e.Violent {
		a.ViolentByNbhd[e.Neighborhood]++
	}
}

// Merge folds another Agg into a. Only called from one goroutine.
func (a *Agg) Merge(o *Agg) {
	a.Total += o.Total
	a.Skipped += o.Skipped
	addMap(a.ByNbhd, o.ByNbhd)
	addMap(a.ByCat, o.ByCat)
	addMap(a.ByMonth, o.ByMonth)
	addMap(a.OpenByNbhd, o.OpenByNbhd)
	addMap(a.ViolentByNbhd, o.ViolentByNbhd)
	for n, cats := range o.NbhdCat {
		m := a.NbhdCat[n]
		if m == nil {
			m = map[string]int{}
			a.NbhdCat[n] = m
		}
		addMap(m, cats)
	}
	for i := range a.ByHour {
		a.ByHour[i] += o.ByHour[i]
	}
}

func addMap(dst, src map[string]int) {
	for k, v := range src {
		dst[k] += v
	}
}
