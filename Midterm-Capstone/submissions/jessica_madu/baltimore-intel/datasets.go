package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// RowParser turns one raw CSV record into an Event. It is created once per
// file (after reading the header) and is read-only afterwards, so it is safe
// to share between goroutines.
type RowParser func(rec []string) (Event, bool)

// Dataset is the STRATEGY interface: each public dataset knows how to bind to
// its own CSV header and how to interpret its rows. The pipeline never needs
// to know which dataset it is processing.
type Dataset interface {
	Name() string
	Bind(header []string) (RowParser, error)
}

// spec is a table-driven Dataset. Column names are given as candidate lists
// because Open Baltimore has renamed columns across dataset versions.
type spec struct {
	name                    string
	nbhd, cat, when, status []string
	classify                func(cat, status string) (open, violent bool)
}

// FACTORY functions: callers get a ready-to-use Dataset without knowing the
// column details.
func NewCrimeDataset() Dataset {
	violentWords := []string{"HOMICIDE", "SHOOTING", "ROBBERY", "AGG. ASSAULT", "AGG ASSAULT", "RAPE"}
	return spec{
		name:   "Part 1 Crime Data",
		nbhd:   []string{"neighborhood"},
		cat:    []string{"description", "crimedescription", "offense"},
		when:   []string{"crimedatetime", "crimedate", "datetime", "date"},
		status: nil,
		classify: func(cat, _ string) (bool, bool) {
			for _, w := range violentWords {
				if strings.Contains(cat, w) {
					return false, true
				}
			}
			return false, false
		},
	}
}

func New311Dataset() Dataset {
	return spec{
		name:   "311 Customer Service Requests",
		nbhd:   []string{"neighborhood"},
		cat:    []string{"srtype", "requesttype", "servicerequesttype", "type"},
		when:   []string{"createddate", "createdate", "created", "datecreated", "opened"},
		status: []string{"srstatus", "status"},
		classify: func(_, status string) (bool, bool) {
			for _, w := range []string{"OPEN", "NEW", "PROGRESS", "PENDING"} {
				if strings.Contains(status, w) {
					return true, false
				}
			}
			return false, false
		},
	}
}

func (s spec) Name() string { return s.name }

func (s spec) Bind(header []string) (RowParser, error) {
	n, c, w := find(header, s.nbhd), find(header, s.cat), find(header, s.when)
	if n < 0 || c < 0 || w < 0 {
		return nil, fmt.Errorf("%s: could not find required columns (neighborhood=%v category=%v date=%v) in header %v",
			s.name, n >= 0, c >= 0, w >= 0, header)
	}
	st := find(header, s.status) // optional
	need := max(n, max(c, w))
	return func(rec []string) (Event, bool) {
		if len(rec) <= need {
			return Event{}, false
		}
		nb := strings.ToUpper(strings.TrimSpace(rec[n]))
		if nb == "" {
			return Event{}, false // cannot join a row with no neighborhood
		}
		cat := strings.ToUpper(strings.TrimSpace(rec[c]))
		if cat == "" {
			cat = "UNKNOWN"
		}
		status := ""
		if st >= 0 && st < len(rec) {
			status = strings.ToUpper(strings.TrimSpace(rec[st]))
		}
		t, hasTime, ok := parseTime(strings.TrimSpace(rec[w]))
		ev := Event{Neighborhood: nb, Category: cat, When: t, HasDate: ok, HasTime: ok && hasTime}
		ev.Open, ev.Violent = s.classify(cat, status)
		return ev, true
	}, nil
}

func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimPrefix(s, "\uFEFF") {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func find(header, cands []string) int {
	for _, c := range cands {
		for i, h := range header {
			if norm(h) == c {
				return i
			}
		}
	}
	return -1
}

var layouts = []struct {
	layout  string
	hasTime bool
}{
	{"2006-01-02T15:04:05.000", true},
	{"2006-01-02T15:04:05Z07:00", true},
	{"2006-01-02T15:04:05", true},
	{"2006-01-02 15:04:05-07", true},
	{"2006-01-02 15:04:05", true},
	{"2006/01/02 15:04:05-07", true},
	{"2006/01/02 15:04:05", true},
	{"1/2/2006 3:04:05 PM", true},
	{"1/2/2006 15:04", true},
	{"1/2/2006", false},
	{"2006-01-02", false},
	{"2006/01/02", false},
}

func parseTime(s string) (t time.Time, hasTime, ok bool) {
	if s == "" {
		return
	}
	for _, l := range layouts {
		if tt, err := time.Parse(l.layout, s); err == nil {
			return tt, l.hasTime, true
		}
	}
	return
}
