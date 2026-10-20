package main

import (
	"strconv"
	"strings"
	"time"
)

// TimeFilter optionally restricts both datasets to the same period so that
// they are comparable. From is inclusive, To is exclusive; zero means open.
type TimeFilter struct {
	From time.Time
	To   time.Time
}

// Allows reports whether t falls inside the filter window.
func (f TimeFilter) Allows(t time.Time) bool {
	if !f.From.IsZero() && t.Before(f.From) {
		return false
	}
	if !f.To.IsZero() && !t.Before(f.To) {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Dataset 1: Open Baltimore crime incidents (NIBRS Group A / Part 1)
// ---------------------------------------------------------------------------

// Indexes into the row produced for the crime dataset (order of crimeSpec).
const (
	crimeWhen = iota
	crimeDesc
	crimeNeigh
	crimeCount
)

// crimeSpec is the factory for the crime dataset schema. Header aliases cover
// the NIBRS, legacy Part 1 and third-party re-publications of the dataset.
func crimeSpec() SourceSpec {
	return SourceSpec{
		Name: "crime",
		Fields: []FieldSpec{
			{Name: "when", Required: true,
				Aliases: []string{"CrimeDateTime", "Crime_DateTime", "CrimeDate", "Crime Date", "DateTime", "Date"}},
			{Name: "description", Required: true,
				Aliases: []string{"Description", "Crime", "CrimeDescription", "Offense"}},
			{Name: "neighborhood", Required: true,
				Aliases: []string{"Neighborhood", "Neighbourhood"}},
			{Name: "count", Required: false,
				Aliases: []string{"Total_Incidents", "TotalIncidents", "Incidents"}},
		},
	}
}

// CrimeStats is everything we keep about the crime in one neighborhood.
type CrimeStats struct {
	Total   int
	Violent int
	ByCat   map[string]int
}

// CrimeAgg aggregates crime rows. It is NOT safe for concurrent use; every
// pipeline worker gets its own instance and the instances are merged at the end.
type CrimeAgg struct {
	filter  TimeFilter
	tp      timeParser
	violent []string
	byNeigh map[string]*CrimeStats
	byHour  [24]int
	stats   RowStats
}

// newCrimeAgg is the factory for crime aggregators.
func newCrimeAgg(f TimeFilter, loc *time.Location, violentKeywords []string) *CrimeAgg {
	return &CrimeAgg{
		filter:  f,
		tp:      timeParser{loc: loc},
		violent: violentKeywords,
		byNeigh: make(map[string]*CrimeStats),
	}
}

// AddRow implements Aggregator.
func (a *CrimeAgg) AddRow(v []string) {
	n := normalizeNeighborhood(v[crimeNeigh])
	if n == "" {
		a.stats.NoNeighborhood++
		return
	}
	t, ok := a.tp.parse(v[crimeWhen])
	if !ok {
		a.stats.BadDate++
		return
	}
	if !a.filter.Allows(t) {
		a.stats.OutOfRange++
		return
	}
	count := 1
	if c, err := strconv.ParseFloat(strings.TrimSpace(v[crimeCount]), 64); err == nil && c >= 1 {
		count = int(c)
	}
	cat := strings.ToUpper(strings.TrimSpace(v[crimeDesc]))
	if cat == "" {
		cat = "UNKNOWN"
	}
	cs := a.byNeigh[n]
	if cs == nil {
		cs = &CrimeStats{ByCat: make(map[string]int)}
		a.byNeigh[n] = cs
	}
	cs.Total += count
	if containsAny(cat, a.violent) {
		cs.Violent += count
	}
	cs.ByCat[cat] += count
	a.byHour[t.Hour()] += count
	a.stats.Accepted++
}

// Merge implements Aggregator. It must only be called after both aggregators
// have stopped receiving rows.
func (a *CrimeAgg) Merge(other Aggregator) {
	o := other.(*CrimeAgg)
	for name, ocs := range o.byNeigh {
		cs := a.byNeigh[name]
		if cs == nil {
			a.byNeigh[name] = ocs
			continue
		}
		cs.Total += ocs.Total
		cs.Violent += ocs.Violent
		for k, c := range ocs.ByCat {
			cs.ByCat[k] += c
		}
	}
	for h := range a.byHour {
		a.byHour[h] += o.byHour[h]
	}
	a.stats.merge(o.stats)
}

// Stats implements Aggregator.
func (a *CrimeAgg) Stats() RowStats { return a.stats }

// ---------------------------------------------------------------------------
// Dataset 2: Open Baltimore 311 customer service requests
// ---------------------------------------------------------------------------

// Indexes into the row produced for the 311 dataset (order of requestSpec).
const (
	reqCreated = iota
	reqType
	reqNeigh
	reqClosed
)

// requestSpec is the factory for the 311 dataset schema.
func requestSpec() SourceSpec {
	return SourceSpec{
		Name: "311",
		Fields: []FieldSpec{
			{Name: "created", Required: true,
				Aliases: []string{"CreatedDate", "Created_Date", "Created", "OpenDate", "Open Date"}},
			{Name: "type", Required: true,
				Aliases: []string{"SRType", "SR_Type", "ServiceRequestType", "RequestType", "Type"}},
			{Name: "neighborhood", Required: true,
				Aliases: []string{"Neighborhood", "Neighbourhood"}},
			{Name: "closed", Required: false,
				Aliases: []string{"CloseDate", "ClosedDate", "Close_Date", "DateClosed"}},
		},
	}
}

// RequestStats is everything we keep about the 311 requests of one neighborhood.
type RequestStats struct {
	Total     int
	Disorder  int                  // requests whose type matches the disorder keywords
	Closed    int                  // requests with a usable close date
	ByType    map[string]int       // request type -> count
	Durations map[string][]float32 // request type -> days from creation to close
}

// RequestAgg aggregates 311 rows; same ownership rules as CrimeAgg.
type RequestAgg struct {
	filter   TimeFilter
	tp       timeParser
	disorder []string
	byNeigh  map[string]*RequestStats
	byHour   [24]int
	stats    RowStats
}

// newRequestAgg is the factory for 311 aggregators.
func newRequestAgg(f TimeFilter, loc *time.Location, disorderKeywords []string) *RequestAgg {
	return &RequestAgg{
		filter:   f,
		tp:       timeParser{loc: loc},
		disorder: disorderKeywords,
		byNeigh:  make(map[string]*RequestStats),
	}
}

// AddRow implements Aggregator.
func (a *RequestAgg) AddRow(v []string) {
	n := normalizeNeighborhood(v[reqNeigh])
	if n == "" {
		a.stats.NoNeighborhood++
		return
	}
	created, ok := a.tp.parse(v[reqCreated])
	if !ok {
		a.stats.BadDate++
		return
	}
	if !a.filter.Allows(created) {
		a.stats.OutOfRange++
		return
	}
	typ := strings.ToUpper(strings.TrimSpace(v[reqType]))
	if typ == "" {
		typ = "UNKNOWN"
	}
	rs := a.byNeigh[n]
	if rs == nil {
		rs = &RequestStats{
			ByType:    make(map[string]int),
			Durations: make(map[string][]float32),
		}
		a.byNeigh[n] = rs
	}
	rs.Total++
	rs.ByType[typ]++
	if containsAny(typ, a.disorder) {
		rs.Disorder++
	}
	a.byHour[created.Hour()]++
	if closed, ok := a.tp.parse(v[reqClosed]); ok {
		days := closed.Sub(created).Hours() / 24
		if days >= 0 && days <= 3650 {
			rs.Closed++
			rs.Durations[typ] = append(rs.Durations[typ], float32(days))
		}
	}
	a.stats.Accepted++
}

// Merge implements Aggregator.
func (a *RequestAgg) Merge(other Aggregator) {
	o := other.(*RequestAgg)
	for name, ors := range o.byNeigh {
		rs := a.byNeigh[name]
		if rs == nil {
			a.byNeigh[name] = ors
			continue
		}
		rs.Total += ors.Total
		rs.Disorder += ors.Disorder
		rs.Closed += ors.Closed
		for k, c := range ors.ByType {
			rs.ByType[k] += c
		}
		for k, d := range ors.Durations {
			rs.Durations[k] = append(rs.Durations[k], d...)
		}
	}
	for h := range a.byHour {
		a.byHour[h] += o.byHour[h]
	}
	a.stats.merge(o.stats)
}

// Stats implements Aggregator.
func (a *RequestAgg) Stats() RowStats { return a.stats }
