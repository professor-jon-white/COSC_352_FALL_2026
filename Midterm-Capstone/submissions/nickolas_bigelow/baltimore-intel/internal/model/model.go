// Package model defines the compact in-memory records shared by every stage.
//
// Memory design: one Baltimore year is ~50k crimes and several hundred
// thousand 311 requests. Records are therefore stored as flat value slices
// (no per-record pointers, so the GC scans almost nothing), coordinates are
// float32 (~1 m precision at this latitude), timestamps are int64 Unix
// seconds instead of time.Time (24 bytes with a *Location pointer), and every
// repeated string (neighborhood, crime type, 311 type, agency) is interned to
// a uint16 ID. A Crime is 24 bytes; a ServiceRequest is 40 bytes.
package model

import (
	"strings"
	"sync"
	"unicode"
)

// ID is an interned string handle. 0 always means "unknown / blank".
type ID = uint16

// Interner maps strings to small integer IDs. It is safe for concurrent use:
// lookups of already-seen strings take only a read lock, which is the common
// case once the first few thousand rows have been parsed.
type Interner struct {
	mu        sync.RWMutex
	ids       map[string]ID
	names     []string
	fromCaps  []bool // display name was derived from an ALL-CAPS source
	normalize func(string) string
}

func NewInterner(normalize func(string) string) *Interner {
	if normalize == nil {
		normalize = strings.TrimSpace
	}
	return &Interner{ids: map[string]ID{"": 0}, names: []string{""}, fromCaps: []bool{false}, normalize: normalize}
}

// ID returns the handle for raw, adding it if new.
func (in *Interner) ID(raw string) ID {
	key := in.normalize(raw)
	if key == "" {
		return 0
	}
	in.mu.RLock()
	id, ok := in.ids[key]
	in.mu.RUnlock()
	if ok {
		in.maybeImproveDisplay(id, raw)
		return id
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	if id, ok = in.ids[key]; ok { // another goroutine won the race
		return id
	}
	if len(in.names) >= 1<<16-1 {
		return 0 // table full: degrade to "unknown" rather than overflow
	}
	id = ID(len(in.names))
	in.ids[key] = id
	in.names = append(in.names, displayName(raw))
	in.fromCaps = append(in.fromCaps, isAllUpper(raw))
	return id
}

// Lookup returns the ID for raw without inserting it.
func (in *Interner) Lookup(raw string) (ID, bool) {
	in.mu.RLock()
	defer in.mu.RUnlock()
	id, ok := in.ids[in.normalize(raw)]
	return id, ok && id != 0
}

// Name returns the display name for id.
func (in *Interner) Name(id ID) string {
	in.mu.RLock()
	defer in.mu.RUnlock()
	if int(id) < len(in.names) {
		return in.names[id]
	}
	return ""
}

// Len is the number of IDs including the reserved 0.
func (in *Interner) Len() int {
	in.mu.RLock()
	defer in.mu.RUnlock()
	return len(in.names)
}

// Some datasets publish neighborhoods in ALL CAPS and others in Title Case.
// Prefer a mixed-case spelling for display when one shows up.
func (in *Interner) maybeImproveDisplay(id ID, raw string) {
	in.mu.RLock()
	caps := in.fromCaps[id]
	in.mu.RUnlock()
	if !caps || isAllUpper(raw) {
		return
	}
	in.mu.Lock()
	if in.fromCaps[id] {
		in.names[id] = displayName(raw)
		in.fromCaps[id] = false
	}
	in.mu.Unlock()
}

func isAllUpper(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsLetter(r) {
			hasLetter = true
		}
	}
	return hasLetter
}

func displayName(raw string) string {
	s := strings.Join(strings.Fields(raw), " ")
	if !isAllUpper(s) {
		return s
	}
	// Title-case ALL-CAPS names: "BELAIR-EDISON" -> "Belair-Edison"
	b := []rune(strings.ToLower(s))
	up := true
	for i, r := range b {
		if up && unicode.IsLetter(r) {
			b[i] = unicode.ToUpper(r)
		}
		up = r == ' ' || r == '-' || r == '/' || r == '(' || r == '.'
	}
	return string(b)
}

// NormalizeNeighborhood is the cross-dataset join key: case, spacing and
// punctuation differences between datasets must not split a neighborhood.
func NormalizeNeighborhood(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" || s == "UNKNOWN" || s == "N/A" || s == "NULL" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		switch {
		case r == '\'' || r == '.' || r == ',':
			continue
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '/':
			space = true
		default:
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeLabel is used for categorical labels (crime type, agency, ...).
func NormalizeLabel(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

// ---- records ------------------------------------------------------------

// Crime is one Part 1 crime incident (24 bytes).
type Crime struct {
	Time      int64 // Unix seconds, UTC
	Lat, Lon  float32
	Hood      ID
	Desc      ID // interned Description, e.g. "AGG. ASSAULT"
	Incidents uint16
	Violent   bool
}

// ServiceRequest is one 311 request (40 bytes).
type ServiceRequest struct {
	Created, Closed, Due int64 // Unix seconds; 0 = missing
	Lat, Lon             float32
	Hood                 ID
	Type                 ID // interned SRType
	Agency               ID
	Sanitation           bool // trash / dumping / rodent / dirty-alley family
}

// LiquorOutlet is one licensed establishment (latest license per number).
type LiquorOutlet struct {
	Name, Address, Class, Kind string
	Zip                        string
	Lat, Lon                   float32
	Hood                       ID // assigned by spatial join (licenses carry no neighborhood)
	Year                       int
}

// VacantNotice is one open vacant building notice.
type VacantNotice struct {
	Date     int64
	Lat, Lon float32
	Hood     ID
	Typology ID // Housing Market Typology 2023
}

// Dataset is everything the analyses can see. After ingestion it is
// read-only, so the five analyses can share it across goroutines without locks.
type Dataset struct {
	Hoods      *Interner // shared neighborhood join key across all datasets
	CrimeTypes *Interner
	SRTypes    *Interner
	Agencies   *Interner
	Typologies *Interner
	Crimes     []Crime
	Requests   []ServiceRequest
	Outlets    []LiquorOutlet
	Vacants    []VacantNotice
	Year       int
	RadiusM    float64 // spatial-join radius for "near a liquor outlet"
}

func NewDataset() *Dataset {
	return &Dataset{
		Hoods:      NewInterner(NormalizeNeighborhood),
		CrimeTypes: NewInterner(NormalizeLabel),
		SRTypes:    NewInterner(NormalizeLabel),
		Agencies:   NewInterner(NormalizeLabel),
		Typologies: NewInterner(NormalizeLabel),
		RadiusM:    150,
	}
}

// InBaltimore rejects blank (0,0) and wildly mis-geocoded points.
func InBaltimore(lat, lon float64) bool {
	return lat > 39.19 && lat < 39.38 && lon > -76.72 && lon < -76.52
}
