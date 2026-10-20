package join

import (
	"strings"

	"baltimore-intel/internal/model"
)

type Category int

const (
	CatOther Category = iota
	CatSanitation
	CatLighting
	CatTransport
	CatExcluded // information calls and proactive city work
)

// Classify maps a 311 SRType to a category. Order matters: lighting is
// checked before the TRM-/TRT-/TRS- prefixes so street lights aren't
// counted as general transportation.
func Classify(t string) Category {
	switch {
	case strings.HasPrefix(t, "ECC-"), strings.Contains(t, "Rat Rubout Proactive"):
		return CatExcluded
	case strings.Contains(t, "StLight"), strings.Contains(t, "Street Light"):
		return CatLighting
	case strings.Contains(t, "Dirty Street"), strings.Contains(t, "Dirty Alley"),
		strings.Contains(t, "Illegal Dumping"), strings.Contains(t, "Sanitation Property"),
		strings.Contains(t, "Graffiti"), strings.Contains(t, "SW-Cleaning"),
		strings.Contains(t, "Vacant Building"):
		return CatSanitation
	case strings.HasPrefix(t, "TRT-"), strings.HasPrefix(t, "TRM-"), strings.HasPrefix(t, "TRS-"):
		return CatTransport
	}
	return CatOther
}

// IsViolent flags violent Part 1 crimes by description keywords.
func IsViolent(desc string) bool {
	d := strings.ToUpper(desc)
	return strings.Contains(d, "HOMICIDE") || strings.Contains(d, "SHOOTING") ||
		strings.Contains(d, "RAPE") || strings.Contains(d, "ROBBERY") ||
		strings.Contains(d, "AGG")
}

// Stats is the joined, per-neighborhood view of both datasets.
type Stats struct {
	Crime, Violent                  int
	Calls311                        int
	Sanitation, Lighting, Transport int
	CrimeByMonth, CallsByMonth      [12]int
}

// add accumulates another Stats into s (used when merging worker results).
func (s *Stats) add(o *Stats) {
	s.Crime += o.Crime
	s.Violent += o.Violent
	s.Calls311 += o.Calls311
	s.Sanitation += o.Sanitation
	s.Lighting += o.Lighting
	s.Transport += o.Transport
	for i := range s.CrimeByMonth {
		s.CrimeByMonth[i] += o.CrimeByMonth[i]
		s.CallsByMonth[i] += o.CallsByMonth[i]
	}
}

type Dataset struct {
	ByHood        map[string]*Stats
	Excluded      int // 311 rows skipped as non-complaints
	CrimeByHour   [24]int
	LightingHour  [24]int
	TransportHour [24]int
}

func newDataset() *Dataset {
	return &Dataset{ByHood: make(map[string]*Stats, 300)}
}

// stats returns the entry for a neighborhood, creating it if needed.
func (d *Dataset) stats(k string) *Stats {
	s, ok := d.ByHood[k]
	if !ok {
		s = &Stats{}
		d.ByHood[k] = s
	}
	return s
}

func (d *Dataset) addCrime(c model.CrimeRecord) {
	s := d.stats(c.Neighborhood)
	s.Crime++
	s.CrimeByMonth[c.When.Month()-1]++
	d.CrimeByHour[c.When.Hour()]++
	if IsViolent(c.Description) {
		s.Violent++
	}
}

func (d *Dataset) addRequest(r model.ServiceRequest) {
	cat := Classify(r.Type)
	if cat == CatExcluded {
		d.Excluded++
		return
	}
	s := d.stats(r.Neighborhood)
	s.Calls311++
	s.CallsByMonth[r.Created.Month()-1]++
	switch cat {
	case CatSanitation:
		s.Sanitation++
	case CatLighting:
		s.Lighting++
		d.LightingHour[r.Created.Hour()]++
	case CatTransport:
		s.Transport++
		d.TransportHour[r.Created.Hour()]++
	}
}

// Build joins crime and 311 on the normalized neighborhood key,
// single-threaded. It is the reference implementation that BuildParallel
// is verified against.
func Build(crimes []model.CrimeRecord, reqs []model.ServiceRequest) *Dataset {
	d := newDataset()
	for _, c := range crimes {
		d.addCrime(c)
	}
	for _, r := range reqs {
		d.addRequest(r)
	}
	return d
}
