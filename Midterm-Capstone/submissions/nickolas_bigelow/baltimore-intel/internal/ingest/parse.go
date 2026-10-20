package ingest

import (
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed tz database so America/New_York works on any machine
)

// Eastern is Baltimore's local time zone, used for hour-of-day analysis.
var Eastern = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.UTC
	}
	return loc
}()

var timeLayouts = []string{
	"2006/01/02 15:04:05-07", // Open Baltimore CSV export: 2022/03/14 23:15:00+00
	"2006/01/02 15:04:05",
	"2006-01-02T15:04:05Z07:00", // ISO 8601
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"01/02/2006 03:04:05 PM",
	"1/2/2006 3:04:05 PM",
	"01/02/2006 15:04",
	"01/02/2006",
	"2006/01/02",
	"2006-01-02",
}

// ParseTime accepts every date format the portal has used, plus ArcGIS REST
// epoch milliseconds. Returns Unix seconds.
func ParseTime(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if c := s[0]; c >= '0' && c <= '9' && len(s) >= 10 && !strings.ContainsAny(s, "/-: ") {
		ms, err := strconv.ParseInt(strings.SplitN(s, ".", 2)[0], 10, 64)
		if err == nil {
			if ms > 1e11 { // milliseconds
				return ms / 1000, true
			}
			return ms, true
		}
	}
	for _, l := range timeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// ParseCoord parses a coordinate, returning ok=false for blanks/garbage.
func ParseCoord(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v, err == nil
}

// ParseGeoLocation reads "(39.29, -76.61)" style columns.
func ParseGeoLocation(s string) (lat, lon float64, ok bool) {
	s = strings.Trim(strings.TrimSpace(s), "()")
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lat, ok1 := ParseCoord(parts[0])
	lon, ok2 := ParseCoord(parts[1])
	return lat, lon, ok1 && ok2
}

// YearOf returns the calendar year of a Unix timestamp in Baltimore time.
func YearOf(unix int64) int { return time.Unix(unix, 0).In(Eastern).Year() }
