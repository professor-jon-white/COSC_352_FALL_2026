package ingestion

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

// parseDate converts either ArcGIS epoch timestamps (milliseconds, or seconds)
// or GeoJSON date strings to Unix milliseconds. Unknown/null dates return 0.
func parseDate(properties map[string]json.RawMessage, keys ...string) int64 {
	for _, key := range keys {
		raw, ok := properties[key]
		if !ok || len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var asString string
		if err := json.Unmarshal(raw, &asString); err == nil {
			s := strings.TrimSpace(asString)
			if s == "" {
				continue
			}
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				if ms := numericTimestamp(v); ms != 0 {
					return ms
				}
			}
			for _, layout := range []string{
				time.RFC3339Nano,
				"2006-01-02T15:04:05.999999999",
				"2006-01-02T15:04:05",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02 15:04:05",
				"2006-01-02",
				"01/02/2006 03:04:05 PM",
				"01/02/2006 03:04 PM",
				"01/02/2006 15:04:05",
				"01/02/2006",
			} {
				if t, err := time.Parse(layout, s); err == nil {
					return t.UnixMilli()
				}
			}
			continue
		}
		var n float64
		if err := json.Unmarshal(raw, &n); err == nil {
			if ms := numericTimestamp(n); ms != 0 {
				return ms
			}
		}
	}
	return 0
}

func numericTimestamp(n float64) int64 {
	if n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0
	}
	// Seconds have ~10 digits for modern records; milliseconds ~13.
	if n < 100_000_000_000 {
		n *= 1000
	}
	if n > float64(math.MaxInt64) {
		return 0
	}
	return int64(math.Round(n))
}
