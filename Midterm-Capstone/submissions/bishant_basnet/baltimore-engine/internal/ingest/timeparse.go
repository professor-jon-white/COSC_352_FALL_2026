package ingest

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	_ "time/tzdata" // embed tz database so the binary works on any Mac
)

// Baltimore is the local time zone all timestamps are converted into.
var Baltimore = loadLoc("America/New_York")

func loadLoc(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return l
}

// Layouts seen in Open Baltimore / ArcGIS CSV exports.
var layouts = []string{
	"2006/01/02 15:04:05-07",
	"2006/01/02 15:04:05-07:00",
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006/01/02",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05-07",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"1/2/2006 3:04:05 PM",
	"1/2/2006 3:04 PM",
	"1/2/2006 15:04:05",
	"1/2/2006 15:04",
	"1/2/2006",
}

// lastLayout caches the index of the last layout that worked. A file almost
// always uses one format, so this turns ~15 failed parses per row into 1
// successful one. It is atomic because many workers call ParseTime at once.
var lastLayout atomic.Int32

// ParseTime parses a timestamp; values without a zone are taken as Baltimore time.
func ParseTime(s string) (time.Time, bool) { return ParseTimeIn(s, Baltimore) }

// ParseTimeIn parses a timestamp; values without an explicit zone are
// interpreted in `assume` (e.g. UTC for the NIBRS crime export, which the
// portal documents as downloading in UTC). The result is in Baltimore time.
func ParseTimeIn(s string, assume *time.Location) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if len(s) >= 10 && isDigits(s) { // epoch seconds / milliseconds
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			if len(s) >= 13 {
				return time.UnixMilli(n).In(Baltimore), true
			}
			return time.Unix(n, 0).In(Baltimore), true
		}
	}
	hint := int(lastLayout.Load())
	if t, err := time.ParseInLocation(layouts[hint], s, assume); err == nil {
		return t.In(Baltimore), true
	}
	for i, l := range layouts {
		if i == hint {
			continue
		}
		if t, err := time.ParseInLocation(l, s, assume); err == nil {
			lastLayout.Store(int32(i))
			return t.In(Baltimore), true
		}
	}
	return time.Time{}, false
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
