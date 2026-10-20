// Package model holds the plain data types shared by every pipeline stage.
//
// Raw records (Crime, Request, License) are deliberately compact: only the
// fields the analyses need are kept, and repeated strings are interned by the
// ingest stage, so hundreds of thousands of rows stay cheap on the heap.
package model

import (
	"sort"
	"time"
)

// Night is defined in Eastern local time as 21:00 through 03:59.
const (
	NightStart = 21
	NightEnd   = 4
)

// IsNight reports whether a local hour (0-23) falls in the night window.
// A negative hour means "time of day unknown" and is never night.
func IsNight(hour int) bool {
	if hour < 0 {
		return false
	}
	return hour >= NightStart || hour < NightEnd
}

// Crime is one NIBRS Group A incident.
type Crime struct {
	When      time.Time // UTC, as published
	Offense   string    // upper-case offense description
	Violent   bool      // homicide, rape, robbery, aggravated assault
	Shooting  bool
	LocalHour int    // 0-23 in America/New_York, -1 if the time is unknown
	Hood      string // canonical neighborhood, "" if blank in the source
	Block     string // normalised street-block key, "" if no usable address
}

// Request is one 311 customer service request.
type Request struct {
	Created   time.Time
	Closed    time.Time // zero if there is no close date
	Type      string    // e.g. "SW-Dirty Alley"
	Prefix    string    // e.g. "SW"
	Hood      string
	Block     string
	Completed bool // status "Closed" with a valid close date
}

// License is one liquor-license row (one licensee in one license year).
type License struct {
	Key    string
	Start  time.Time
	End    time.Time
	Kind   string // establishment type, e.g. "TAVERN"
	Active bool   // status is not "Closed"
	Block  string
}

// HoodAgg is everything we know about one neighborhood after the join.
type HoodAgg struct {
	Crimes       int
	Violent      int
	TimedViolent int // violent crimes with a known time of day
	NightViolent int
	Shootings    int

	Requests   int
	Open       int // requests with no close date
	ReqByType  map[string]int
	CloseHours []float64 // hours from created to closed, completed requests only

	Licenses  int
	LicByKind map[string]int
}

func newHood() *HoodAgg {
	return &HoodAgg{ReqByType: make(map[string]int), LicByKind: make(map[string]int)}
}

func (h *HoodAgg) merge(o *HoodAgg) {
	h.Crimes += o.Crimes
	h.Violent += o.Violent
	h.TimedViolent += o.TimedViolent
	h.NightViolent += o.NightViolent
	h.Shootings += o.Shootings
	h.Requests += o.Requests
	h.Open += o.Open
	h.Licenses += o.Licenses
	for k, v := range o.ReqByType {
		h.ReqByType[k] += v
	}
	for k, v := range o.LicByKind {
		h.LicByKind[k] += v
	}
	h.CloseHours = append(h.CloseHours, o.CloseHours...)
}

// BlockAgg is everything we know about one street block after the join.
type BlockAgg struct {
	Crimes       int
	Violent      int
	NightViolent int
	Requests     int
	Kinds        []string // one entry per liquor license on the block
}

func (b *BlockAgg) merge(o *BlockAgg) {
	b.Crimes += o.Crimes
	b.Violent += o.Violent
	b.NightViolent += o.NightViolent
	b.Requests += o.Requests
	b.Kinds = append(b.Kinds, o.Kinds...)
}

// Coverage tracks how many rows of a dataset made it into the join.
type Coverage struct {
	Rows    int // rows inside the analysis window
	Located int // rows that could be assigned to a neighborhood
	Ignored int // rows deliberately skipped (e.g. 311 call-centre info requests)
}

// Quality summarises join coverage for the report.
type Quality struct {
	Crime, Requests, Licenses Coverage
}

func (c *Coverage) add(o Coverage) {
	c.Rows += o.Rows
	c.Located += o.Located
	c.Ignored += o.Ignored
}

// Acc is a mutable accumulator. Every worker goroutine owns a private Acc, so
// no locking is needed; the per-worker results are merged once at the end.
type Acc struct {
	Hoods  map[string]*HoodAgg
	Blocks map[string]*BlockAgg
	Q      Quality
}

// NewAcc returns an empty accumulator.
func NewAcc() *Acc {
	return &Acc{Hoods: make(map[string]*HoodAgg), Blocks: make(map[string]*BlockAgg)}
}

// Hood returns the aggregate for a neighborhood, creating it on first use.
func (a *Acc) Hood(name string) *HoodAgg {
	h, ok := a.Hoods[name]
	if !ok {
		h = newHood()
		a.Hoods[name] = h
	}
	return h
}

// Block returns the aggregate for a street block, creating it on first use.
func (a *Acc) Block(key string) *BlockAgg {
	b, ok := a.Blocks[key]
	if !ok {
		b = &BlockAgg{}
		a.Blocks[key] = b
	}
	return b
}

// Merge folds another accumulator into this one.
func (a *Acc) Merge(o *Acc) {
	for name, oh := range o.Hoods {
		a.Hood(name).merge(oh)
	}
	for key, ob := range o.Blocks {
		a.Block(key).merge(ob)
	}
	a.Q.Crime.add(o.Q.Crime)
	a.Q.Requests.add(o.Q.Requests)
	a.Q.Licenses.add(o.Q.Licenses)
}

// Dataset is the finished, read-only result of the join. Analyses may read it
// from many goroutines at once; nothing mutates it after the pipeline returns.
type Dataset struct {
	From, To    time.Time
	Hoods       map[string]*HoodAgg
	Blocks      map[string]*BlockAgg
	Quality     Quality
	IndexBlocks int // street blocks known to the block->neighborhood index
}

// HoodNames returns neighborhood names in sorted order (deterministic output).
func (d *Dataset) HoodNames() []string {
	names := make([]string, 0, len(d.Hoods))
	for n := range d.Hoods {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
