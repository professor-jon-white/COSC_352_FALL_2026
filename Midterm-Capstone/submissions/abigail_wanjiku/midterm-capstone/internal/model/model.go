// Package model defines the record and key types shared by every pipeline
// stage, so stages depend on model rather than on each other.
package model

import "time"

// Key identifies one join bucket: a neighborhood in a given month.
type Key struct {
	Neighborhood string    // normalized: trimmed, uppercased (e.g. "BELAIR-EDISON")
	Month        time.Time // midnight on the first day of the month, Baltimore time
}

// Crime is one row of data/crime.csv, reduced to the fields the pipeline uses.
type Crime struct {
	OccurredAt   time.Time // CrimeDateTime, converted from UTC to Baltimore time
	Neighborhood string    // normalized
	CrimeType    string    // Description, e.g. "COMMON ASSAULT"
	Outdoor      bool      // Inside_Outside == "O"
}

// ServiceRequest is one row of data/requests_311.csv, reduced to the fields
// the pipeline uses.
type ServiceRequest struct {
	CreatedAt    time.Time // CreatedDate, Baltimore time
	Neighborhood string    // normalized
	RequestType  string    // SRType, e.g. "HCD-Illegal Dumping"
	Status       string    // SRStatus, e.g. "Closed (Transferred)"
	ClosedAt     time.Time // CloseDate; zero if the request is not closed
}

// Summary is the combined record for one Key: what happened in a neighborhood
// during a month, from both datasets. A Summary whose Key.Month is zero covers
// every month (see aggregate.ByNeighborhood).
type Summary struct {
	Key Key

	CrimeCount         int
	CrimesByType       map[string]int // Crime.CrimeType -> count
	NightOutdoorCrimes int            // outdoor crimes at night (see match.IsNight)

	RequestCount   int
	RequestsByType map[string]int // ServiceRequest.RequestType -> count

	// Time to close, over requests with a usable close time. Totals are kept
	// instead of averages so summaries can be merged exactly.
	ClosedCount     int
	CloseDaysTotal  float64
	ClosedByType    map[string]int     // RequestType -> closed requests
	CloseDaysByType map[string]float64 // RequestType -> total days to close
	BadCloseTimes   int                // closed requests excluded because ClosedAt is well before CreatedAt
}

// NewSummary returns an empty Summary for k with its maps allocated.
func NewSummary(k Key) *Summary {
	return &Summary{
		Key:             k,
		CrimesByType:    map[string]int{},
		RequestsByType:  map[string]int{},
		ClosedByType:    map[string]int{},
		CloseDaysByType: map[string]float64{},
	}
}

// AvgDaysToClose returns the mean days from creation to close, and false if
// no request in the summary has a usable close time.
func (s *Summary) AvgDaysToClose() (float64, bool) {
	if s.ClosedCount == 0 {
		return 0, false
	}
	return s.CloseDaysTotal / float64(s.ClosedCount), true
}

// Merge adds o's counts into s. Keys are not compared.
func (s *Summary) Merge(o *Summary) {
	s.CrimeCount += o.CrimeCount
	for t, n := range o.CrimesByType {
		s.CrimesByType[t] += n
	}
	s.NightOutdoorCrimes += o.NightOutdoorCrimes
	s.RequestCount += o.RequestCount
	for t, n := range o.RequestsByType {
		s.RequestsByType[t] += n
	}
	s.ClosedCount += o.ClosedCount
	s.CloseDaysTotal += o.CloseDaysTotal
	for t, n := range o.ClosedByType {
		s.ClosedByType[t] += n
	}
	for t, d := range o.CloseDaysByType {
		s.CloseDaysByType[t] += d
	}
	s.BadCloseTimes += o.BadCloseTimes
}

// KeyFor builds the join key for an already-normalized neighborhood and a
// timestamp, bucketing by month in the timestamp's own location.
func KeyFor(neighborhood string, t time.Time) Key {
	return Key{
		Neighborhood: neighborhood,
		Month:        time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()),
	}
}
